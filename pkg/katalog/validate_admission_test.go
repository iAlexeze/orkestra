package katalog

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func katalogWithValidationRule(crdName string, rules ...orktypes.ValidationRule) *Katalog {
	return &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			crdName: {
				Name:       crdName,
				Validation: &orktypes.ValidationConfig{Rules: rules},
			},
		},
	}
}

func katalogWithMutationRule(crdName string, rules ...orktypes.MutationRule) *Katalog {
	return &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			crdName: {
				Name:     crdName,
				Mutation: &orktypes.MutationConfig{Rules: rules},
			},
		},
	}
}

func TestValidateAdmissionOperators_NoCRDs(t *testing.T) {
	k := &Katalog{enabledCRDs: map[string]orktypes.CRDEntry{}}
	assert.NoError(t, k.validateAdmissionOperators())
}

func TestValidateAdmissionOperators_NoOperatorSet(t *testing.T) {
	k := katalogWithValidationRule("app", orktypes.ValidationRule{
		Field: "spec.image", Prefix: "myorg/", Message: "must be from myorg",
	})
	assert.NoError(t, k.validateAdmissionOperators())
}

func TestValidateAdmissionOperators_KnownOperator(t *testing.T) {
	k := katalogWithValidationRule("app", orktypes.ValidationRule{
		Field: "spec.replicas", Operator: orktypes.ConditionLte, Value: "10", Message: "too many replicas",
	})
	assert.NoError(t, k.validateAdmissionOperators())
}

func TestValidateAdmissionOperators_UnknownOperator(t *testing.T) {
	k := katalogWithValidationRule("app", orktypes.ValidationRule{
		Field: "spec.replicas", Operator: "lte2", Value: "10", Message: "too many replicas",
	})
	err := k.validateAdmissionOperators()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lte2")
	assert.Contains(t, err.Error(), "app")
	assert.Contains(t, err.Error(), "spec.replicas")
}

func TestValidateAdmissionOperators_UnknownOperatorInWhen(t *testing.T) {
	k := katalogWithValidationRule("app", orktypes.ValidationRule{
		Field: "spec.replicas", Prefix: "x", Message: "msg",
		When: []orktypes.Condition{{Field: "spec.tier", Operator: "greaterOrEqual"}},
	})
	err := k.validateAdmissionOperators()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "greaterOrEqual")
}

func TestValidateAdmissionOperators_UnknownOperatorInAnyOf(t *testing.T) {
	k := katalogWithValidationRule("app", orktypes.ValidationRule{
		Field: "spec.replicas", Prefix: "x", Message: "msg",
		AnyOf: []orktypes.Condition{{Field: "spec.tier", Operator: "notARealOp"}},
	})
	err := k.validateAdmissionOperators()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notARealOp")
}

func TestValidateAdmissionOperators_MutationRuleUnknownOperatorInWhen(t *testing.T) {
	k := katalogWithMutationRule("app", orktypes.MutationRule{
		Field:   "spec.tier",
		Default: "standard",
		When:    []orktypes.Condition{{Field: "spec.env", Operator: "isEqualTo"}},
	})
	err := k.validateAdmissionOperators()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "isEqualTo")
}

func TestValidateAdmissionOperators_MutationRuleKnownOperator(t *testing.T) {
	k := katalogWithMutationRule("app", orktypes.MutationRule{
		Field:   "spec.tier",
		Default: "standard",
		When:    []orktypes.Condition{{Field: "spec.env", Operator: orktypes.ConditionGte, Value: "1"}},
	})
	assert.NoError(t, k.validateAdmissionOperators())
}
