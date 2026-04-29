// pkg/registry/resolve.go
//
// Reference resolution for ork registry commands.
//
// A bare reference like "postgres:v14" is resolved to a full OCI reference
// using the following priority:
//
//  1. Full OCI reference (starts with "oci://") — used as-is
//  2. ORKESTRA_REGISTRY env var + "/name:version"
//  3. Default: ghcr.io/orkspace/orkestra-registry/name:version
//
// The "oci://" prefix is stripped before passing to ORAS — it is a user-facing
// convention to signal "this is an OCI reference", not part of the actual URL.
package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultRegistry is the official Orkestra pattern registry.
	DefaultRegistry = "ghcr.io/orkspace/orkestra-registry/patterns"

	// EnvRegistry is the environment variable for overriding the registry.
	EnvRegistry = "ORKESTRA_REGISTRY"

	// CacheDir is the local cache directory for pulled patterns.
	// Resolved relative to the user's home directory.
	CacheDir = ".orkestra/registry"
)

// Ref holds a resolved OCI reference.
type Ref struct {
	// Registry is the hostname (e.g. "ghcr.io").
	Registry string

	// Repository is the full repository path without the registry
	// (e.g. "orkspace/orkestra-registry/postgres").
	Repository string

	// Tag is the version tag (e.g. "v14").
	Tag string

	// Full is the complete reference without the oci:// prefix.
	// Suitable for passing directly to ORAS.
	Full string
}

// Resolve converts a user-supplied reference to a fully qualified OCI Ref.
//
//	"postgres:v14"                          → ghcr.io/orkspace/orkestra-registry/postgres:v14
//	"oci://ghcr.io/myorg/patterns/redis:v7" → ghcr.io/myorg/patterns/redis:v7
//	"myorg/redis:v7" (with ORKESTRA_REGISTRY set) → resolved against env
func Resolve(input string) (*Ref, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty reference")
	}

	// Strip oci:// prefix — user-facing convention only
	raw := strings.TrimPrefix(input, "oci://")

	// Full reference already provided (contains a dot in the host segment)
	if looksLikeFull(raw) {
		return parseRef(raw)
	}

	// Use ORKESTRA_REGISTRY env var or default
	base := os.Getenv(EnvRegistry)
	if base == "" {
		base = DefaultRegistry
	}
	base = strings.TrimPrefix(base, "oci://")
	base = strings.TrimSuffix(base, "/")

	return parseRef(base + "/" + raw)
}

// looksLikeFull returns true when the reference appears to already contain
// a registry hostname (has a dot or colon before the first slash).
func looksLikeFull(ref string) bool {
	slashIdx := strings.Index(ref, "/")
	if slashIdx < 0 {
		return false
	}
	host := ref[:slashIdx]
	return strings.Contains(host, ".") || strings.Contains(host, ":") || host == "localhost"
}

// parseRef splits a full reference into its components.
func parseRef(full string) (*Ref, error) {
	// Separate tag
	tag := "latest"
	name := full
	if idx := strings.LastIndex(full, ":"); idx > strings.LastIndex(full, "/") {
		tag = full[idx+1:]
		name = full[:idx]
	}

	// Separate registry from repository
	slashIdx := strings.Index(name, "/")
	if slashIdx < 0 {
		return nil, fmt.Errorf("invalid reference %q: missing repository path", full)
	}
	registry := name[:slashIdx]
	repo := name[slashIdx+1:]

	return &Ref{
		Registry:   registry,
		Repository: repo,
		Tag:        tag,
		Full:       full,
	}, nil
}

// CachePath returns the local filesystem path for this ref.
// Structure: ~/.orkestra/registry/<registry>/<repository>/<tag>/
func (r *Ref) CachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	// Replace colons in port numbers for filesystem safety
	registry := strings.ReplaceAll(r.Registry, ":", "_")
	repo := filepath.FromSlash(r.Repository)
	return filepath.Join(home, CacheDir, registry, repo, r.Tag), nil
}

// IsCached returns true when the pattern is already in the local cache.
func (r *Ref) IsCached() bool {
	path, err := r.CachePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(path, FileKatalog))
	return err == nil
}

// String returns the full reference with oci:// prefix for display.
func (r *Ref) String() string {
	return "oci://" + r.Full
}

// ShortName returns the name:tag portion for display.
func (r *Ref) ShortName() string {
	parts := strings.Split(r.Repository, "/")
	return parts[len(parts)-1] + ":" + r.Tag
}
