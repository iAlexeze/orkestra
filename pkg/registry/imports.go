// pkg/registry/imports.go
//
// Extracts OCI references from a katalog or komposer file so they can be
// pre-pulled before running validate, generate, or run commands.
package registry

import (
	"fmt"
	"os"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"gopkg.in/yaml.v3"
)

// OCIImports holds all OCI references found in a katalog or komposer file.
type OCIImports struct {
	MotifImports    []orktypes.MotifImport
	RegistrySources []orktypes.RegistrySource
}

// Empty returns true when there are no OCI refs to pull.
func (o *OCIImports) Empty() bool {
	return len(o.MotifImports) == 0 && len(o.RegistrySources) == 0
}

// ExtractOCIImports parses a katalog or komposer file and returns all OCI
// references that must be pre-pulled before validate/generate/run.
//
// For Katalog files: spec.crds[*].imports[*].motif refs that resolve to OCI.
// For Komposer files: imports.registry[*] sources where oci: true or the URL
// has an oci:// prefix.
//
// Motif shorthands (bare names like "postgres" or "postgres:v0.1.0") are
// treated as OCI — they resolve against the default motif registry, same
// as the LoadImport resolution order.
func ExtractOCIImports(filePath string) (*OCIImports, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", filePath, err)
	}

	var kf orktypes.KatalogFile
	if err := yaml.Unmarshal(data, &kf); err != nil {
		return nil, fmt.Errorf("parsing %q: %w", filePath, err)
	}

	out := &OCIImports{}

	// Motif imports from spec.crds[*].imports (Katalog and inline Komposer CRDs)
	for _, crd := range kf.Spec.CRDs {
		for _, imp := range crd.Imports {
			if isOCIMotifImport(imp) {
				out.MotifImports = append(out.MotifImports, imp)
			}
		}
	}

	// Registry imports from imports.registry (Komposer)
	if kf.Imports != nil {
		for _, src := range kf.Imports.Registry {
			if isOCIRegistrySource(src) {
				out.RegistrySources = append(out.RegistrySources, src)
			}
		}
	}

	return out, nil
}

// LocalMotifImport describes a motif import in a katalog that uses a local file path.
// Local imports are valid for development (ork simulate, ork template) but cannot
// be resolved by consumers after the katalog is published.
type LocalMotifImport struct {
	CRDName string
	Index   int
	Path    string
}

// ExtractLocalMotifImports parses a katalog.yaml and returns any motif imports
// that reference local file paths. The caller should block ork push when the
// result is non-empty and prompt the user to replace them with OCI refs.
func ExtractLocalMotifImports(filePath string) ([]LocalMotifImport, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", filePath, err)
	}
	var kf orktypes.KatalogFile
	if err := yaml.Unmarshal(data, &kf); err != nil {
		return nil, fmt.Errorf("parsing %q: %w", filePath, err)
	}
	var out []LocalMotifImport
	for crdName, crd := range kf.Spec.CRDs {
		for i, imp := range crd.Imports {
			if ref := strings.TrimSpace(imp.Motif); isMotifFilePath(ref) {
				out = append(out, LocalMotifImport{CRDName: crdName, Index: i, Path: ref})
			}
		}
	}
	return out, nil
}

// isOCIMotifImport mirrors the resolution order in pkg/motif.LoadImport:
//  1. File path → not OCI
//  2. Git URL → not OCI
//  3. oci:// prefix → OCI
//  4. oci: true → OCI
//  5. Bare name (no dots in host, not a git URL) → OCI (resolved against default motif registry)
//  6. Full ref without oci: true → not OCI (requires explicit oci flag)
func isOCIMotifImport(imp orktypes.MotifImport) bool {
	ref := strings.TrimSpace(imp.Motif)
	if ref == "" {
		return false
	}
	if isMotifFilePath(ref) || isMotifGitURL(ref) {
		return false
	}
	if strings.HasPrefix(ref, "oci://") || imp.OCI {
		return true
	}
	// Bare name: no dots in the host segment before the first slash, not a full ref.
	// e.g. "postgres", "postgres:v0.1.0" — resolves against default motif registry.
	return !looksLikeFull(ref)
}

// isOCIRegistrySource returns true when a RegistrySource resolves via OCI.
// Explicit oci: true flag or an oci:// prefix on the URL both count.
func isOCIRegistrySource(src orktypes.RegistrySource) bool {
	return src.OCI || strings.HasPrefix(strings.TrimSpace(src.URL), "oci://")
}

// isMotifFilePath mirrors isFilePath in pkg/motif/loader.go.
func isMotifFilePath(ref string) bool {
	if strings.HasPrefix(ref, "./") || strings.HasPrefix(ref, "/") || strings.HasPrefix(ref, "../") {
		return true
	}
	return strings.HasSuffix(ref, ".yaml") || strings.HasSuffix(ref, ".yml")
}

// isMotifGitURL mirrors isGitURL in pkg/motif/loader.go.
func isMotifGitURL(ref string) bool {
	return strings.HasPrefix(ref, "https://") ||
		strings.HasPrefix(ref, "http://") ||
		strings.HasPrefix(ref, "git@")
}
