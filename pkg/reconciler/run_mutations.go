// pkg/reconciler/run_mutation.go
package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/metrics"
	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// MutationResult holds the outcome of applying mutation rules.
type MutationResult struct {
	Applied int
	Changes []MutationChange
}

// MutationChange describes one field mutation that was applied.
type MutationChange struct {
	Field    string
	OldValue string
	NewValue interface{}
	Type     string // "default" or "override"
}

// runMutation applies mutation rules to the CR and patches it if any values changed.
//
// Rule semantics:
//   - default:  set the field only when it is absent or empty in the CR
//   - override: always set the field, regardless of current value
//
// Type preservation:
//
//	YAML `default: 2`    → int64(2)  in patch → API server receives integer
//	YAML `default: "2"`  → string    in patch → use only for string fields
//	YAML `default: true` → bool      in patch → API server receives boolean
//
//	Template expressions (containing "{{") always produce strings — use them
//	only on string fields. For typed fields (int, bool), use literal YAML values.
//
// Mutation failures are non-fatal — the reconcile continues and the next cycle
// retries. The object is not patched if no rules produce a change.
func runMutation(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	obj domain.Object,
	resolver *orktmpl.Resolver,
	cfg *orktypes.MutationConfig,
	gvr schema.GroupVersionResource,
	crdName string,
) (*MutationResult, error) {
	result := &MutationResult{}

	if cfg == nil || len(cfg.Rules) == 0 {
		return result, nil
	}

	u, ok := toUnstructured(obj)
	if !ok {
		logger.Debug().
			Str("crd", crdName).
			Msg("mutation: typed object — skipping declarative mutation (use Go hooks)")
		return result, nil
	}

	var err error

	// patch accumulates all field changes.
	// Keys are the top-level field names (e.g. "spec").
	// setNestedPatch builds the nested structure from dot-notation paths.
	patch := map[string]interface{}{}
	hasPatch := false

	for _, rule := range cfg.Rules {
		// currentVal is the string representation of the current field value.
		// resolveField uses anyToString — so integers come back as "2", bools as "true".
		currentVal, found := resolveField(u.Object, rule.Field)

		// ── Determine the mutation type and raw desired value ─────────────────
		// rawVal preserves the YAML-native type (int64, bool, string).
		// mutationType is recorded for metrics and change log.
		var rawVal interface{}
		var mutationType string

		rawVal, mutationType, err = resolveRuleValue(rule, found, currentVal, resolver)
		if err != nil {
			return nil, fmt.Errorf("mutation: field %q: %w", rule.Field, err)
		}
		if rawVal == nil {
			continue // rule did not apply (default with existing value, or no value declared)
		}

		// ── Skip if unchanged ─────────────────────────────────────────────────
		// Compare as strings — currentVal is already a string from resolveField.
		// fmt.Sprintf("%v", rawVal) gives "2" for int64(2), "true" for bool(true).
		if fmt.Sprintf("%v", rawVal) == currentVal {
			continue
		}

		// ── Accumulate patch ──────────────────────────────────────────────────
		setNestedPatch(patch, rule.Field, rawVal)
		hasPatch = true

		result.Changes = append(result.Changes, MutationChange{
			Field:    rule.Field,
			OldValue: currentVal,
			NewValue: rawVal,
			Type:     mutationType,
		})

		metrics.RecordMutationFieldDetail(crdName, rule.Field, mutationType)

		logger.Debug().
			Str("crd", crdName).
			Str("name", obj.GetName()).
			Str("field", rule.Field).
			Str("old", currentVal).
			Str("new", fmt.Sprintf("%v", rawVal)).
			Str("type", mutationType).
			Msg("mutation: rule applied")
	}

	if !hasPatch {
		return result, nil
	}

	// ── Apply patch ───────────────────────────────────────────────────────────
	data, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("mutation: marshalling patch: %w", err)
	}

	ns := obj.GetNamespace()
	_, patchErr := kube.DynamicClient().
		Resource(gvr).
		Namespace(ns).
		Patch(ctx, obj.GetName(), types.MergePatchType, data, metav1.PatchOptions{})

	if patchErr != nil {
		if errors.IsConflict(patchErr) {
			logger.Debug().
				Str("crd", crdName).
				Str("name", obj.GetName()).
				Msg("mutation: resource version conflict — will retry on next reconcile")
			return result, nil
		}
		return nil, fmt.Errorf("mutation: patching %s/%s: %w", ns, obj.GetName(), patchErr)
	}

	result.Applied = len(result.Changes)
	metrics.RecordMutationTotal(crdName)

	logger.Info().
		Str("crd", crdName).
		Str("name", obj.GetName()).
		Int("fieldsChanged", result.Applied).
		Msg("mutation: rules applied")

	return result, nil
}

// resolveRuleValue determines the value to set for one mutation rule.
//
// Returns:
//   - rawVal:       the typed value to write (int64, bool, string, or nil)
//   - mutationType: "default" or "override" for metrics
//   - err:          non-nil if a template expression failed to resolve
//
// Returns (nil, "", nil) when the rule does not apply — caller should skip.
func resolveRuleValue(
	rule orktypes.MutationRule,
	found bool,
	currentVal string,
	resolver *orktmpl.Resolver,
) (rawVal interface{}, mutationType string, err error) {

	switch {
	case rule.Override != nil:
		// Override — always apply, regardless of current value
		mutationType = "override"
		rawVal, err = resolveTypedValue(rule.Override, resolver)

	case rule.Default != nil:
		// Default — apply only when the field is absent or empty
		if found && currentVal != "" {
			return nil, "", nil // field already has a value — skip
		}
		mutationType = "default"
		rawVal, err = resolveTypedValue(rule.Default, resolver)

	default:
		return nil, "", nil // rule declares neither Default nor Override
	}

	return rawVal, mutationType, err
}

// resolveTypedValue converts a mutation rule value to its final form.
//
// If the value is a string containing "{{", it is treated as a template
// expression and resolved against the CR. The result is always a string.
//
// If the value is a string without "{{", it is returned as-is.
//
// If the value is a non-string native YAML type (int64, bool, float64),
// it is returned directly — preserving the type for the JSON patch.
// The API server receives an integer, not a string.
func resolveTypedValue(val interface{}, resolver *orktmpl.Resolver) (interface{}, error) {
	strVal, isStr := val.(string)
	if !isStr {
		// Native YAML type — int64, bool, float64
		// Return directly: JSON marshal will produce 2, true, 3.14 — not "2", "true", "3.14"
		return val, nil
	}

	// String value — check for template expression
	if !strings.Contains(strVal, "{{") {
		// Static string — return as-is
		return strVal, nil
	}

	// Template expression — resolve against the CR
	// Template expressions always produce strings. Only use them on string fields.
	resolved, err := resolver.Resolve(strVal)
	if err != nil {
		return nil, fmt.Errorf("resolving template expression %q: %w", strVal, err)
	}
	return resolved, nil
}

// setNestedPatch sets a value at a dot-notation path inside a patch map,
// creating intermediate maps as needed.
//
// "spec.replicas" with int64(2)   → {"spec": {"replicas": 2}}
// "spec.port"     with int64(8080) → {"spec": {"port": 8080}}
// "spec.env"      with "production" → {"spec": {"env": "production"}}
//
// The native Go type is preserved — JSON marshal produces the correct
// JSON type for the Kubernetes API server.
func setNestedPatch(patch map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := patch

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value // preserve native type
			return
		}
		if _, ok := current[part]; !ok {
			current[part] = map[string]interface{}{}
		}
		next, ok := current[part].(map[string]interface{})
		if !ok {
			// Existing value is not a map — replace with map
			current[part] = map[string]interface{}{}
			next = current[part].(map[string]interface{})
		}
		current = next
	}
}

// toUnstructured attempts to cast a domain.Object to *unstructured.Unstructured.
// Returns false for typed objects — declarative mutation is not supported for typed CRDs.
// func toUnstructured(obj domain.Object) (*unstructured.Unstructured, bool) {
// 	u, ok := obj.(*unstructured.Unstructured)
// 	return u, ok
// }
