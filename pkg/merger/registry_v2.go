// pkg/merger/registry_v2.go
package merger

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	orasauth "oras.land/oras-go/v2/registry/remote/auth"
)

const orasPullTimeout = 2 * time.Minute

// loadRegistrySource loads a single registry pattern entry.
//
// Resolution sequence:
//  1. Parse url@version shorthand or url + version fields
//  2. Determine pull method: OCI or Git
//  3. Pull pattern to a temp directory
//  4. Validate all 5 required files exist and are non-empty (fail fast)
//  5. Load katalog.yaml or komposer.yaml based on UseKomposer
//  6. Parse and return the CRD entries
func (m *Merger) loadRegistrySource(src orktypes.RegistrySource) ([]orktypes.CRDEntry, error) {
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

	// Route to the correct loader based on the document kind
	switch doc.Kind {
	case konfig.KatalogKind():
		if src.UseKomposer {
			// useKomposer: true but file is a Katalog — informative error
			return nil, fmt.Errorf(
				"registry %q@%s: useKomposer is true but komposer.yaml contains kind %q — "+
					"check the upstream pattern's komposer.yaml",
				cleanURL, version, doc.Kind,
			)
		}
		return m.loadKatalog(sourcePath, doc)

	case konfig.KomposerKind():
		if !src.UseKomposer {
			// useKomposer: false but loaded katalog.yaml contains a Komposer
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
//
// Dispatches to OCI pull or Git pull based on the oci flag.
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
		// err = m.deprecatedPullOCIPattern(url, version, tmpDir, auth)
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

// DEPRECATED
// deprecatedPullOCIPattern fetches an OCI artifact and extracts it to tmpDir.
// Uses ORAS protocol. The artifact ref is constructed as: url:version
func (m *Merger) deprecatedPullOCIPattern(url, version, tmpDir string, auth *utils.FileAuth) error {
	// Normalise URL — OCI refs do not have scheme prefixes
	ref := strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
	ref = strings.TrimSuffix(ref, "/")
	ref = fmt.Sprintf("%s:%s", ref, version)

	logger.Debug().
		Str("ref", ref).
		Str("dst", tmpDir).
		Msg("registry: pulling OCI artifact")

	// Build the ORAS pull command
	// When the ork registry CLI is complete, this will use the ORAS Go library.
	// For now, try shell out to oras if available, otherwise return a clear error.
	if err := shellOutOrasPull(ref, tmpDir, auth); err != nil {
		return fmt.Errorf("OCI pull %q: %w\n\n"+
			"  Ensure the oras CLI is installed: https://oras.land/docs/installation\n"+
			"  Or use oci: false with a Git URL to pull via git instead.",
			ref, err)
	}

	logger.Info().
		Str("ref", ref).
		Msg("registry: OCI artifact pulled")

	return nil
}

// pullOCIPattern fetches an OCI artifact and extracts it to tmpDir using ORAS Go library.
func (m *Merger) pullOCIPattern(url, version, tmpDir string, auth *utils.FileAuth) error {
	// Normalise URL — OCI refs do not have scheme prefixes
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

	logger.Info().
		Str("ref", ref).
		Str("dst", tmpDir).
		Msg("registry: OCI artifact pulled successfully")

	return nil
}

// orasPull pulls an OCI artifact using the ORAS Go library into dst.
func orasPull(ref, dst string, auth *utils.FileAuth) error {
	ctx, cancel := context.WithTimeout(context.Background(), orasPullTimeout)
	defer cancel()

	// Split repo and reference (tag or digest)
	repoName, reference, ok := strings.Cut(ref, ":")
	if !ok || reference == "" {
		return fmt.Errorf("invalid OCI reference %q: missing tag or digest", ref)
	}

	repo, err := remote.NewRepository(repoName)
	if err != nil {
		return fmt.Errorf("creating repository for %q: %w", repoName, err)
	}

	// Configure authentication
	// TODO: Not working with GHCR
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
						// Token-style auth
						return orasauth.Credential{
							RefreshToken: auth.BearerToken,
						}, nil
					}
				}
				return orasauth.EmptyCredential, nil
			},
		}
	}

	// Local file store (extracts files into dst)
	store, err := file.New(dst)
	if err != nil {
		return fmt.Errorf("creating file store: %w", err)
	}
	defer store.Close()

	// Copy from remote repo@reference into local store
	_, err = oras.Copy(ctx,
		repo, reference, // source: repo + tag/digest
		store, "", // dest: file store
		oras.DefaultCopyOptions,
	)
	if err != nil {
		return fmt.Errorf("pulling OCI artifact %q: %w", ref, err)
	}

	return nil
}

// DEPRECATED
// shellOutOrasPull shells out to the oras CLI to pull an OCI artifact.
// This is a bridge until the ORAS Go library is integrated natively.
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
				// oras accepts --password for token auth as well
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

// pullGitPattern fetches a pattern from a Git-based registry.
// Reuses the existing Git/HTTP infrastructure for GitHub, GitLab, and generic Git.
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
// No clone needed — one HTTP request per file.
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
	return nil
}

// pullGitLabPattern fetches each required file individually via GitLab raw URLs.
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
	return nil
}

// pullGenericGitPattern clones the repository and copies the required files.
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

	// Copy only the required files from the clone to tmpDir
	for _, filename := range orktypes.RequiredFiles {
		src := filepath.Join(cloneDir, filename)
		data, err := os.ReadFile(src)
		if err != nil {
			// File might be optional — collect and validate in validatePatternStructure
			continue
		}
		if err := os.WriteFile(filepath.Join(tmpDir, filename), data, 0644); err != nil {
			return fmt.Errorf("copying %q: %w", filename, err)
		}
	}

	return nil
}

// ── Pattern validation ────────────────────────────────────────────────────────

// validatePatternStructure checks that all five required files exist and are
// non-empty in the pulled pattern directory.
//
// Fail fast — return an error describing every violation, not just the first.
// A pattern that is missing a CRD definition, a README, or an example CR is
// not ready for distribution and should not be used.
//
// This enforcement exists to make the ecosystem consistent. Every pattern
// in OrkestraRegistry is documented (README.md), testable (cr.yaml), and
// structurally complete. The check runs at pull time so failures surface
// immediately during `ork validate` rather than at runtime.
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
