// pkg/reconciler/run_mutation.go
package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// MutationResult holds the outcome of applying mutation rules.
type MutationResult struct {
	// Applied — number of rules that changed a field value
	Applied int

	// Changes — one entry per field that was mutated
	Changes []MutationChange
}

// MutationChange describes one field mutation that was applied.
type MutationChange struct {
	Field    string
	OldValue string
	NewValue string
	Type     string // "default" or "override"
}

// RunMutation applies mutation rules to the CR.
//
// For each rule:
//   - "default" rules: set the field only if it is currently absent or empty
//   - "override" rules: always set the field, regardless of current value
//
// After evaluating all rules, if any changes were computed, a single
// merge patch is applied via the Kubernetes API. The caller should
// re-read the object from the informer cache after mutation.
//
// Returns a MutationResult. A non-zero Applied count means the CR was patched
// and the caller should re-read the object before proceeding.
func RunMutation(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	obj domain.Object,
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
		// Typed objects — cannot apply dot-notation mutations.
		// Use Go hooks for typed mutation.
		logger.Debug().
			Str("crd", crdName).
			Msg("mutation: typed object — skipping declarative mutation (use Go hooks)")
		return result, nil
	}

	// Build template resolver for resolving Override/Default template expressions
	resolver, err := orktmpl.NewResolver(ctx, obj)
	if err != nil {
		return nil, fmt.Errorf("mutation: building resolver: %w", err)
	}

	// Build the patch map — only fields that need changing
	patch := map[string]interface{}{}
	hasPatch := false

	for _, rule := range cfg.Rules {
		currentVal, found := resolveField(u.Object, rule.Field)

		// Determine what value to set
		var newVal string
		var mutationType string

		if rule.Override != "" {
			// Override — always set, regardless of current value
			resolved, err := resolver.Resolve(rule.Override)
			if err != nil {
				return nil, fmt.Errorf("mutation: resolving override for field %q: %w", rule.Field, err)
			}
			newVal = resolved
			mutationType = "override"
		} else if rule.Default != "" {
			// Default — set only if field is absent or empty
			if found && currentVal != "" {
				continue // field already has a value, skip
			}
			resolved, err := resolver.Resolve(rule.Default)
			if err != nil {
				return nil, fmt.Errorf("mutation: resolving default for field %q: %w", rule.Field, err)
			}
			newVal = resolved
			mutationType = "default"
		} else {
			continue // rule has neither Default nor Override — skip
		}

		// Skip if the value hasn't changed — avoid unnecessary patches
		if newVal == currentVal {
			continue
		}

		// Build the nested patch path for this field
		setNestedPatch(patch, rule.Field, newVal)
		hasPatch = true

		result.Changes = append(result.Changes, MutationChange{
			Field:    rule.Field,
			OldValue: currentVal,
			NewValue: newVal,
			Type:     mutationType,
		})

		// Per-field metric
		metrics.RecordMutationFieldDetail(crdName, rule.Field, mutationType)

		logger.Debug().
			Str("crd", crdName).
			Str("name", obj.GetName()).
			Str("field", rule.Field).
			Str("old", currentVal).
			Str("new", newVal).
			Str("type", mutationType).
			Msg("mutation: applying rule")
	}

	if !hasPatch {
		return result, nil // no changes needed
	}

	// Wrap in metadata.annotations path is wrong — the patch IS the spec patch
	// The patch needs to wrap the actual field paths
	// For spec.* fields the patch is {"spec": {"replicas": "1"}}
	// setNestedPatch already builds this correctly

	data, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("mutation: marshalling patch: %w", err)
	}

	// Apply via merge patch
	var resource interface {
		Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*unstructured.Unstructured, error)
	}

	ns := obj.GetNamespace()
	if ns != "" {
		resource = kube.Dynamic().Resource(gvr).Namespace(ns).(interface {
			Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*unstructured.Unstructured, error)
		})
	} else {
		resource = kube.Dynamic().Resource(gvr).(interface {
			Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*unstructured.Unstructured, error)
		})
	}

	_, patchErr := resource.Patch(
		ctx,
		obj.GetName(),
		types.MergePatchType,
		data,
		metav1.PatchOptions{},
	)
	if patchErr != nil {
		if errors.IsConflict(patchErr) {
			// Conflict — the object was updated between our read and our patch.
			// Return without error — the workqueue will re-queue and we'll retry.
			logger.Debug().
				Str("crd", crdName).
				Str("name", obj.GetName()).
				Msg("mutation: conflict on patch — will retry on next reconcile")
			return result, nil
		}
		return nil, fmt.Errorf("mutation: patching %s/%s: %w", ns, obj.GetName(), patchErr)
	}

	result.Applied = len(result.Changes)

	// Aggregate metric — one per reconcile where mutations were applied
	metrics.RecordMutationTotal(crdName)

	logger.Info().
		Str("crd", crdName).
		Str("name", obj.GetName()).
		Int("fieldsChanged", result.Applied).
		Msg("mutation: rules applied")

	return result, nil
}

// setNestedPatch sets a value at a dot-notation path in a nested map.
// Creates intermediate maps as needed.
// "spec.replicas" with value "1" produces: {"spec": {"replicas": "1"}}
func setNestedPatch(patch map[string]interface{}, path string, value string) {
	parts := strings.Split(path, ".")
	current := patch

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}
		if _, ok := current[part]; !ok {
			current[part] = map[string]interface{}{}
		}
		next, ok := current[part].(map[string]interface{})
		if !ok {
			// Overwrite non-map with map — shouldn't happen in practice
			current[part] = map[string]interface{}{}
			next = current[part].(map[string]interface{})
		}
		current = next
	}
}
