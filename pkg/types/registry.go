// pkg/types/registry.go
package orktypes

// RegistrySource declares one or more Katalogs to pull from an Orkestra registry.
//
// An Orkestra registry is a Git repository with this structure:
//
//   registry/
//     katalogs/
//       website/
//         katalog.yaml
//       platform-namespace/
//         katalog.yaml
//     hooks/
//       website-hooks/
//         ...
//     core/				Implementations for orkestra runtime
//       deployments/
//         deployment.go
// 		service/
// 			service.go
//
// The registry URL is resolved from the ORK_REGISTRY environment variable
// unless overridden per-source with the url field.
//
// Example Komposer declaration:
//
//   sources:
//     registry:
//       - katalog:
//           website:
//             branch: main
//           platform-namespace:
//             version: v1.2.0
//           application:
//             sha: abc123def456
//
//       # With explicit registry URL and auth (overrides ORK_REGISTRY)
//       - url: https://github.com/myorg/private-registry
//         auth:
//           type: github
//           fromEnv: GITHUB_TOKEN
//         katalog:
//           internal-crd:
//             branch: main

// RegistrySource is one entry in sources.registry.
// Multiple entries can reference different registries with different auth.
type RegistrySource struct {
	// URL — explicit registry URL for this source entry.
	// When empty, resolved from the ORK_REGISTRY environment variable.
	// Supports GitHub, GitLab, and any Git URL.
	URL string `yaml:"url,omitempty"`

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
