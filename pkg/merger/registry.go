// pkg/merger/registry.go
package merger

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
)

// registryPath is the root directory inside the registry repo where katalogs live.
// registry/katalogs/<name>/katalog.yaml
const (
	registryKatalogPath = "registry/katalogs"
	registryFileName    = "katalog.yaml"
)

// loadRegistrySource loads all Katalog entries declared in one RegistrySource block.
//
// Resolution order:
//  1. src.URL if set — explicit registry URL for this source
//  2. ORK_REGISTRY environment variable — global registry URL
//  3. Error — no registry configured
//
// For GitHub and GitLab URLs, raw file content is fetched directly via HTTPS —
// no git clone needed, one HTTP request per katalog entry.
// For all other URLs, the registry is cloned to a temp dir and files are read locally.
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

	// Resolve auth for this registry source
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

// loadRegistryKatalog fetches one katalog from the registry by name and ref.
//
// File path inside the registry: registry/katalogs/<name>/katalog.yaml
//
// For GitHub/GitLab: constructs a raw file URL and fetches directly.
// For other Git URLs: clones the repo at the given ref and reads the file.
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
		logger.Debug().
			Str("url", rawURL).
			Str("katalog", name).
			Msg("merger: fetching katalog from GitHub registry")
		data, err = utils.LoadFileWithAuth(rawURL, auth)
		if err != nil {
			return nil, fmt.Errorf(
				"fetching %q at ref %q from GitHub registry: %w\n"+
					"  URL tried: %s",
				name, gitRef, err, rawURL,
			)
		}

	case isGitLabURL(registryURL):
		rawURL := gitlabRawURL(registryURL, gitRef, filePath)
		logger.Debug().
			Str("url", rawURL).
			Str("katalog", name).
			Msg("merger: fetching katalog from GitLab registry")
		data, err = utils.LoadFileWithAuth(rawURL, auth)
		if err != nil {
			return nil, fmt.Errorf(
				"fetching %q at ref %q from GitLab registry: %w\n"+
					"  URL tried: %s",
				name, gitRef, err, rawURL,
			)
		}

	default:
		// Generic Git URL — clone and read
		logger.Debug().
			Str("registry", registryURL).
			Str("katalog", name).
			Str("ref", gitRef).
			Msg("merger: cloning registry to fetch katalog")
		data, err = m.fetchFromGitRegistry(registryURL, gitRef, filePath, auth)
		if err != nil {
			return nil, fmt.Errorf(
				"fetching %q at ref %q from git registry %q: %w",
				name, gitRef, registryURL, err,
			)
		}
	}

	// Parse the fetched data as a Katalog document
	sourcePath := fmt.Sprintf("registry:%s/%s@%s", registryURL, name, gitRef)
	doc, err := parseKatalogDoc(data, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("parsing katalog %q: %w", name, err)
	}
	if doc == nil {
		return nil, fmt.Errorf(
			"katalog %q at ref %q is not a valid Katalog document — "+
				"expected kind: Katalog at %s",
			name, gitRef, filePath,
		)
	}

	return m.loadKatalog(sourcePath, doc)
}

// fetchFromGitRegistry clones a git registry to a temp dir and reads a file.
// Used for non-GitHub/GitLab registries where raw URL construction is not standard.
func (m *Merger) fetchFromGitRegistry(
	registryURL, ref, filePath string,
	auth *utils.FileAuth,
) ([]byte, error) {
	// Inject auth into the clone URL if basic auth is provided
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

// resolveRegistryURL returns the effective registry URL.
// Priority: explicit src.URL > ORK_REGISTRY env var.
func (m *Merger) resolveRegistryURL(srcURL string) (string, error) {
	if srcURL != "" {
		return srcURL, nil
	}

	env := os.Getenv("ORK_REGISTRY")
	if env != "" {
		return env, nil
	}

	// Check the merger's configured registry (set via konstructOrkestra)
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

// isGitHubURL reports whether the URL points to a GitHub repository.
func isGitHubURL(u string) bool {
	return strings.Contains(u, "github.com") && !strings.HasSuffix(u, ".git")
}

// isGitLabURL reports whether the URL points to a GitLab repository.
func isGitLabURL(u string) bool {
	return strings.Contains(u, "gitlab.com") && !strings.HasSuffix(u, ".git")
}

// githubRawURL constructs the raw content URL for a file in a GitHub repository.
//
// https://github.com/owner/repo  →
// https://raw.githubusercontent.com/owner/repo/<ref>/<path>
func githubRawURL(repoURL, ref, filePath string) string {
	// Normalise: strip trailing .git, strip trailing /
	u := strings.TrimSuffix(strings.TrimSuffix(repoURL, ".git"), "/")

	// Replace github.com with raw.githubusercontent.com
	u = strings.Replace(u, "https://github.com/", "https://raw.githubusercontent.com/", 1)
	u = strings.Replace(u, "http://github.com/", "https://raw.githubusercontent.com/", 1)

	return fmt.Sprintf("%s/%s/%s", u, ref, filePath)
}

// gitlabRawURL constructs the raw content URL for a file in a GitLab repository.
//
// https://gitlab.com/owner/repo  →
// https://gitlab.com/owner/repo/-/raw/<ref>/<path>
func gitlabRawURL(repoURL, ref, filePath string) string {
	u := strings.TrimSuffix(strings.TrimSuffix(repoURL, ".git"), "/")
	return fmt.Sprintf("%s/-/raw/%s/%s", u, ref, filePath)
}

// injectAuthIntoURL injects basic auth credentials into a URL for git clone.
// Returns the URL unchanged when auth is nil or not basic type.
func injectAuthIntoURL(rawURL string, auth *utils.FileAuth) string {
	if auth == nil || strings.ToLower(auth.Type) != "basic" || auth.Username == "" {
		return rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // can't parse — return as-is, clone will fail with original error
	}

	parsed.User = url.UserPassword(auth.Username, auth.Password)
	return parsed.String()
}

// ── Auth resolution ───────────────────────────────────────────────────────────

// resolveRegistryAuth resolves a FileSourceAuth to a FileAuth by reading
// environment variables. Returns nil when auth is nil (unauthenticated).
func resolveRegistryAuth(auth *orktypes.FileSourceAuth) (*utils.FileAuth, error) {
	if auth == nil {
		return nil, nil
	}
	return auth.Resolve()
}
