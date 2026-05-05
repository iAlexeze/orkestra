// pkg/motif/loader.go
//
// Loads a Motif from a file path or registry reference.
// Resolution follows the same semantics as RegistrySource in a Komposer —
// if you know how to pull a pattern, you already know how to pull a Motif.
//
// The Orkestra registry houses both patterns and motifs. Each Motif is a
// standalone OCI artifact or Git repo with motif.yaml at its root.
package motif

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orkspace/orkestra/pkg/merger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"gopkg.in/yaml.v3"
)

// Load loads a Motif from a local file path.
// For full import resolution (registry, OCI, auth), use LoadImport.
func Load(path string) (*orktypes.Motif, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading motif %s: %w", path, err)
	}
	return parse(data)
}

// LoadImport resolves and loads a Motif from a MotifImport declaration.
// Supports file paths, OCI artifacts, and Git registries — the same
// resolution semantics as RegistrySource in a Komposer.
//
// File path (developer path):
//
//	imp.Motif = "./postgres/motif.yaml"
//
// OCI artifact — motif.yaml at artifact root:
//
//	imp.Motif = "ghcr.io/orkspace/orkestra-registry/postgres@v16"
//	imp.OCI   = true
//
// Git registry — motif.yaml at repo root (standalone Motif repo):
//
//	imp.Motif = "https://github.com/myorg/postgres-motif@main"
//
// Pattern with bundled Motif (pattern includes both katalog.yaml and motif.yaml):
//
//	imp.Motif = "ghcr.io/orkspace/orkestra-registry/postgres@v16"
//	imp.OCI   = true
func LoadImport(imp *orktypes.MotifImport) (*orktypes.Motif, error) {
	ref := imp.Motif

	// File path — relative, absolute, or ends with .yaml/.yml
	if isFilePath(ref) {
		return Load(ref)
	}

	// Registry pull — parse url@version shorthand (mirrors RegistrySource.ResolvedURL)
	cleanURL, version := resolveRef(ref, imp.Version, imp.OCI)

	// Resolve auth credentials from environment variables (same as merger auth)
	auth, err := imp.Auth.Resolve()
	if err != nil {
		return nil, fmt.Errorf("motif %q: auth: %w", ref, err)
	}

	// Pull to temp directory — dedicated Motif pull fetches only motif.yaml for Git
	// repos; OCI pull fetches the full artifact (motif.yaml must be at the root)
	tmpDir, cleanup, err := merger.PullMotifToDir(cleanURL, version, imp.OCI, auth)
	if err != nil {
		return nil, fmt.Errorf("motif %q@%s: pull failed: %w", cleanURL, version, err)
	}
	defer cleanup()

	// Motif artifacts contain motif.yaml at the root
	data, err := os.ReadFile(filepath.Join(tmpDir, "motif.yaml"))
	if err != nil {
		return nil, fmt.Errorf("motif %q@%s: motif.yaml not found in artifact: %w", cleanURL, version, err)
	}

	return parse(data)
}

// isFilePath reports whether ref is a local file reference.
func isFilePath(ref string) bool {
	if strings.HasPrefix(ref, "./") || strings.HasPrefix(ref, "/") || strings.HasPrefix(ref, "../") {
		return true
	}
	return strings.HasSuffix(ref, ".yaml") || strings.HasSuffix(ref, ".yml")
}

// resolveRef parses a Motif ref into (cleanURL, version).
// Mirrors RegistrySource.ResolvedURL exactly.
func resolveRef(ref, version string, oci bool) (cleanURL, resolvedVersion string) {
	if idx := strings.LastIndex(ref, "@"); idx != -1 {
		return ref[:idx], ref[idx+1:]
	}
	cleanURL = ref
	if version != "" {
		return cleanURL, version
	}
	if oci {
		return cleanURL, "latest"
	}
	return cleanURL, "main"
}

func parse(data []byte) (*orktypes.Motif, error) {
	var m orktypes.Motif
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing motif: %w", err)
	}
	if m.Kind != "Motif" {
		return nil, fmt.Errorf("expected kind: Motif, got: %s", m.Kind)
	}
	if m.Metadata.Name == "" {
		return nil, fmt.Errorf("motif metadata.name is required")
	}
	return &m, nil
}
