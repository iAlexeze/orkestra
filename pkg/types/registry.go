// pkg/types/registry.go
package types

import "strings"

// RegistrySource declares one pattern to pull from a registry.
//
// A registry source can be Git-based or OCI-based. The source type is
// determined by the oci field (false by default — git pull).
//
// After pulling, Orkestra validates the five required files exist and are
// non-empty. It then loads either katalog.yaml (default) or komposer.yaml
// based on useKomposer. Exactly one is loaded — not both.
//
// URL shorthand — version inline with @:
//
//   - url: ghcr.io/konduktor-io/orkestra-registry/postgres@v14
//     oci: true
//
// Explicit form:
//
//   - url: ghcr.io/konduktor-io/orkestra-registry/postgres
//     version: v14
//     oci: true
//
// Git form:
//
//   - url: https://github.com/myorg/registry
//     version: main
//     oci: false      # default
//     useKomposer: true
//     auth:
//       type: github
//       fromEnv: GITHUB_TOKEN
//
// Private OCI with auth:
//
//   - url: registry.myorg.com/operators/postgres@v14-hardened
//     oci: true
//     auth:
//       type: basic
//       usernameFromEnv: REGISTRY_USER
//       passwordFromEnv: REGISTRY_PASSWORD

// RegistrySource is one entry in sources.registry.
type RegistrySource struct {
	// URL — the registry URL.
	//
	// Git:  https://github.com/myorg/orkestra-registry
	// OCI:  ghcr.io/konduktor-io/orkestra-registry/postgres
	//
	// Shorthand — embed version with @:
	//   ghcr.io/konduktor-io/orkestra-registry/postgres@v14
	//   https://github.com/myorg/registry@main
	//
	// When @ is present, Version field is ignored.
	URL string `yaml:"url" validate:"required"`

	// Version — the version, tag, branch, or SHA to pull.
	//
	// For OCI:  semantic version tag    "v14", "v14.2.0"
	// For Git:  branch, tag, or SHA     "main", "v1.0.0", "abc123"
	//
	// Ignored when URL contains @.
	// Defaults to "main" for Git, "latest" for OCI when not set.
	Version string `yaml:"version,omitempty"`

	// OCI — when true, pull the pattern as an OCI artifact.
	// When false (default), pull via Git clone or raw file fetch.
	//
	// OCI artifacts are pulled using the ORAS protocol.
	// GitHub and GitLab URLs with oci: false use raw file HTTP — no clone needed.
	// Other Git URLs with oci: false use git clone.
	OCI bool `yaml:"oci,omitempty"`

	// UseKomposer — when true, load komposer.yaml from the pulled pattern.
	// When false (default), load katalog.yaml.
	//
	// Exactly one file is loaded — not both.
	//
	// UseKomposer: false (default)
	//   Use this when you want the CRD definitions and will override them
	//   inline in your own Komposer. This is the common case.
	//
	// UseKomposer: true
	//   Use this when you want to accept the upstream operator's full
	//   source tree as-is — their sources, their defaults, everything.
	//   Useful for internal teams with a canonical registry where the
	//   upstream Komposer is exactly what you want to run.
	//
	// Warning: loading a Komposer from a registry source means that
	// Komposer's own sources are also resolved. A Komposer that sources
	// other Katalogs will pull those too. Understand the upstream
	// dependency tree before enabling this.
	UseKomposer bool `yaml:"useKomposer,omitempty"`

	// Auth — optional authentication for the registry.
	// When empty, requests are unauthenticated.
	// Auth credentials are resolved from environment variables — never literals.
	Auth *FileSourceAuth `yaml:"auth,omitempty"`

	// Katalog — map of katalog names to their version references.
	// Key: katalog name (directory name under registry/katalogs/).
	// Value: version reference (branch, sha, or version tag).
	Katalog map[string]RegistryRef `yaml:"katalog,omitempty"`

	// Future source types (not yet implemented):
	// Hooks map[string]RegistryRef `yaml:"hooks,omitempty"`
	// Core  map[string]RegistryRef `yaml:"core,omitempty"`
}

// RegistryRef declares how to resolve a specific item from the registry.
// Exactly one of Branch, SHA, or Version should be set.
// When none are set, defaults to the default branch of the registry.
//
// Example:
//
//	website:
//	  branch: main        # track a branch
//
//	platform-namespace:
//	  version: v1.2.0     # pin to a release tag
//
//	application:
//	  sha: abc123def456   # pin to an exact commit
type RegistryRef struct {
	// Branch — track a branch (e.g. "main", "develop").
	// The latest commit on the branch is fetched.
	Branch string `yaml:"branch,omitempty"`

	// Version — pin to a release tag (e.g. "v1.2.0", "v2").
	// Git tags are fetched — version is used as the tag name.
	Version string `yaml:"version,omitempty"`

	// SHA — pin to an exact commit hash.
	// The full or partial SHA works (Git resolves partial SHAs).
	SHA string `yaml:"sha,omitempty"`
}

// Ref returns the effective git ref for this RegistryRef.
// Priority: SHA > Version > Branch > "main" (default).
func (r RegistryRef) Ref() string {
	if r.SHA != "" {
		return r.SHA
	}
	if r.Version != "" {
		return r.Version
	}
	if r.Branch != "" {
		return r.Branch
	}
	return "main"
}

// IsDefault reports whether no ref is explicitly set.
func (r RegistryRef) IsDefault() bool {
	return r.SHA == "" && r.Version == "" && r.Branch == ""
}

// ResolvedURL returns the effective URL with the version extracted.
// Handles both the @ shorthand and the explicit Version field.
//
// Returns (cleanURL, version).
// cleanURL is the URL without the @ suffix.
// version is the resolved version string (never empty).
func (r RegistrySource) ResolvedURL() (cleanURL, version string) {
	url := strings.TrimSpace(r.URL)

	if idx := strings.LastIndex(url, "@"); idx != -1 {
		// @ shorthand — split on last @
		// handles: ghcr.io/myorg/postgres@v14
		// handles: https://github.com/myorg/registry@main
		cleanURL = url[:idx]
		version = url[idx+1:]
		return
	}

	cleanURL = url
	version = strings.TrimSpace(r.Version)
	if version == "" {
		version = r.defaultVersion()
	}
	return
}

// defaultVersion returns the default version when none is declared.
func (r RegistrySource) defaultVersion() string {
	if r.OCI {
		return "latest"
	}
	return "main"
}

// SourceFile returns the filename Orkestra should load after pulling the pattern.
// Either "katalog.yaml" or "komposer.yaml" — never both.
func (r RegistrySource) SourceFile() string {
	if r.UseKomposer {
		return "komposer.yaml"
	}
	return "katalog.yaml"
}

// RequiredFiles lists the five files every valid registry pattern must contain.
// Validated after pull — fail fast if any are missing or empty.
//
// These five files define the standard pattern structure. Enforcing their
// presence at pull time means every pattern in the ecosystem is documented,
// testable, and consistent. A registry entry without a README, a CRD, or
// an example CR is not ready for distribution.
var RequiredFiles = []string{
	"crd.yaml",
	"katalog.yaml",
	"komposer.yaml",
	"cr.yaml",
	"README.md",
}
