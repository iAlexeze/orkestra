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

func katalogWithIDPValidationRule(crdName string, idp *orktypes.IDPConfig, rules ...orktypes.ValidationRule) *Katalog {
	return &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			crdName: {
				Name:       crdName,
				IDP:        idp,
				Validation: &orktypes.ValidationConfig{Rules: rules},
			},
		},
	}
}

func TestValidateValidationRuleLinks_NoLink(t *testing.T) {
	k := katalogWithValidationRule("app", orktypes.ValidationRule{
		Field: "spec.image", Prefix: "myorg/", Message: "must be from myorg",
	})
	assert.NoError(t, k.validateValidationRuleLinks())
}

func TestValidateValidationRuleLinks_MatchesAdditionalLabelField(t *testing.T) {
	idp := &orktypes.IDPConfig{AdditionalFields: &orktypes.AdditionalIDPFields{
		Labels: map[string]orktypes.IDPFieldConfig{"team": {Label: "Team"}},
	}}
	k := katalogWithIDPValidationRule("app", idp, orktypes.ValidationRule{
		Field: `{{ isDNS1123Subdomain team }}`, Link: "team", Equals: "true", Message: "must be a valid subdomain",
	})
	assert.NoError(t, k.validateValidationRuleLinks())
}

func TestValidateValidationRuleLinks_MatchesAdditionalAnnotationField(t *testing.T) {
	idp := &orktypes.IDPConfig{AdditionalFields: &orktypes.AdditionalIDPFields{
		Annotations: map[string]orktypes.IDPFieldConfig{"platform.myorg.io/jira-ticket": {Label: "Jira Ticket"}},
	}}
	k := katalogWithIDPValidationRule("app", idp, orktypes.ValidationRule{
		Field: `{{ getAnnotation . "platform.myorg.io/jira-ticket" }}`, Link: "platform.myorg.io/jira-ticket", Message: "must be set",
	})
	assert.NoError(t, k.validateValidationRuleLinks())
}

func TestValidateValidationRuleLinks_SpecFieldWithWrappingExpression(t *testing.T) {
	// link: pointing at a spec field is valid when Field wraps it in
	// something other than the plain "spec.<name>" path — e.g. a format
	// check built on a notes: function.
	idp := &orktypes.IDPConfig{Fields: map[string]orktypes.IDPFieldConfig{
		"repoURL": {Label: "Repository URL"},
	}}
	k := katalogWithIDPValidationRule("app", idp, orktypes.ValidationRule{
		Field: `{{ isValidGitRepository .spec.repoURL }}`, Link: "repoURL", Equals: "true", Message: "must be a valid git repository",
	})
	assert.NoError(t, k.validateValidationRuleLinks())
}

func TestValidateValidationRuleLinks_RedundantSpecFieldLink(t *testing.T) {
	idp := &orktypes.IDPConfig{Fields: map[string]orktypes.IDPFieldConfig{
		"team": {Label: "Team"},
	}}
	k := katalogWithIDPValidationRule("app", idp, orktypes.ValidationRule{
		Field: "spec.team", Link: "team", Operator: orktypes.ConditionExists, Message: "team is required",
	})
	err := k.validateValidationRuleLinks()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redundant")
	assert.Contains(t, err.Error(), "team")
}

func TestValidateValidationRuleLinks_UnknownLink(t *testing.T) {
	idp := &orktypes.IDPConfig{AdditionalFields: &orktypes.AdditionalIDPFields{
		Labels: map[string]orktypes.IDPFieldConfig{"team": {Label: "Team"}},
	}}
	k := katalogWithIDPValidationRule("app", idp, orktypes.ValidationRule{
		Field: `{{ isDNS1123Subdomain typo }}`, Link: "typo", Equals: "true", Message: "must be valid",
	})
	err := k.validateValidationRuleLinks()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "typo")
	assert.Contains(t, err.Error(), "does not match any idp field")
}

func TestValidateValidationRuleLinks_NoIDPConfig(t *testing.T) {
	k := katalogWithIDPValidationRule("app", nil, orktypes.ValidationRule{
		Field: `{{ isDNS1123Subdomain team }}`, Link: "team", Equals: "true", Message: "must be valid",
	})
	err := k.validateValidationRuleLinks()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "team")
}
