package katalog

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

// validateIDPFieldPaths validates idp.fields path configurations:
//   - Paths (if set) must be unique across all fields
//   - Path must be a valid dot-notation path (no empty segments, no leading/trailing dots)
//   - Warn if path is nested (contains a dot) — schema existence validation is not yet implemented
func (k *Katalog) validateIDPFieldPaths() error {
	for crdName, crd := range k.enabledCRDs {
		if !crd.IDPEnabled() {
			continue
		}

		if !crd.HasIDPFields() {
			continue
		}

		seenPaths := make(map[string]bool)

		for name, config := range crd.IDP.Fields {
			// 1. Skip if no path is set (flat field)
			if !config.HasSpecPath() {
				continue
			}

			specPath := config.SpecPath(name)

			// 2. Validate the path format (no empty segments)
			if err := validatePathFormat(specPath); err != nil {
				return fmt.Errorf(
					"CRD %q: idp.fields %q has invalid path %q: %w",
					crdName, name, specPath, err,
				)
			}

			// 3. Check for duplicate spec paths
			if seenPaths[specPath] {
				return errDuplicateIDPFieldPath(crdName, specPath, name)
			}
			seenPaths[specPath] = true

			// 4. If path is nested, warn that validation against the CRD
			//    schema is not yet implemented (to be added later)
			if isNestedPath(specPath) {
				crd.Warnings.AddWarning(fmt.Sprintf(
					"CRD %q: idp.fields %q has nested path %q — "+
						"schema validation is not yet available. Verify the path exists "+
						"in the CRD spec (e.g., 'spec.%s') and that it is a leaf field.",
					crdName, name, specPath, specPath,
				))
			}
		}
	}
	return nil
}

// validatePathFormat ensures the path is a valid dot-notation path:
//   - No empty segments (e.g., "app..repository")
//   - No leading or trailing dots
//   - Each segment is a valid Kubernetes qualified name
func validatePathFormat(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".") {
		return fmt.Errorf("path cannot start or end with a dot")
	}

	parts := strings.Split(path, ".")
	for _, part := range parts {
		if part == "" {
			return fmt.Errorf("path contains empty segment (double dot)")
		}
		if errs := validation.IsQualifiedName(part); len(errs) > 0 {
			return fmt.Errorf("path segment %q is not a valid Kubernetes name: %s", part, strings.Join(errs, "; "))
		}
	}

	return nil
}

func errDuplicateIDPFieldPath(crd, path, field string) error {
	return fmt.Errorf(`
──────────────────────────────────────────────
%s Duplicate idp.fields path: %q
   CRD: %s
   Used by field: %q

Each idp.fields entry must have a unique spec path — two fields cannot map
to the same location in the CRD spec.

If you need two fields to represent the same spec path, consider using a
single field with an enum or conditional logic.
──────────────────────────────────────────────`, failureMark(), path, crd, field)
}
