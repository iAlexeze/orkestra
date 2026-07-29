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
❌ Unknown operator %q
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
──────────────────────────────────────────────`, op, crd, field)
}
