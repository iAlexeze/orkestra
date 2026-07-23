package reconciler

import (
	"context"

	"github.com/orkspace/orkestra/pkg/logger"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// applyReconcileTimeValidation evaluates validation rules against the live CR.
// Warn violations are logged as advisory — reconcile continues.
// Deny violations halt reconcile — the caller patches status and returns the error.
// Always returns the ValidationResult so the caller can pass it to patchStatusWithChildren.
func (r *GenericReconciler[PTR]) applyReconcileTimeValidation(ctx context.Context, resolver *orktmpl.Resolver, obj PTR) (*ValidationResult, error) {
	if r.crd.Validation == nil || len(r.crd.Validation.Rules) == 0 {
		return nil, nil
	}

	result := runValidation(resolver.Data(), resolver, r.crd.Validation, r.crd.APITypes.Kind)

	for _, w := range result.Warnings {
		logger.FromContext(ctx).Warn().
			Str("name", obj.GetName()).
			Str("crd", r.crd.GVKString()).
			Str("field", w.Field).
			Str("message", w.Message).
			Msg("reconcile validation: warn")
	}

	if result.Deny {
		return result, result.DenialError()
	}

	return result, nil
}

// applyReconcileTimeMutation applies mutation defaults to the CR and patches
// the spec subresource when changes are needed.
// Mutation failures are non-fatal — the caller logs and continues.
func (r *GenericReconciler[PTR]) applyReconcileTimeMutation(ctx context.Context, resolver *orktmpl.Resolver, obj PTR) error {
	if r.crd.Mutation == nil || len(r.crd.Mutation.Rules) == 0 {
		return nil
	}

	_, err := runMutation(ctx, r.kube, obj, resolver, r.crd.Mutation, r.crd.GVR(), r.crd.APITypes.Kind)
	return err
}

// validationRuleAction looks up the declared action for a rule by its field
// and rule type string. Returns "" (deny) when no match is found — this is the
// fail-safe default: unknown or ambiguous rules block rather than warn.
func validationRuleAction(cfg *orktypes.ValidationConfig, field, rt string) orktypes.ValidationAction {
	for _, rule := range cfg.Rules {
		if rule.Field == field && ruleType(rule) == rt {
			return rule.Action
		}
	}
	return "" // default: deny
}
