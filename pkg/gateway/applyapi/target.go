package applyapi

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/logger"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// isTargetRequest reports whether raw is a target-mode request.
//
// Detection rule: presence of "target" key, regardless of whether
// "apiVersion" is also present. This lets callers migrate incrementally
// by adding "target" without immediately removing Kubernetes fields.
func isTargetRequest(raw map[string]interface{}) bool {
	_, ok := raw["target"]
	return ok
}

// BuildCRFromTarget constructs a full Kubernetes CR from a flat field map
// submitted in target mode.
//
// Field routing — determined by where each field name is declared in the
// Katalog's IDP config:
//
//	idp.fields                        → spec.<field>
//	idp.additionalFields.labels       → metadata.labels[<field>]
//	idp.additionalFields.annotations  → metadata.annotations[<field>]
//	not declared anywhere             → silently ignored
//
// The resolver receives the flat field map so idp.name and idp.namespace
// expressions reference submitted values directly:
//
//	idp.name:      '{{ repoSlug .repository }}'  → .repository from the request
//	idp.namespace: '{{ .team }}-{{ .environment }}' → .team and .environment
//
// When idp.name isn't declared, the caller's own "name" field is used as-is,
// matching full CR mode's metadata.name behavior for the same case.
//
// Returns an error only when idp.name or idp.namespace are declared but
// cannot be resolved to a non-empty string — SSA would reject the CR anyway,
// so we surface the error here with a clearer message.
func BuildCRFromTarget(
	raw map[string]interface{},
	crd *orktypes.CRDEntry,
	notes orktypes.NoteRegistry,
) (*unstructured.Unstructured, error) {
	// 1. Build the CR skeleton
	obj := newCRSkeleton(crd)

	// 2. Route fields to their destinations
	routeFields(raw, crd, obj)

	// 3. Resolve idp.name and idp.namespace
	if err := resolveIDPIdentity(raw, crd, notes, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// newCRSkeleton creates a blank CR with the correct apiVersion, kind,
// and empty metadata/spec structures.
func newCRSkeleton(crd *orktypes.CRDEntry) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": crd.APIVersion(),
			"kind":       crd.Kind(),
			"metadata": map[string]interface{}{
				"labels":      map[string]interface{}{},
				"annotations": map[string]interface{}{},
			},
			"spec": map[string]interface{}{},
		},
	}
}

// routeFields routes each submitted field to its declared destination:
//   - idp.fields → spec (supports nested dot-paths via 'path' field)
//   - idp.additionalFields.labels → metadata.labels
//   - idp.additionalFields.annotations → metadata.annotations
//   - unknown fields are silently ignored
func routeFields(
	raw map[string]interface{},
	crd *orktypes.CRDEntry,
	obj *unstructured.Unstructured,
) {
	meta := obj.Object["metadata"].(map[string]interface{})
	labels := meta["labels"].(map[string]interface{})
	annotations := meta["annotations"].(map[string]interface{})
	spec := obj.Object["spec"].(map[string]interface{})

	// Build lookup maps
	//   - specPathLookup: field name → spec path (flat or nested)
	//   - labelFields: field name → config (for labels)
	//   - annotationFields: field name → config (for annotations)
	specPathLookup := make(map[string]string, len(crd.IDP.Fields))
	for name, config := range crd.IDP.Fields {
		specPathLookup[name] = config.SpecPath(name)
	}
	labelFields := crd.AdditionalLabelFields()
	annotationFields := crd.AdditionalAnnotationFields()

	// Route each submitted value to its declared destination.
	for key, value := range raw {
		if key == "target" {
			continue
		}

		// ─── Spec fields (supports nested via path) ──────────────────────
		if specPath, ok := specPathLookup[key]; ok {
			if utils.IsNestedPath(specPath) {
				// Nested path — set at the dot-notation path
				if err := utils.SetNestedPath(spec, specPath, value); err != nil {
					// Log error but continue — don't fail the request
					logger.Error().Err(err).
						Str("path", specPath).
						Msg("apply api: failed to set spec path")
					continue
				}
			} else {
				// Flat path — direct assignment
				spec[specPath] = value
			}
			continue
		}

		// ─── Labels ──────────────────────────────────────────────────────
		if utils.MapContains(labelFields, key) {
			labels[key] = fmt.Sprintf("%v", value)
			continue
		}

		// ─── Annotations ──────────────────────────────────────────────────
		if utils.MapContains(annotationFields, key) {
			annotations[key] = fmt.Sprintf("%v", value)
			continue
		}

		// Unknown field — silently ignored.
	}
}

// resolveIDPIdentity resolves idp.name and idp.namespace using the flat field map.
// Notes are available so expressions like `{{ repoSlug .repository }}` work.
func resolveIDPIdentity(
	raw map[string]interface{},
	crd *orktypes.CRDEntry,
	notes orktypes.NoteRegistry,
	obj *unstructured.Unstructured,
) error {
	// Build a resolver data map that exposes both:
	//   .repository  (flat, from the request — target mode)
	//   .spec.repository (nested — for expressions written the Kubernetes way)
	data := make(map[string]interface{}, len(raw)+1)
	for k, v := range raw {
		data[k] = v
	}
	// Merge the built CR so .spec.*, .metadata.* work too
	for k, v := range obj.Object {
		if _, exists := data[k]; !exists {
			data[k] = v
		}
	}

	resolver := orktmpl.NewResolverFromMap(data).WithUserNotes(notes)

	// Resolve idp.name and idp.namespace. Every field the expression
	// references must be present — ResolveStrict reports a missing field
	// even when the composite result is non-empty (e.g. "{{ .team }}-{{ .environment }}"
	// with both fields absent still renders "-", which a plain empty-string
	// check wouldn't catch).
	if crd.HasIDPName() {
		name, missing, err := resolver.ResolveStrict(crd.IDP.Name)
		if err != nil || missing || strings.TrimSpace(name) == "" {
			return fmt.Errorf(
				"idp.name expression %q could not be resolved — "+
					"check that the required fields are present in the request: %w",
				crd.IDP.Name, err,
			)
		}
		obj.SetName(strings.TrimSpace(name))
	} else if name, ok := raw["name"].(string); ok && strings.TrimSpace(name) != "" {
		// idp.name not declared — use the caller-supplied name, matching
		// full CR mode's behavior for the same case.
		obj.SetName(strings.TrimSpace(name))
	}

	if crd.HasIDPNamespace() {
		ns, missing, err := resolver.ResolveStrict(crd.IDP.Namespace)
		if err != nil || missing || strings.TrimSpace(ns) == "" {
			return fmt.Errorf(
				"idp.namespace expression %q could not be resolved — "+
					"check that the required fields are present in the request: %w",
				crd.IDP.Namespace, err,
			)
		}
		obj.SetNamespace(strings.TrimSpace(ns))
	}

	return nil
}
