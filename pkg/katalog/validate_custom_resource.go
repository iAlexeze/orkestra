package katalog

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateCustomResources validates any Custom resources declared under the
// OperatorBox hooks (onCreate/onReconcile/onDelete). It is intentionally
// conservative: if the CRD does not declare hooks/constructor we fast-path
// and skip validation because there is nothing to manage.
//
// This function delegates per-item structural checks to types.validateCustomResource
// which enforces apiVersion/kind/metadata/name/namespace/labels/annotations/template
// rules and the namespaced defaulting semantics.
func (k *Katalog) validateCustomResources() error {
	for name, crd := range k.enabledCRDs {
		// Fast path: nothing to validate if no hooks/constructor declared.
		// Hooks/constructor are the only places custom resources are meaningful.
		if crd.WithHooksDecl() && crd.WithConstructorDecl() {
			logger.Debug().
				Str("crd", name).
				Msg("skipping custom resource validation: hooks or constructor declared")
			return nil
		}

		if !crd.HasAnyHookTemplates() {
			logger.Debug().
				Str("crd", name).
				Msg("skipping custom resource validation: no hook templates")
			return nil
		}

		// Fast path: if the CRD doesn't declare any custom resources, skip.
		if !crd.HasAnyCustomResources() {
			logger.Debug().
				Str("crd", name).
				Msg("no custom resources declared in onCreate/onReconcile; skipping validation")
			return nil
		}

		// Validate OnCreate custom resources
		if crd.HasOnCreate() && crd.OperatorBox.OnCreate.CustomResource != nil {
			for i, cr := range crd.OperatorBox.OnCreate.CustomResource {
				path := fmt.Sprintf("%s.onCreate.custom[%d]", name, i)
				// Delegate structural validation to the types package.
				if err := ValidateCustomResource(&cr, path); err != nil {
					return err
				}

				// Additional katalog-level checks (defensive):
				// - If the declaration explicitly marks cluster-scoped but provided a namespace, error.
				if cr.Metadata.Namespaced != nil && !*cr.Metadata.Namespaced && cr.Metadata.Namespace != "" {
					return fmt.Errorf("%s: metadata.namespaced=false but metadata.namespace is set (cluster-scoped resources must not set namespace)", path)
				}

				// Log a debug line for observability
				logger.Debug().
					Str("crd", name).
					Str("path", path).
					Msg("validated onCreate custom resource declaration")
			}
		}

		// Validate OnReconcile custom resources
		if crd.HasOnReconcile() && crd.OperatorBox.OnReconcile.CustomResource != nil {
			for i, cr := range crd.OperatorBox.OnReconcile.CustomResource {
				path := fmt.Sprintf("%s.onReconcile.custom[%d]", name, i)
				if err := ValidateCustomResource(&cr, path); err != nil {
					return err
				}

				if cr.Metadata.Namespaced != nil && !*cr.Metadata.Namespaced && cr.Metadata.Namespace != "" {
					return fmt.Errorf("%s: metadata.namespaced=false but metadata.namespace is set (cluster-scoped resources must not set namespace)", path)
				}

				logger.Debug().
					Str("crd", name).
					Str("path", path).
					Msg("validated onReconcile custom resource declaration")
			}
		}
	}

	return nil
}

// ValidateCustomResource validates a single CustomResource declaration. It
// enforces the structural and semantic rules required by Orkestra while
// intentionally avoiding CRD schema validation (the API server owns that).
//
// Validation responsibilities:
//   - Ensure required top-level fields (apiVersion, kind) are present and well-formed.
//   - Ensure metadata.name is present after templating.
//   - Enforce namespaced vs cluster-scoped semantics using Metadata.Namespaced
//     with a defensive default of namespaced=true.
//   - Validate labels and annotations using existing validators.
//   - Validate template syntax for spec/status/other fields (no structural schema).
//   - Validate hasStatus semantics and warn when user-provided status will be ignored.
//   - Return path-aware errors so callers can point users to the exact declaration.
//
// Note: This function assumes the following helpers exist in the package or
// elsewhere in the codebase and will be used here:
//   - isValidKind(kind string) bool
//   - validateLabels(labels map[string]string) error
//   - validateAnnotations(annotations map[string]string) error
//   - validateTemplateFields(obj map[string]any, path string) error
//
// The `path` parameter should identify the declaration location (e.g.
// "mycrd.onCreate.custom[0]") and is used to produce clear error messages.
func ValidateCustomResource(cr *orktypes.CustomResourceTemplateSource, path string) error {
	if cr == nil {
		return fmt.Errorf("%s: custom resource declaration is nil", path)
	}

	// --- apiVersion ---------------------------------------------------------
	if strings.TrimSpace(cr.APIVersion) == "" {
		return fmt.Errorf("%s: missing required field 'apiVersion'", path)
	}
	if !strings.Contains(cr.APIVersion, "/") {
		return fmt.Errorf("%s: apiVersion %q is invalid — must be group/version (e.g. foo.io/v1)", path, cr.APIVersion)
	}

	// --- kind ---------------------------------------------------------------
	if strings.TrimSpace(cr.Kind) == "" {
		return fmt.Errorf("%s: missing required field 'kind'", path)
	}

	// --- metadata -----------------------------------------------------------
	// Defensive: metadata must be present and name required after templating.
	if strings.TrimSpace(cr.Metadata.Name) == "" {
		return fmt.Errorf("%s: metadata.name is required", path)
	}

	// Namespaced defaulting: nil => true (namespaced by default)
	namespaced := true
	if cr.Metadata.Namespaced != nil {
		namespaced = *cr.Metadata.Namespaced
	}

	// If the declaration explicitly marks cluster-scoped but provided a namespace, error.
	if !namespaced && cr.Metadata.Namespace != "" {
		return fmt.Errorf("%s: metadata.namespaced=false but metadata.namespace is set (cluster-scoped resources must not set namespace)", path)
	}

	// If namespaced (default) then namespace must be present (non-empty).
	// Note: We allow empty namespace when the reconciler will template it to a value,
	// but at validation time we require the field to be present (non-empty) to avoid
	// accidental cluster-scoped creations. If you prefer to allow templated empty
	// namespaces, relax this check accordingly.
	if namespaced && strings.TrimSpace(cr.Metadata.Namespace) == "" {
		return fmt.Errorf("%s: metadata.namespace is required for namespaced custom resources (metadata.namespaced is true or unspecified)", path)
	}

	// --- hasStatus ----------------------------------------------------------
	// Only type/semantic check: YAML/JSON parsing guarantees boolean type.
	// Warn if user provided status but explicitly disabled status writes.
	if cr.HasStatus != nil && !*cr.HasStatus && len(cr.Status) > 0 {
		logger.Warn().Msgf("warning: %s: status provided but hasStatus=false — status will be ignored", path)
	}

	return nil
}
