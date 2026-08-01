package katalog

import (
	"fmt"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateAdmissionOperators rejects any validation.rules or mutation.rules
// entry — including their own when:/anyOf: gating conditions — that declares
// an operator: string outside the known ConditionOperator set. An unknown
// operator is otherwise silently skipped by the evaluator and the rule
// always passes; this is exactly how operator: in went unimplemented in
// validation.rules before evaluation was consolidated into
// pkg/types/validation_eval.go.
func (k *Katalog) validateAdmissionOperators() error {
	for _, crd := range k.enabledCRDs {
		if crd.HasValidationRules() {
			for _, rule := range crd.Validation.Rules {
				if err := checkKnownOperator(rule.Operator, crd.Name, rule.Field); err != nil {
					return err
				}
				if err := checkConditionOperators(rule.When, crd.Name, rule.Field); err != nil {
					return err
				}
				if err := checkConditionOperators(rule.AnyOf, crd.Name, rule.Field); err != nil {
					return err
				}
			}
		}

		if crd.HasMutationRules() {
			for _, rule := range crd.Mutation.Rules {
				if err := checkConditionOperators(rule.When, crd.Name, rule.Field); err != nil {
					return err
				}
				if err := checkConditionOperators(rule.AnyOf, crd.Name, rule.Field); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// checkKnownOperator returns an error for a set-but-unrecognized operator.
// Empty is fine — it means no explicit operator: was set, not an unknown one.
func checkKnownOperator(op orktypes.ConditionOperator, crdName, field string) error {
	if op == "" || orktypes.IsValidConditionOperator(op) {
		return nil
	}
	return errUnknownOperator(op, crdName, field)
}

func checkConditionOperators(conditions []orktypes.Condition, crdName, field string) error {
	for _, cond := range conditions {
		if err := checkKnownOperator(cond.Operator, crdName, field); err != nil {
			return err
		}
	}
	return nil
}

func errUnknownOperator(op orktypes.ConditionOperator, crd, field string) error {
	return fmt.Errorf(`
──────────────────────────────────────────────
%s Unknown operator %q
   CRD: %s
   field: %s

Allowed values:
  • equals, notEquals
  • contains, notContains, prefix, suffix, regex
  • exists, notExists
  • gt, lt, gte, lte, between, notBetween
  • in, notIn
  • unique
  • typeOf, typeMap, typeList, typeString, typeNumber, typeBool, typeNull
──────────────────────────────────────────────`, failureMark(), op, crd, field)
}

// validateValidationRuleLinks rejects a validation.rules entry's link: value
// that either doesn't name any idp.fields / idp.additionalFields key on the
// CRD (typo, or the field was renamed/removed and this rule wasn't updated),
// or names an idp.fields key whose Field is already the redundant plain
// "spec.<name>" form — link: only earns its keep when Field isn't already a
// clean display name on its own. See ValidationRule.Link.
func (k *Katalog) validateValidationRuleLinks() error {
	for _, crd := range k.enabledCRDs {
		if !crd.HasValidationRules() {
			continue
		}
		for _, rule := range crd.Validation.Rules {
			if rule.Link == "" {
				continue
			}
			if err := checkLink(rule, crd); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkLink(rule orktypes.ValidationRule, crd orktypes.CRDEntry) error {
	if crd.IDP != nil {
		if _, ok := crd.IDP.Fields[rule.Link]; ok {
			if rule.Field == "spec."+rule.Link {
				return errRedundantLink(rule.Link, crd.Name)
			}
			return nil
		}
		if _, ok := crd.AdditionalLabelFields()[rule.Link]; ok {
			return nil
		}
		if _, ok := crd.AdditionalAnnotationFields()[rule.Link]; ok {
			return nil
		}
	}
	return errUnknownLink(rule.Link, crd.Name, rule.Field)
}

func errUnknownLink(link, crd, field string) error {
	return fmt.Errorf(`
──────────────────────────────────────────────
%s link %q does not match any idp field
   CRD: %s
   field: %s

link: must name a key declared in idp.fields, idp.additionalFields.labels,
or idp.additionalFields.annotations for this CRD.
──────────────────────────────────────────────`, failureMark(), link, crd, field)
}

func errRedundantLink(link, crd string) error {
	return fmt.Errorf(`
──────────────────────────────────────────────
%s link %q is redundant
   CRD: %s

This rule's field is already "spec.%s" — already a clean display name on
its own. link: only matters when field is a template expression that isn't
itself a valid display name (e.g. wraps getLabel/getAnnotation, or a notes:
function). Remove link: here.
──────────────────────────────────────────────`, failureMark(), link, crd, link)
}
