package applyapi

import (
	"fmt"
	"strings"

	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
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
// Returns an error only when idp.name or idp.namespace are declared but
// cannot be resolved to a non-empty string — SSA would reject the CR anyway,
// so we surface the error here with a clearer message.
func BuildCRFromTarget(
	raw map[string]interface{},
	crd *orktypes.CRDEntry,
	notes orktypes.NoteRegistry,
) (*unstructured.Unstructured, error) {

	obj := &unstructured.Unstructured{
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

	meta := obj.Object["metadata"].(map[string]interface{})
	labels := meta["labels"].(map[string]interface{})
	annotations := meta["annotations"].(map[string]interface{})
	spec := obj.Object["spec"].(map[string]interface{})

	// Build O(1) destination sets from the IDP field declarations.
	specFields := make(map[string]struct{}, len(crd.IDP.Fields))
	for name := range crd.IDP.Fields {
		specFields[name] = struct{}{}
	}
	labelFields := crd.AdditionalLabelFields()
	annotationFields := crd.AdditionalAnnotationFields()

	// Route each submitted value to its declared destination.
	for key, value := range raw {
		switch key {
		case "target":
			// The routing key itself is not a field value.
			continue
		}

		switch {
		case setContains(specFields, key):
			spec[key] = value

		case mapContains(labelFields, key):
			// Labels must be strings. Convert with Sprintf for safety.
			labels[key] = fmt.Sprintf("%v", value)

		case mapContains(annotationFields, key):
			annotations[key] = fmt.Sprintf("%v", value)

			// Unknown field — silently ignored. The caller may pass extra keys;
			// only declared fields are meaningful. Callers receive field shapes
			// from the schema API so unknown fields indicate a client bug, not a
			// server error.
		}
	}

	// Resolve idp.name and idp.namespace using the flat field map as data.
	// Notes are available so expressions like `{{ repoSlug .repository }}`
	// can call the same note library used in Katalog status fields.
	resolver := orktmpl.NewResolverFromMap(raw).WithUserNotes(notes)

	if crd.HasIDPName() {
		name, err := resolver.Resolve(crd.IDP.Name)
		if err != nil || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf(
				"idp.name expression %q could not be resolved — "+
					"check that the required fields are present in the request: %w",
				crd.IDP.Name, err,
			)
		}
		obj.SetName(strings.TrimSpace(name))
	}

	if crd.HasIDPNamespace() {
		ns, err := resolver.Resolve(crd.IDP.Namespace)
		if err != nil || strings.TrimSpace(ns) == "" {
			return nil, fmt.Errorf(
				"idp.namespace expression %q could not be resolved — "+
					"check that the required fields are present in the request: %w",
				crd.IDP.Namespace, err,
			)
		}
		obj.SetNamespace(strings.TrimSpace(ns))
	}

	return obj, nil
}

// setContains is a nil-safe struct{} map membership check.
func setContains(s map[string]struct{}, key string) bool {
	_, ok := s[key]
	return ok
}

// mapContains is a nil-safe IDPFieldConfig map membership check.
func mapContains(m map[string]orktypes.IDPFieldConfig, key string) bool {
	if m == nil {
		return false
	}
	_, ok := m[key]
	return ok
}
