package reconciler

import (
	"context"
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/logger"
	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// applyReconcileTimeValidation evaluates validation rules against the live CR.
// Deny rules return an error — reconcile halts and the error is recorded.
// Warn rules log an advisory — reconcile continues.
func (r *GenericReconciler[PTR]) applyReconcileTimeValidation(ctx context.Context, obj PTR) error {
	if r.crd.Validation == nil || len(r.crd.Validation.Rules) == 0 {
		return nil
	}

	result := runValidation(obj, r.crd.Validation, r.crd.APITypes.Kind)
	if result.Passed {
		return nil
	}

	var denials []ValidationViolation

	for _, v := range result.Violations {
		action := validationRuleAction(r.crd.Validation, v.Field, v.Rule)
		if action.IsWarn() {
			logger.FromContext(ctx).Warn().
				Str("name", obj.GetName()).
				Str("field", v.Field).
				Str("message", v.Message).
				Msg("reconcile validation: warn")
		} else {
			denials = append(denials, v)
		}
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
