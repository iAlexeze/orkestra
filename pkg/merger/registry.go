// pkg/merger/registry.go
package merger

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	orasauth "oras.land/oras-go/v2/registry/remote/auth"
)

// registryKatalogPath is the root directory inside the registry repo where katalogs live.
// registry/katalogs/<name>/katalog.yaml
const (
	registryKatalogPath = "registry/katalogs"
	registryFileName    = "katalog.yaml"
	orasPullTimeout     = 2 * time.Minute
)

// optionalPatternFiles lists files fetched when present but not required.
// motif.yaml may be bundled with a pattern when the pattern exposes a reusable Motif.
var optionalPatternFiles = []string{"motif.yaml"}

// ── Current registry loading (v2 protocol) ───────────────────────────────────

// loadRegistrySource loads a single registry pattern entry.
//
// Resolution sequence:
//  1. Parse url@version shorthand or url + version fields
//  2. Determine pull method: OCI or Git
//  3. Pull pattern to a temp directory
//  4. Validate all 5 required files exist and are non-empty (fail fast)
//  5. Load katalog.yaml or komposer.yaml based on UseKomposer
//  6. Parse and return the CRD entries
func (m *Merger) loadRegistrySource(src orktypes.RegistrySource) (map[string]orktypes.CRDEntry, error) {
	cleanURL, version := src.ResolvedURL()

	logger.Info().
		Str("url", cleanURL).
		Str("version", version).
		Bool("oci", src.OCI).
		Str("loads", src.SourceFile()).
		Msg("merger: pulling registry source")

	// Resolve auth credentials from environment variables
	auth, err := resolveRegistryAuth(src.Auth)
	if err != nil {
		return nil, fmt.Errorf("registry %q: auth: %w", cleanURL, err)
	}

	// Pull the pattern to a temp directory
	tmpDir, cleanup, err := m.pullPattern(cleanURL, version, src.OCI, auth)
	if err != nil {
		return nil, fmt.Errorf("registry %q@%s: pull failed: %w", cleanURL, version, err)
	}
	defer cleanup()

	// Validate the five required files — fail fast on any violation
	if err := validatePatternStructure(tmpDir, cleanURL, version); err != nil {
		return nil, err
	}

	// Load the source file: katalog.yaml or komposer.yaml
	sourceFile := filepath.Join(tmpDir, src.SourceFile())
	sourcePath := fmt.Sprintf("registry:%s@%s/%s", cleanURL, version, src.SourceFile())

	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("registry %q@%s: reading %s: %w",
			cleanURL, version, src.SourceFile(), err)
	}

	doc, err := parseKatalogDoc(data, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("registry %q@%s: parsing %s: %w",
			cleanURL, version, src.SourceFile(), err)
	}
	if doc == nil {
		return nil, fmt.Errorf(
			"registry %q@%s: %s is not a valid Katalog or Komposer document",
			cleanURL, version, src.SourceFile(),
		)
	}

	switch doc.Kind {
	case konfig.KatalogKind():
		if src.UseKomposer {
			return nil, fmt.Errorf(
				"registry %q@%s: useKomposer is true but komposer.yaml contains kind %q — "+
					"check the upstream pattern's komposer.yaml",
				cleanURL, version, doc.Kind,
			)
		}
		return m.loadKatalog(sourcePath, doc)

	case konfig.KomposerKind():
		if !src.UseKomposer {
			return nil, fmt.Errorf(
				"registry %q@%s: useKomposer is false but katalog.yaml contains kind %q — "+
					"set useKomposer: true to load the upstream Komposer, or check the pattern structure",
				cleanURL, version, doc.Kind,
			)
		}
		return m.loadKomposer(sourcePath, doc)

	default:
		return nil, fmt.Errorf(
			"registry %q@%s: %s has unexpected kind %q — expected %q or %q",
			cleanURL, version, src.SourceFile(), doc.Kind,
			konfig.KatalogKind(), konfig.KomposerKind(),
		)
	}
}

// pullPattern fetches a registry pattern to a temp directory.
// Returns the temp dir path and a cleanup function.
func (m *Merger) pullPattern(
	url, version string,
	oci bool,
	auth *utils.FileAuth,
) (tmpDir string, cleanup func(), err error) {
	tmpDir, err = os.MkdirTemp("", "orkestra-registry-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir: %w", err)
	}

	cleanup = func() { os.RemoveAll(tmpDir) }

	if oci {
		err = m.pullOCIPattern(url, version, tmpDir, auth)
	} else {
		err = m.pullGitPattern(url, version, tmpDir, auth)
	}

	if err != nil {
		cleanup()
		return "", nil, err
	}

	return tmpDir, cleanup, nil
}

// ── OCI pull ──────────────────────────────────────────────────────────────────

// pullOCIPattern fetches an OCI artifact and extracts it to tmpDir using ORAS Go library.
func (m *Merger) pullOCIPattern(url, version, tmpDir string, auth *utils.FileAuth) error {
	ref := strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
	ref = strings.TrimSuffix(ref, "/")
	ref = fmt.Sprintf("%s:%s", ref, version)

	logger.Debug().
		Str("ref", ref).
		Str("dst", tmpDir).
		Msg("registry: pulling OCI artifact with ORAS Go library")

	if err := orasPull(ref, tmpDir, auth); err != nil {
		return fmt.Errorf("OCI pull %q: %w", ref, err)
	}

	logger.Debug().
		Str("ref", ref).
		Str("dst", tmpDir).
		Msg("registry: OCI artifact pulled successfully")

	return nil
}

// orasPull pulls an OCI artifact using the ORAS Go library into dst.
func orasPull(ref, dst string, auth *utils.FileAuth) error {
	ctx, cancel := context.WithTimeout(context.Background(), orasPullTimeout)
	defer cancel()

	repoName, reference, ok := strings.Cut(ref, ":")
	if !ok || reference == "" {
		return fmt.Errorf("invalid OCI reference %q: missing tag or digest", ref)
	}

	repo, err := remote.NewRepository(repoName)
	if err != nil {
		return fmt.Errorf("creating repository for %q: %w", repoName, err)
	}

	if auth != nil {
		repo.Client = &orasauth.Client{
			ClientID: "orkestra",
			Credential: func(ctx context.Context, registry string) (orasauth.Credential, error) {
				switch strings.ToLower(auth.Type) {
				case "basic":
					if auth.Username != "" && auth.Password != "" {
						return orasauth.Credential{
							Username: auth.Username,
							Password: auth.Password,
						}, nil
					}
				case "bearer", "github":
					if auth.BearerToken != "" {
						return orasauth.Credential{
							RefreshToken: auth.BearerToken,
						}, nil
					}
				}
				return orasauth.EmptyCredential, nil
			},
		}
	}

	store, err := file.New(dst)
	if err != nil {
		return fmt.Errorf("creating file store: %w", err)
	}
	defer store.Close()

	_, err = oras.Copy(ctx,
		repo, reference,
		store, "",
		oras.DefaultCopyOptions,
	)
	if err != nil {
		return fmt.Errorf("pulling OCI artifact %q: %w", ref, err)
	}

	return nil
}

// shellOutOrasPull shells out to the oras CLI. Kept for fallback use.
// DEPRECATED: prefer orasPull (Go library).
func shellOutOrasPull(ref, dst string, auth *utils.FileAuth) error {
	args := []string{"pull", "--output", dst}

	if auth != nil {
		switch strings.ToLower(auth.Type) {
		case "basic":
			if auth.Username != "" {
				args = append(args, "--username", auth.Username)
			}
			if auth.Password != "" {
				args = append(args, "--password", auth.Password)
			}
		case "bearer", "github":
			if auth.BearerToken != "" {
				args = append(args, "--password", auth.BearerToken)
			}
		}
	}

	args = append(args, ref)

	cmd := exec.Command("oras", args...)
	cmd.Dir = dst
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// ── Git pull ──────────────────────────────────────────────────────────────────

func (m *Merger) pullGitPattern(url, version, tmpDir string, auth *utils.FileAuth) error {
	switch {
	case isGitHubURL(url):
		return m.pullGitHubPattern(url, version, tmpDir, auth)
	case isGitLabURL(url):
		return m.pullGitLabPattern(url, version, tmpDir, auth)
	default:
		return pullGenericGitPattern(url, version, tmpDir, auth)
	}
}

// pullGitHubPattern fetches each required file individually via GitHub raw URLs.
// Optional files (motif.yaml) are fetched silently when present.
func (m *Merger) pullGitHubPattern(url, version, tmpDir string, auth *utils.FileAuth) error {
	for _, filename := range orktypes.RequiredFiles {
		rawURL := githubRawURL(url, version, filename)
		data, err := utils.LoadFileWithAuth(rawURL, auth)
		if err != nil {
			return fmt.Errorf("fetching %q from GitHub at ref %q: %w", filename, version, err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, filename), data, 0644); err != nil {
			return fmt.Errorf("writing %q: %w", filename, err)
		}
	}
	for _, filename := range optionalPatternFiles {
		rawURL := githubRawURL(url, version, filename)
		if data, err := utils.LoadFileWithAuth(rawURL, auth); err == nil {
			_ = os.WriteFile(filepath.Join(tmpDir, filename), data, 0644)
		}
	}
	return nil
}

// pullGitLabPattern fetches each required file individually via GitLab raw URLs.
// Optional files (motif.yaml) are fetched silently when present.
func (m *Merger) pullGitLabPattern(url, version, tmpDir string, auth *utils.FileAuth) error {
	for _, filename := range orktypes.RequiredFiles {
		rawURL := gitlabRawURL(url, version, filename)
		data, err := utils.LoadFileWithAuth(rawURL, auth)
		if err != nil {
			return fmt.Errorf("fetching %q from GitLab at ref %q: %w", filename, version, err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, filename), data, 0644); err != nil {
			return fmt.Errorf("writing %q: %w", filename, err)
		}
	}
	for _, filename := range optionalPatternFiles {
		rawURL := gitlabRawURL(url, version, filename)
		if data, err := utils.LoadFileWithAuth(rawURL, auth); err == nil {
			_ = os.WriteFile(filepath.Join(tmpDir, filename), data, 0644)
		}
	}
	return nil
}

// pullGenericGitPattern clones the repository and copies the required + optional files.
func pullGenericGitPattern(url, version, tmpDir string, auth *utils.FileAuth) error {
	cloneDir, err := os.MkdirTemp("", "orkestra-clone-*")
	if err != nil {
		return fmt.Errorf("creating clone dir: %w", err)
	}
	defer os.RemoveAll(cloneDir)

	cloneURL := injectAuthIntoURL(url, auth)
	if err := gitClone(cloneURL, cloneDir, version); err != nil {
		return err
	}

	allFiles := append(orktypes.RequiredFiles, optionalPatternFiles...)
	for _, filename := range allFiles {
		src := filepath.Join(cloneDir, filename)
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		if err := os.WriteFile(filepath.Join(tmpDir, filename), data, 0644); err != nil {
			return fmt.Errorf("copying %q: %w", filename, err)
		}
	}

	return nil
}

// pullMotifFromGit fetches only motif.yaml from a Git host.
// Used for standalone Motif repos (not full patterns).
func (m *Merger) pullMotifFromGit(url, version, tmpDir string, auth *utils.FileAuth) error {
	var rawURL string
	switch {
	case isGitHubURL(url):
		rawURL = githubRawURL(url, version, "motif.yaml")
	case isGitLabURL(url):
		rawURL = gitlabRawURL(url, version, "motif.yaml")
	default:
		return m.fetchMotifFromGenericGit(url, version, tmpDir, auth)
	}

	data, err := utils.LoadFileWithAuth(rawURL, auth)
	if err != nil {
		return fmt.Errorf("fetching motif.yaml from %s@%s: %w", url, version, err)
	}
	return os.WriteFile(filepath.Join(tmpDir, "motif.yaml"), data, 0644)
}

// fetchMotifFromGenericGit clones a repo and copies motif.yaml.
func (m *Merger) fetchMotifFromGenericGit(url, version, tmpDir string, auth *utils.FileAuth) error {
	cloneDir, err := os.MkdirTemp("", "orkestra-motif-clone-*")
	if err != nil {
		return fmt.Errorf("creating clone dir: %w", err)
	}
	defer os.RemoveAll(cloneDir)

	cloneURL := injectAuthIntoURL(url, auth)
	if err := gitClone(cloneURL, cloneDir, version); err != nil {
		return err
	}

	data, err := os.ReadFile(filepath.Join(cloneDir, "motif.yaml"))
	if err != nil {
		return fmt.Errorf("motif.yaml not found in repository %s@%s", url, version)
	}
	return os.WriteFile(filepath.Join(tmpDir, "motif.yaml"), data, 0644)
}

// ── Pattern validation ────────────────────────────────────────────────────────

func validatePatternStructure(dir, url, version string) error {
	var violations []string

	for _, filename := range orktypes.RequiredFiles {
		path := filepath.Join(dir, filename)

		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			violations = append(violations, fmt.Sprintf("  missing: %s", filename))
			continue
		}
		if err != nil {
			violations = append(violations, fmt.Sprintf("  error checking %s: %v", filename, err))
			continue
		}
		if info.Size() == 0 {
			violations = append(violations, fmt.Sprintf("  empty:   %s", filename))
		}
	}

	if len(violations) == 0 {
		return nil
	}

	return fmt.Errorf(
		"registry pattern %q@%s failed structure validation:\n%s\n\n"+
			"Every Orkestra registry pattern must contain these five files,\n"+
			"each non-empty:\n"+
			"  crd.yaml        the CRD definition\n"+
			"  katalog.yaml    operator behavior and reconcile templates\n"+
			"  komposer.yaml   example import showing how to consume this pattern\n"+
			"  cr.yaml         example custom resource to test with\n"+
			"  README.md       documentation — fields, overrides, examples\n\n"+
			"See: https://docs.orkestra.io/registry/contributing",
		url, version,
		strings.Join(violations, "\n"),
	)
}

// ── Deprecated registry protocol (catalog-map based) ─────────────────────────
// The functions below implement the older registry.katalog map-based protocol.
// New code should use loadRegistrySource (the v2 OCI/Git pull protocol).

// loadRegistrySourceDeprecated loads all Katalog entries declared in one RegistrySource block
// using the older catalog-map protocol.
func (m *Merger) loadRegistrySourceDeprecated(src orktypes.RegistrySource) (map[string]orktypes.CRDEntry, error) {
	registryURL, err := m.resolveRegistryURL(src.URL)
	if err != nil {
		return nil, err
	}

	if len(src.Katalog) == 0 {
		logger.Warn().
			Str("registry", registryURL).
			Msg("merger: registry source has no katalog entries — nothing to load")
		return nil, nil
	}

	auth, err := resolveRegistryAuth(src.Auth)
	if err != nil {
		return nil, fmt.Errorf("registry %q: auth: %w", registryURL, err)
	}

	allCRDs := make(map[string]orktypes.CRDEntry)

	for name, ref := range src.Katalog {
		crds, err := m.loadRegistryKatalog(registryURL, name, ref, auth)
		if err != nil {
			return nil, fmt.Errorf("registry %q: katalog %q: %w", registryURL, name, err)
		}
		for k, v := range crds {
			allCRDs[k] = v
		}

		logger.Info().
			Str("registry", registryURL).
			Str("katalog", name).
			Str("ref", ref.Ref()).
			Int("crds", len(crds)).
			Msg("merger: registry katalog loaded")
	}

	return allCRDs, nil
}

func (m *Merger) loadRegistryKatalog(
	registryURL, name string,
	ref orktypes.RegistryRef,
	auth *utils.FileAuth,
) (map[string]orktypes.CRDEntry, error) {
	filePath := filepath.Join(registryKatalogPath, name, registryFileName)
	gitRef := ref.Ref()

	var data []byte
	var err error

	switch {
	case isGitHubURL(registryURL):
		rawURL := githubRawURL(registryURL, gitRef, filePath)
		data, err = utils.LoadFileWithAuth(rawURL, auth)
		if err != nil {
			return nil, fmt.Errorf(
				"fetching %q at ref %q from GitHub registry: %w\n  URL tried: %s",
				name, gitRef, err, rawURL,
			)
		}

	case isGitLabURL(registryURL):
		rawURL := gitlabRawURL(registryURL, gitRef, filePath)
		data, err = utils.LoadFileWithAuth(rawURL, auth)
		if err != nil {
			return nil, fmt.Errorf(
				"fetching %q at ref %q from GitLab registry: %w\n  URL tried: %s",
				name, gitRef, err, rawURL,
			)
		}

	default:
		data, err = m.fetchFromGitRegistry(registryURL, gitRef, filePath, auth)
		if err != nil {
			return nil, fmt.Errorf(
				"fetching %q at ref %q from git registry %q: %w",
				name, gitRef, registryURL, err,
			)
		}
	}

	sourcePath := fmt.Sprintf("registry:%s/%s@%s", registryURL, name, gitRef)
	doc, err := parseKatalogDoc(data, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("parsing katalog %q: %w", name, err)
	}
	if doc == nil {
		return nil, fmt.Errorf(
			"katalog %q at ref %q is not a valid Katalog document — expected kind: Katalog at %s",
			name, gitRef, filePath,
		)
	}

	return m.loadKatalog(sourcePath, doc)
}

func (m *Merger) fetchFromGitRegistry(
	registryURL, ref, filePath string,
	auth *utils.FileAuth,
) ([]byte, error) {
	cloneURL := injectAuthIntoURL(registryURL, auth)

	tmpDir, err := os.MkdirTemp("", "orkestra-registry-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := gitClone(cloneURL, tmpDir, ref); err != nil {
		return nil, err
	}

	fullPath := filepath.Join(tmpDir, filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"file %q not found in registry at ref %q — "+
				"check that the katalog name and registry structure are correct",
			filePath, ref,
		)
	}

	return os.ReadFile(fullPath)
}

// ── Registry URL resolution ───────────────────────────────────────────────────

func (m *Merger) resolveRegistryURL(srcURL string) (string, error) {
	if srcURL != "" {
		return srcURL, nil
	}

	env := os.Getenv("ORK_REGISTRY")
	if env != "" {
		return env, nil
	}

	if m.registryURL != "" {
		return m.registryURL, nil
	}

	return "", fmt.Errorf(
		"no registry URL configured for this source.\n\n" +
			"Set the registry URL using one of:\n" +
			"  1. ORK_REGISTRY environment variable:\n" +
			"       export ORK_REGISTRY=https://github.com/myorg/orkestra-registry\n\n" +
			"  2. Explicit url in the source block:\n" +
			"       sources:\n" +
			"         registry:\n" +
			"           - url: https://github.com/myorg/orkestra-registry\n" +
			"             katalog:\n" +
			"               website:\n" +
			"                 branch: main",
	)
}

// ── URL construction ──────────────────────────────────────────────────────────

func isGitHubURL(u string) bool {
	return strings.Contains(u, "github.com") && !strings.HasSuffix(u, ".git")
}

func isGitLabURL(u string) bool {
	return strings.Contains(u, "gitlab.com") && !strings.HasSuffix(u, ".git")
}

func githubRawURL(repoURL, ref, filePath string) string {
	u := strings.TrimSuffix(strings.TrimSuffix(repoURL, ".git"), "/")
	u = strings.Replace(u, "https://github.com/", "https://raw.githubusercontent.com/", 1)
	u = strings.Replace(u, "http://github.com/", "https://raw.githubusercontent.com/", 1)
	return fmt.Sprintf("%s/%s/%s", u, ref, filePath)
}

func gitlabRawURL(repoURL, ref, filePath string) string {
	u := strings.TrimSuffix(strings.TrimSuffix(repoURL, ".git"), "/")
	return fmt.Sprintf("%s/-/raw/%s/%s", u, ref, filePath)
}

func injectAuthIntoURL(rawURL string, auth *utils.FileAuth) string {
	if auth == nil || strings.ToLower(auth.Type) != "basic" || auth.Username == "" {
		return rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	parsed.User = url.UserPassword(auth.Username, auth.Password)
	return parsed.String()
}

// ── Auth resolution ───────────────────────────────────────────────────────────

func resolveRegistryAuth(auth *orktypes.FileSourceAuth) (*utils.FileAuth, error) {
	if auth == nil {
		return nil, nil
	}
	return auth.Resolve()
}
