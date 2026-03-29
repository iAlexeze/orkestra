package reconciler

import (
	"context"
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/pkg/logger"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// applyReconcileTimeValidation evaluates validation rules against the live CR.
// deny rules return an error — reconcile halts and the error is recorded.
// warn rules record an active warning on the health API — reconcile continues.
func (r *GenericReconciler[T]) applyReconcileTimeValidation(ctx context.Context, obj T) error {
    if r.rc.Validation == nil || len(r.rc.Validation.Rules) == 0 {
        return nil
    }

    // Build the object map from the CR — same approach as admission handler
    u, ok := any(obj).(*unstructured.Unstructured)
    if !ok {
        return nil // typed objects — validation not supported without unstructured
    }

    denials, warnings := evaluateValidationRulesMap(u.Object, r.rc.Validation)

    // Record warn violations — they surface on the health API
    for _, w := range warnings {
        logger.FromContext(ctx).Warn().
            Str("name", obj.GetName()).
            Str("field", w.Field).
            Str("message", w.Message).
            Msg("reconcile validation: warn")
        // TODO: record to active warnings on CRDHealth for /katalog/{crd}
    }

    if len(denials) > 0 {
        msgs := make([]string, 0, len(denials))
        for _, d := range denials {
            msgs = append(msgs, fmt.Sprintf("field %q: %s", d.Field, d.Message))
        }
        return fmt.Errorf("validation denied: %s", strings.Join(msgs, "; "))
    }

    return nil
}

// applyReconcileTimeMutation applies mutation defaults to the CR and patches
// the spec subresource when changes are needed.
// Mutation failures are non-fatal — logged and skipped.
func (r *GenericReconciler[T]) applyReconcileTimeMutation(ctx context.Context, obj T) error {
    if r.rc.Mutation == nil || len(r.rc.Mutation.Rules) == 0 {
        return nil
    }

    u, ok := any(obj).(*unstructured.Unstructured)
    if !ok {
        return nil
    }

    // Work on a copy — compare to find what changed
    original := u.DeepCopy().Object
    mutated := deepCopyMapReconcile(u.Object)

    resolver := orktmpl.NewResolverFromMap(mutated)
    changed := false

    for _, rule := range r.rc.Mutation.Rules {
        currentVal, found := resolveFieldPathReconcile(mutated, rule.Field)

        var newVal string

        switch {
        case rule.Override != "":
            resolved, err := resolver.Resolve(rule.Override)
            if err != nil {
                continue
            }
            newVal = resolved

        case rule.Default != "":
            if found && currentVal != "" {
                continue // field already has a value
            }
            resolved, err := resolver.Resolve(rule.Default)
            if err != nil {
                continue
            }
            newVal = resolved

        default:
            continue
        }

        if newVal == currentVal {
            continue
        }

        setNestedPatch(mutated, rule.Field, newVal)
        changed = true

        logger.FromContext(ctx).Debug().
            Str("name", obj.GetName()).
            Str("field", rule.Field).
            Str("was", currentVal).
            Str("now", newVal).
            Msg("reconcile mutation: applied")
    }

    if !changed {
        return nil
    }

    // Patch the CR spec with the mutated values.
    // We only patch fields that changed — build a minimal merge patch.
    specPatch := buildSpecPatch(original, mutated)
    if len(specPatch) == 0 {
        return nil
    }

    return r.kube.PatchSpec(ctx, obj, r.crd.GVR, specPatch)
}

func buildSpecPatch(original, mutated map[string]interface{}) map[string]interface{} {
    origSpec, _ := original["spec"].(map[string]interface{})
    mutSpec, _ := mutated["spec"].(map[string]interface{})
    if origSpec == nil || mutSpec == nil {
        return nil
    }

    patch := map[string]interface{}{}
    for k, mv := range mutSpec {
        ov, exists := origSpec[k]
        if !exists || fmt.Sprintf("%v", ov) != fmt.Sprintf("%v", mv) {
            patch[k] = mv
        }
    }
    return patch
}