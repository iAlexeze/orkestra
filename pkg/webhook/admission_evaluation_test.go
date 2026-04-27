// Tests for the pure evaluation logic in admission_evaluation.go and
// the shared helper functions in admission.go.
//
// Package webhook (white-box) — gives direct access to unexported functions
// without requiring exported test shims. These are the innermost unit tests:
// no network, no Kubernetes, no filesystem.
package webhook

import (
	"context"
	"encoding/json"
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ── anyToString ───────────────────────────────────────────────────────────────

func TestAnyToString(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{name: "string", input: "hello", want: "hello"},
		{name: "empty string", input: "", want: ""},
		{name: "bool true", input: true, want: "true"},
		{name: "bool false", input: false, want: "false"},
		{name: "int64 positive", input: int64(42), want: "42"},
		{name: "int64 negative", input: int64(-7), want: "-7"},
		{name: "int64 zero", input: int64(0), want: "0"},
		{name: "float64 integer value", input: float64(5), want: "5"},
		{name: "float64 decimal value", input: float64(3.14), want: "3.14"},
		{name: "float64 zero", input: float64(0), want: "0"},
		{name: "nil", input: nil, want: ""},
		{name: "other type falls back to fmt", input: []string{"a", "b"}, want: "[a b]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, anyToString(tc.input))
		})
	}
}

// ── resolveFieldPath ──────────────────────────────────────────────────────────

func TestResolveFieldPath(t *testing.T) {
	obj := map[string]interface{}{
		"apiVersion": "demo.orkestra.io/v1alpha1",
		"metadata": map[string]interface{}{
			"name":      "my-site",
			"namespace": "default",
			"labels": map[string]interface{}{
				"team": "platform",
			},
		},
		"spec": map[string]interface{}{
			"image":    "myorg/nginx:1.25",
			"replicas": float64(3),
			"enabled":  true,
			"db": map[string]interface{}{
				"engine": "postgres",
			},
		},
	}

	tests := []struct {
		name      string
		path      string
		wantValue string
		wantFound bool
	}{
		{name: "top-level string", path: "apiVersion", wantValue: "demo.orkestra.io/v1alpha1", wantFound: true},
		{name: "nested string", path: "spec.image", wantValue: "myorg/nginx:1.25", wantFound: true},
		{name: "nested float64 (int)", path: "spec.replicas", wantValue: "3", wantFound: true},
		{name: "nested bool", path: "spec.enabled", wantValue: "true", wantFound: true},
		{name: "deeply nested", path: "spec.db.engine", wantValue: "postgres", wantFound: true},
		{name: "nested under metadata", path: "metadata.name", wantValue: "my-site", wantFound: true},
		{name: "three levels deep", path: "metadata.labels.team", wantValue: "platform", wantFound: true},
		{name: "missing top-level", path: "status", wantValue: "", wantFound: false},
		{name: "missing nested", path: "spec.port", wantValue: "", wantFound: false},
		{name: "missing deeply nested", path: "spec.db.host", wantValue: "", wantFound: false},
		{name: "intermediate is a leaf not a map", path: "spec.image.subfield", wantValue: "", wantFound: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, found := resolveFieldPath(obj, tc.path)
			assert.Equal(t, tc.wantValue, got, "value mismatch")
			assert.Equal(t, tc.wantFound, found, "found mismatch")
		})
	}
}

func TestResolveFieldPath_EmptyObject(t *testing.T) {
	obj := map[string]interface{}{}
	val, found := resolveFieldPath(obj, "spec.image")
	assert.Equal(t, "", val)
	assert.False(t, found)
}

// ── setFieldPath ──────────────────────────────────────────────────────────────

func TestSetFieldPath(t *testing.T) {
	t.Run("top-level field", func(t *testing.T) {
		obj := map[string]interface{}{}
		setFieldPath(obj, "name", "orkestra")
		assert.Equal(t, "orkestra", obj["name"])
	})

	t.Run("nested field creates intermediate map", func(t *testing.T) {
		obj := map[string]interface{}{}
		setFieldPath(obj, "spec.image", "myorg/app:latest")
		spec, ok := obj["spec"].(map[string]interface{})
		require.True(t, ok, "spec should be a map")
		assert.Equal(t, "myorg/app:latest", spec["image"])
	})

	t.Run("deeply nested creates all levels", func(t *testing.T) {
		obj := map[string]interface{}{}
		setFieldPath(obj, "spec.db.host", "localhost")
		spec := obj["spec"].(map[string]interface{})
		db := spec["db"].(map[string]interface{})
		assert.Equal(t, "localhost", db["host"])
	})

	t.Run("overwrites existing scalar", func(t *testing.T) {
		obj := map[string]interface{}{
			"spec": map[string]interface{}{"replicas": "1"},
		}
		setFieldPath(obj, "spec.replicas", "3")
		spec := obj["spec"].(map[string]interface{})
		assert.Equal(t, "3", spec["replicas"])
	})

	t.Run("sibling keys preserved", func(t *testing.T) {
		obj := map[string]interface{}{
			"spec": map[string]interface{}{"image": "nginx"},
		}
		setFieldPath(obj, "spec.replicas", "2")
		spec := obj["spec"].(map[string]interface{})
		assert.Equal(t, "nginx", spec["image"])
		assert.Equal(t, "2", spec["replicas"])
	})
}

// ── resolveValidationOperator ─────────────────────────────────────────────────

func TestResolveValidationOperator(t *testing.T) {
	tests := []struct {
		name    string
		rule    orktypes.ValidationRule
		wantOp  orktypes.ConditionOperator
		wantVal string
	}{
		{
			name:    "equals shorthand",
			rule:    orktypes.ValidationRule{Equals: "production"},
			wantOp:  orktypes.ConditionEquals,
			wantVal: "production",
		},
		{
			name:    "notEquals shorthand",
			rule:    orktypes.ValidationRule{NotEquals: "test"},
			wantOp:  orktypes.ConditionNotEquals,
			wantVal: "test",
		},
		{
			name:    "prefix shorthand",
			rule:    orktypes.ValidationRule{Prefix: "myorg/"},
			wantOp:  orktypes.ConditionPrefix,
			wantVal: "myorg/",
		},
		{
			name:    "suffix shorthand",
			rule:    orktypes.ValidationRule{Suffix: ":latest"},
			wantOp:  orktypes.ConditionSuffix,
			wantVal: ":latest",
		},
		{
			name:    "contains shorthand",
			rule:    orktypes.ValidationRule{Contains: "nginx"},
			wantOp:  orktypes.ConditionContains,
			wantVal: "nginx",
		},
		{
			name:    "min shorthand maps to ConditionGt",
			rule:    orktypes.ValidationRule{Min: "1"},
			wantOp:  orktypes.ConditionGt,
			wantVal: "1",
		},
		{
			name:    "max shorthand maps to ConditionLt",
			rule:    orktypes.ValidationRule{Max: "10"},
			wantOp:  orktypes.ConditionLt,
			wantVal: "10",
		},
		{
			name:    "explicit operator and value",
			rule:    orktypes.ValidationRule{Operator: orktypes.ConditionContains, Value: "platform"},
			wantOp:  orktypes.ConditionContains,
			wantVal: "platform",
		},
		{
			name:    "no shorthand defaults to exists",
			rule:    orktypes.ValidationRule{Field: "spec.image"},
			wantOp:  orktypes.ConditionExists,
			wantVal: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotOp, gotVal := resolveValidationOperator(tc.rule)
			assert.Equal(t, tc.wantOp, gotOp)
			assert.Equal(t, tc.wantVal, gotVal)
		})
	}
}

// ── evaluateOneRule ───────────────────────────────────────────────────────────

func specObj(fields map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"spec": fields}
}

func TestEvaluateOneRule(t *testing.T) {
	tests := []struct {
		name     string
		obj      map[string]interface{}
		rule     orktypes.ValidationRule
		wantPass bool
	}{
		// ── exists ────────────────────────────────────────────────────────────
		{
			name:     "exists: field present with value — passes",
			obj:      specObj(map[string]interface{}{"image": "myorg/app:v1"}),
			rule:     orktypes.ValidationRule{Field: "spec.image", Operator: orktypes.ConditionExists, Message: "required"},
			wantPass: true,
		},
		{
			name:     "exists: field missing — fails",
			obj:      specObj(map[string]interface{}{}),
			rule:     orktypes.ValidationRule{Field: "spec.image", Operator: orktypes.ConditionExists, Message: "required"},
			wantPass: false,
		},
		{
			name:     "exists: field present but empty string — fails",
			obj:      specObj(map[string]interface{}{"image": ""}),
			rule:     orktypes.ValidationRule{Field: "spec.image", Operator: orktypes.ConditionExists, Message: "required"},
			wantPass: false,
		},
		// ── notExists ─────────────────────────────────────────────────────────
		{
			name:     "notExists: field absent — passes",
			obj:      specObj(map[string]interface{}{}),
			rule:     orktypes.ValidationRule{Field: "spec.debug", Operator: orktypes.ConditionNotExists, Message: "debug not allowed"},
			wantPass: true,
		},
		{
			name:     "notExists: field present with value — fails",
			obj:      specObj(map[string]interface{}{"debug": "true"}),
			rule:     orktypes.ValidationRule{Field: "spec.debug", Operator: orktypes.ConditionNotExists, Message: "debug not allowed"},
			wantPass: false,
		},
		{
			name:     "notExists: field present but empty — passes",
			obj:      specObj(map[string]interface{}{"debug": ""}),
			rule:     orktypes.ValidationRule{Field: "spec.debug", Operator: orktypes.ConditionNotExists, Message: "debug not allowed"},
			wantPass: true,
		},
		// ── equals ────────────────────────────────────────────────────────────
		{
			name:     "equals: exact match — passes",
			obj:      specObj(map[string]interface{}{"env": "production"}),
			rule:     orktypes.ValidationRule{Field: "spec.env", Equals: "production", Message: "must be production"},
			wantPass: true,
		},
		{
			name:     "equals: no match — fails",
			obj:      specObj(map[string]interface{}{"env": "staging"}),
			rule:     orktypes.ValidationRule{Field: "spec.env", Equals: "production", Message: "must be production"},
			wantPass: false,
		},
		{
			name:     "equals: field missing — fails",
			obj:      specObj(map[string]interface{}{}),
			rule:     orktypes.ValidationRule{Field: "spec.env", Equals: "production", Message: "must be production"},
			wantPass: false,
		},
		// ── notEquals ─────────────────────────────────────────────────────────
		{
			name:     "notEquals: different value — passes",
			obj:      specObj(map[string]interface{}{"env": "staging"}),
			rule:     orktypes.ValidationRule{Field: "spec.env", NotEquals: "production", Message: "must not be production"},
			wantPass: true,
		},
		{
			name:     "notEquals: matching value — fails",
			obj:      specObj(map[string]interface{}{"env": "production"}),
			rule:     orktypes.ValidationRule{Field: "spec.env", NotEquals: "production", Message: "must not be production"},
			wantPass: false,
		},
		{
			name:     "notEquals: field missing — passes",
			obj:      specObj(map[string]interface{}{}),
			rule:     orktypes.ValidationRule{Field: "spec.env", NotEquals: "production", Message: "must not be production"},
			wantPass: true,
		},
		// ── prefix ────────────────────────────────────────────────────────────
		{
			name:     "prefix: correct prefix — passes",
			obj:      specObj(map[string]interface{}{"image": "myorg/nginx:1.25"}),
			rule:     orktypes.ValidationRule{Field: "spec.image", Prefix: "myorg/", Message: "must use myorg registry"},
			wantPass: true,
		},
		{
			name:     "prefix: wrong prefix — fails",
			obj:      specObj(map[string]interface{}{"image": "docker.io/nginx:1.25"}),
			rule:     orktypes.ValidationRule{Field: "spec.image", Prefix: "myorg/", Message: "must use myorg registry"},
			wantPass: false,
		},
		{
			name:     "prefix: field missing — fails",
			obj:      specObj(map[string]interface{}{}),
			rule:     orktypes.ValidationRule{Field: "spec.image", Prefix: "myorg/", Message: "must use myorg registry"},
			wantPass: false,
		},
		// ── suffix ────────────────────────────────────────────────────────────
		{
			name:     "suffix: correct suffix — passes",
			obj:      specObj(map[string]interface{}{"image": "myorg/app:v1.0"}),
			rule:     orktypes.ValidationRule{Field: "spec.image", Suffix: ":v1.0", Message: "must use v1.0 tag"},
			wantPass: true,
		},
		{
			name:     "suffix: wrong suffix — fails",
			obj:      specObj(map[string]interface{}{"image": "myorg/app:latest"}),
			rule:     orktypes.ValidationRule{Field: "spec.image", Suffix: ":v1.0", Message: "must use v1.0 tag"},
			wantPass: false,
		},
		// ── contains ──────────────────────────────────────────────────────────
		{
			name:     "contains: substring present — passes",
			obj:      specObj(map[string]interface{}{"image": "registry.myorg.io/nginx:1.25"}),
			rule:     orktypes.ValidationRule{Field: "spec.image", Contains: "myorg", Message: "must be from myorg"},
			wantPass: true,
		},
		{
			name:     "contains: substring absent — fails",
			obj:      specObj(map[string]interface{}{"image": "docker.io/nginx:1.25"}),
			rule:     orktypes.ValidationRule{Field: "spec.image", Contains: "myorg", Message: "must be from myorg"},
			wantPass: false,
		},
		// ── min (ConditionGt) ─────────────────────────────────────────────────
		{
			name:     "min: value exactly at minimum — passes",
			obj:      specObj(map[string]interface{}{"replicas": float64(1)}),
			rule:     orktypes.ValidationRule{Field: "spec.replicas", Min: "1", Message: "at least 1 replica"},
			wantPass: true,
		},
		{
			name:     "min: value above minimum — passes",
			obj:      specObj(map[string]interface{}{"replicas": float64(5)}),
			rule:     orktypes.ValidationRule{Field: "spec.replicas", Min: "1", Message: "at least 1 replica"},
			wantPass: true,
		},
		{
			name:     "min: value below minimum — fails",
			obj:      specObj(map[string]interface{}{"replicas": float64(0)}),
			rule:     orktypes.ValidationRule{Field: "spec.replicas", Min: "1", Message: "at least 1 replica"},
			wantPass: false,
		},
		{
			name:     "min: field missing — fails",
			obj:      specObj(map[string]interface{}{}),
			rule:     orktypes.ValidationRule{Field: "spec.replicas", Min: "1", Message: "at least 1 replica"},
			wantPass: false,
		},
		{
			name:     "min: non-numeric field value — fails",
			obj:      specObj(map[string]interface{}{"replicas": "lots"}),
			rule:     orktypes.ValidationRule{Field: "spec.replicas", Min: "1", Message: "at least 1 replica"},
			wantPass: false,
		},
		{
			name:     "min: non-numeric config value — rule skipped (passes)",
			obj:      specObj(map[string]interface{}{"replicas": float64(5)}),
			rule:     orktypes.ValidationRule{Field: "spec.replicas", Min: "not-a-number", Message: "invalid config"},
			wantPass: true,
		},
		// ── max (ConditionLt) ─────────────────────────────────────────────────
		{
			name:     "max: value at maximum — passes",
			obj:      specObj(map[string]interface{}{"replicas": float64(10)}),
			rule:     orktypes.ValidationRule{Field: "spec.replicas", Max: "10", Message: "no more than 10"},
			wantPass: true,
		},
		{
			name:     "max: value above maximum — fails",
			obj:      specObj(map[string]interface{}{"replicas": float64(15)}),
			rule:     orktypes.ValidationRule{Field: "spec.replicas", Max: "10", Message: "no more than 10"},
			wantPass: false,
		},
		{
			name:     "max: value below maximum — passes",
			obj:      specObj(map[string]interface{}{"replicas": float64(3)}),
			rule:     orktypes.ValidationRule{Field: "spec.replicas", Max: "10", Message: "no more than 10"},
			wantPass: true,
		},
		{
			name:     "max: field missing — fails",
			obj:      specObj(map[string]interface{}{}),
			rule:     orktypes.ValidationRule{Field: "spec.replicas", Max: "10", Message: "no more than 10"},
			wantPass: false,
		},
		{
			name:     "max: non-numeric field value — fails",
			obj:      specObj(map[string]interface{}{"replicas": "many"}),
			rule:     orktypes.ValidationRule{Field: "spec.replicas", Max: "10", Message: "no more than 10"},
			wantPass: false,
		},
		{
			name:     "max: non-numeric config value — rule skipped (passes)",
			obj:      specObj(map[string]interface{}{"replicas": float64(5)}),
			rule:     orktypes.ValidationRule{Field: "spec.replicas", Max: "not-a-number", Message: "invalid config"},
			wantPass: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			violation := evaluateOneRule(tc.obj, tc.rule)
			if tc.wantPass {
				assert.Nil(t, violation, "expected rule to pass — got violation: %+v", violation)
			} else {
				assert.NotNil(t, violation, "expected rule to fail — got nil violation")
			}
		})
	}
}

func TestEvaluateOneRule_ViolationFieldsArePopulated(t *testing.T) {
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"image": "docker.io/nginx:1.25",
		},
	}
	rule := orktypes.ValidationRule{
		Field:   "spec.image",
		Prefix:  "myorg/",
		Message: "image must be from the myorg registry",
		Action:  orktypes.ValidationActionDeny,
	}

	v := evaluateOneRule(obj, rule)

	require.NotNil(t, v)
	assert.Equal(t, "spec.image", v.Field)
	assert.Equal(t, "image must be from the myorg registry", v.Message)
	assert.Equal(t, "docker.io/nginx:1.25", v.Got)
	assert.Equal(t, string(orktypes.ConditionPrefix), v.RuleType)
}

// ── evaluateValidationRules: multi-rule ───────────────────────────────────────

func TestEvaluateValidationRules_MixedDenyAndWarn(t *testing.T) {
	h := &WebhookServer{}

	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"image":    "docker.io/nginx:1.25", // violates deny rule
			"replicas": float64(15),            // violates deny rule
		},
		// metadata.labels.team absent → violates warn rule
	}
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.image", Prefix: "myorg/", Message: "wrong registry", Action: orktypes.ValidationActionDeny},
			{Field: "spec.replicas", Max: "10", Message: "too many replicas", Action: orktypes.ValidationActionDeny},
			{Field: "metadata.labels.team", Operator: orktypes.ConditionExists, Message: "team label required", Action: orktypes.ValidationActionWarn},
		},
	}

	denials, warnings := h.evaluateValidationRules(obj, cfg, "Website")

	assert.Len(t, denials, 2, "should collect all deny violations — not fail-fast")
	assert.Len(t, warnings, 1)
	assert.Equal(t, "spec.image", denials[0].Field)
	assert.Equal(t, "spec.replicas", denials[1].Field)
	assert.Equal(t, "metadata.labels.team", warnings[0].Field)
}

func TestEvaluateValidationRules_AllPass(t *testing.T) {
	h := &WebhookServer{}

	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"image":    "myorg/nginx:1.25",
			"replicas": float64(3),
		},
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{"team": "platform"},
		},
	}
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.image", Prefix: "myorg/", Message: "wrong registry", Action: orktypes.ValidationActionDeny},
			{Field: "spec.replicas", Max: "10", Message: "too many", Action: orktypes.ValidationActionDeny},
			{Field: "metadata.labels.team", Operator: orktypes.ConditionExists, Message: "team required", Action: orktypes.ValidationActionWarn},
		},
	}

	denials, warnings := h.evaluateValidationRules(obj, cfg, "Website")

	assert.Empty(t, denials)
	assert.Empty(t, warnings)
}

func TestEvaluateValidationRules_EmptyActionDefaultsToDeny(t *testing.T) {
	h := &WebhookServer{}

	obj := specObj(map[string]interface{}{"image": "bad-registry/nginx"})
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			// Action field not set — should default to deny
			{Field: "spec.image", Prefix: "myorg/", Message: "wrong registry"},
		},
	}

	denials, warnings := h.evaluateValidationRules(obj, cfg, "Website")

	assert.Len(t, denials, 1)
	assert.Empty(t, warnings)
}

// ── applyMutationRules ────────────────────────────────────────────────────────

func TestApplyMutationRules_NilConfig(t *testing.T) {
	h := &WebhookServer{}
	obj := specObj(map[string]interface{}{})

	changes, err := h.applyMutationRules(context.Background(), obj, nil, "Website")

	require.NoError(t, err)
	assert.Empty(t, changes)
}

func TestApplyMutationRules_EmptyRules(t *testing.T) {
	h := &WebhookServer{}
	obj := specObj(map[string]interface{}{})
	cfg := &orktypes.MutationConfig{Rules: []orktypes.MutationRule{}}

	changes, err := h.applyMutationRules(context.Background(), obj, cfg, "Website")

	require.NoError(t, err)
	assert.Empty(t, changes)
}

func TestApplyMutationRules_DefaultAppliedWhenAbsent(t *testing.T) {
	h := &WebhookServer{}
	obj := map[string]interface{}{
		"metadata": map[string]interface{}{"name": "my-site"},
		"spec":     map[string]interface{}{},
	}
	cfg := &orktypes.MutationConfig{
		Rules: []orktypes.MutationRule{
			{Field: "spec.replicas", Default: "2"},
		},
	}

	changes, err := h.applyMutationRules(context.Background(), obj, cfg, "Website")

	require.NoError(t, err)
	require.Len(t, changes, 1)
	assert.Equal(t, "spec.replicas", changes[0].Field)
	assert.Equal(t, "", changes[0].OldValue)
	assert.Equal(t, "2", changes[0].NewValue)
	assert.Equal(t, "default", changes[0].ChangeType)

	// Verify in-place mutation
	spec := obj["spec"].(map[string]interface{})
	assert.Equal(t, "2", spec["replicas"])
}

func TestApplyMutationRules_DefaultSkippedWhenPresent(t *testing.T) {
	h := &WebhookServer{}
	obj := map[string]interface{}{
		"metadata": map[string]interface{}{"name": "my-site"},
		"spec":     map[string]interface{}{"replicas": "5"},
	}
	cfg := &orktypes.MutationConfig{
		Rules: []orktypes.MutationRule{
			{Field: "spec.replicas", Default: "2"},
		},
	}

	changes, err := h.applyMutationRules(context.Background(), obj, cfg, "Website")

	require.NoError(t, err)
	assert.Empty(t, changes, "default must not overwrite an existing value")

	spec := obj["spec"].(map[string]interface{})
	assert.Equal(t, "5", spec["replicas"], "original value must be preserved")
}

func TestApplyMutationRules_DefaultSkippedWhenValueAlreadyMatches(t *testing.T) {
	h := &WebhookServer{}
	obj := map[string]interface{}{
		"metadata": map[string]interface{}{"name": "my-site"},
		"spec":     map[string]interface{}{"logLevel": "info"},
	}
	cfg := &orktypes.MutationConfig{
		Rules: []orktypes.MutationRule{
			{Field: "spec.logLevel", Default: "info"},
		},
	}

	changes, err := h.applyMutationRules(context.Background(), obj, cfg, "Website")

	require.NoError(t, err)
	assert.Empty(t, changes, "no change emitted when value already matches")
}

func TestApplyMutationRules_OverrideAlwaysApplies(t *testing.T) {
	h := &WebhookServer{}
	obj := map[string]interface{}{
		"metadata": map[string]interface{}{"name": "my-site"},
		"spec":     map[string]interface{}{},
	}
	cfg := &orktypes.MutationConfig{
		Rules: []orktypes.MutationRule{
			{Field: "metadata.labels.managed-by", Override: "orkestra"},
		},
	}

	changes, err := h.applyMutationRules(context.Background(), obj, cfg, "Website")

	require.NoError(t, err)
	require.Len(t, changes, 1)
	assert.Equal(t, "metadata.labels.managed-by", changes[0].Field)
	assert.Equal(t, "orkestra", changes[0].NewValue)
	assert.Equal(t, "override", changes[0].ChangeType)
}

func TestApplyMutationRules_OverrideReplacesExistingValue(t *testing.T) {
	h := &WebhookServer{}
	obj := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "my-site",
			"labels": map[string]interface{}{
				"managed-by": "helm",
			},
		},
		"spec": map[string]interface{}{},
	}
	cfg := &orktypes.MutationConfig{
		Rules: []orktypes.MutationRule{
			{Field: "metadata.labels.managed-by", Override: "orkestra"},
		},
	}

	changes, err := h.applyMutationRules(context.Background(), obj, cfg, "Website")

	require.NoError(t, err)
	require.Len(t, changes, 1)
	assert.Equal(t, "helm", changes[0].OldValue)
	assert.Equal(t, "orkestra", changes[0].NewValue)
	assert.Equal(t, "override", changes[0].ChangeType)
}

func TestApplyMutationRules_MultipleRules(t *testing.T) {
	h := &WebhookServer{}
	obj := map[string]interface{}{
		"metadata": map[string]interface{}{"name": "my-site"},
		"spec":     map[string]interface{}{},
	}
	cfg := &orktypes.MutationConfig{
		Rules: []orktypes.MutationRule{
			{Field: "spec.replicas", Default: "2"},
			{Field: "spec.logLevel", Default: "info"},
			{Field: "metadata.labels.managed-by", Override: "orkestra"},
		},
	}

	changes, err := h.applyMutationRules(context.Background(), obj, cfg, "Website")

	require.NoError(t, err)
	assert.Len(t, changes, 3)

	spec := obj["spec"].(map[string]interface{})
	assert.Equal(t, "2", spec["replicas"])
	assert.Equal(t, "info", spec["logLevel"])
}

func TestApplyMutationRules_NestedFieldCreated(t *testing.T) {
	h := &WebhookServer{}
	obj := map[string]interface{}{
		"metadata": map[string]interface{}{"name": "my-site"},
		"spec":     map[string]interface{}{},
	}
	cfg := &orktypes.MutationConfig{
		Rules: []orktypes.MutationRule{
			{Field: "spec.db.engine", Default: "postgres"},
		},
	}

	changes, err := h.applyMutationRules(context.Background(), obj, cfg, "Website")

	require.NoError(t, err)
	require.Len(t, changes, 1)

	spec := obj["spec"].(map[string]interface{})
	db, ok := spec["db"].(map[string]interface{})
	require.True(t, ok, "intermediate map should have been created")
	assert.Equal(t, "postgres", db["engine"])
}

// ── deepCopyMap ───────────────────────────────────────────────────────────────

func TestDeepCopyMap_IndependentFromOriginal(t *testing.T) {
	original := map[string]interface{}{
		"spec": map[string]interface{}{
			"image": "myorg/nginx:1.25",
		},
	}

	cp := deepCopyMap(original)

	// Mutate the copy — original must not change
	cp["spec"].(map[string]interface{})["image"] = "mutated"

	assert.Equal(t, "myorg/nginx:1.25",
		original["spec"].(map[string]interface{})["image"],
		"original must not be affected by copy mutation")
}

func TestDeepCopyMap_NilReturnsNil(t *testing.T) {
	assert.Nil(t, deepCopyMap(nil))
}

func TestDeepCopyMap_PreservesAllFields(t *testing.T) {
	original := map[string]interface{}{
		"a": "string",
		"b": float64(42),
		"c": true,
		"d": map[string]interface{}{"nested": "value"},
	}

	cp := deepCopyMap(original)

	assert.Equal(t, "string", cp["a"])
	assert.Equal(t, float64(42), cp["b"])
	assert.Equal(t, true, cp["c"])
	assert.Equal(t, "value", cp["d"].(map[string]interface{})["nested"])
}

// ── buildJSONPatch ────────────────────────────────────────────────────────────

func TestBuildJSONPatch_AddForAbsentField(t *testing.T) {
	changes := []fieldChange{
		{Field: "spec.replicas", OldValue: "", NewValue: "2", ChangeType: "default"},
	}

	raw, err := buildJSONPatch(changes)
	require.NoError(t, err)

	var ops []JSONPatchOp
	require.NoError(t, json.Unmarshal(raw, &ops))
	require.Len(t, ops, 1)
	assert.Equal(t, "add", ops[0].Op)
	assert.Equal(t, "/spec/replicas", ops[0].Path)
	assert.Equal(t, "2", ops[0].Value)
}

func TestBuildJSONPatch_ReplaceForExistingField(t *testing.T) {
	changes := []fieldChange{
		{Field: "metadata.labels.managed-by", OldValue: "helm", NewValue: "orkestra", ChangeType: "override"},
	}

	raw, err := buildJSONPatch(changes)
	require.NoError(t, err)

	var ops []JSONPatchOp
	require.NoError(t, json.Unmarshal(raw, &ops))
	require.Len(t, ops, 1)
	assert.Equal(t, "replace", ops[0].Op)
	assert.Equal(t, "/metadata/labels/managed-by", ops[0].Path)
}

func TestBuildJSONPatch_MultipleChanges(t *testing.T) {
	changes := []fieldChange{
		{Field: "spec.replicas", OldValue: "", NewValue: "2", ChangeType: "default"},
		{Field: "spec.logLevel", OldValue: "", NewValue: "info", ChangeType: "default"},
		{Field: "metadata.labels.managed-by", OldValue: "helm", NewValue: "orkestra", ChangeType: "override"},
	}

	raw, err := buildJSONPatch(changes)
	require.NoError(t, err)

	var ops []JSONPatchOp
	require.NoError(t, json.Unmarshal(raw, &ops))
	assert.Len(t, ops, 3)
}

func TestBuildJSONPatch_EmptyChanges(t *testing.T) {
	raw, err := buildJSONPatch(nil)
	require.NoError(t, err)

	var ops []JSONPatchOp
	require.NoError(t, json.Unmarshal(raw, &ops))
	assert.Empty(t, ops)
}

// ── gvrToKey ──────────────────────────────────────────────────────────────────

func TestGVRToKey(t *testing.T) {
	tests := []struct {
		name string
		gvr  metav1.GroupVersionResource
		want string
	}{
		{
			name: "group resource",
			gvr:  metav1.GroupVersionResource{Group: "demo.orkestra.io", Version: "v1alpha1", Resource: "websites"},
			want: "demo.orkestra.io/v1alpha1/websites",
		},
		{
			name: "core group (empty group)",
			gvr:  metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			want: "v1/pods",
		},
		{
			name: "apps group",
			gvr:  metav1.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
			want: "apps/v1/deployments",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, gvrToKey(tc.gvr))
		})
	}
}
