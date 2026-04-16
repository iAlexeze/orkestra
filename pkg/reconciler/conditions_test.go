// pkg/reconciler/conditions_test.go
package reconciler

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// buildUnstructured builds a minimal unstructured CR for condition tests.
func buildUnstructured(spec map[string]interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "demo.orkestra.io/v1alpha1",
			"kind":       "Website",
			"metadata": map[string]interface{}{
				"name":      "my-website",
				"namespace": "default",
			},
			"spec": spec,
		},
	}
}

func TestEvaluateConditions_Empty(t *testing.T) {
	obj := buildUnstructured(map[string]interface{}{})
	if !evaluateConditions(obj, nil) {
		t.Error("empty conditions should always return true")
	}
	if !evaluateConditions(obj, []orktypes.Condition{}) {
		t.Error("empty slice should always return true")
	}
}

func TestEvaluateConditions_EqualsShorthand(t *testing.T) {
	obj := buildUnstructured(map[string]interface{}{"environment": "production"})

	conds := []orktypes.Condition{{Field: "spec.environment", Equals: "production"}}
	if !evaluateConditions(obj, conds) {
		t.Error("equals shorthand should pass when value matches")
	}

	conds = []orktypes.Condition{{Field: "spec.environment", Equals: "staging"}}
	if evaluateConditions(obj, conds) {
		t.Error("equals shorthand should fail when value does not match")
	}
}

func TestEvaluateConditions_AllMustPass(t *testing.T) {
	obj := buildUnstructured(map[string]interface{}{
		"environment": "production",
		"enabled":     "true",
	})

	conds := []orktypes.Condition{
		{Field: "spec.environment", Equals: "production"},
		{Field: "spec.enabled", Equals: "true"},
	}
	if !evaluateConditions(obj, conds) {
		t.Error("all conditions pass — should return true")
	}

	conds = []orktypes.Condition{
		{Field: "spec.environment", Equals: "production"},
		{Field: "spec.enabled", Equals: "false"},
	}
	if evaluateConditions(obj, conds) {
		t.Error("one condition fails — should return false")
	}
}

func TestEvaluateConditions_Operators(t *testing.T) {
	obj := buildUnstructured(map[string]interface{}{
		"image":    "myorg/nginx:1.25",
		"replicas": int64(5),
		"logLevel": "info",
	})

	tests := []struct {
		name   string
		cond   orktypes.Condition
		expect bool
	}{
		{
			name:   "prefix matches",
			cond:   orktypes.Condition{Field: "spec.image", Operator: orktypes.ConditionPrefix, Value: "myorg/"},
			expect: true,
		},
		{
			name:   "prefix no match",
			cond:   orktypes.Condition{Field: "spec.image", Operator: orktypes.ConditionPrefix, Value: "other/"},
			expect: false,
		},
		{
			name:   "suffix matches",
			cond:   orktypes.Condition{Field: "spec.image", Operator: orktypes.ConditionSuffix, Value: ":1.25"},
			expect: true,
		},
		{
			name:   "contains matches",
			cond:   orktypes.Condition{Field: "spec.image", Operator: orktypes.ConditionContains, Value: "nginx"},
			expect: true,
		},
		{
			name:   "notEquals matches",
			cond:   orktypes.Condition{Field: "spec.logLevel", Operator: orktypes.ConditionNotEquals, Value: "debug"},
			expect: true,
		},
		{
			name:   "gt matches — 5 > 3",
			cond:   orktypes.Condition{Field: "spec.replicas", Operator: orktypes.ConditionGt, Value: "3"},
			expect: true,
		},
		{
			name:   "gt fails — 5 not > 10",
			cond:   orktypes.Condition{Field: "spec.replicas", Operator: orktypes.ConditionGt, Value: "10"},
			expect: false,
		},
		{
			name:   "lt matches — 5 < 10",
			cond:   orktypes.Condition{Field: "spec.replicas", Operator: orktypes.ConditionLt, Value: "10"},
			expect: true,
		},
		{
			name:   "exists — present field",
			cond:   orktypes.Condition{Field: "spec.image", Operator: orktypes.ConditionExists},
			expect: true,
		},
		{
			name:   "exists — absent field",
			cond:   orktypes.Condition{Field: "spec.missing", Operator: orktypes.ConditionExists},
			expect: false,
		},
		{
			name:   "notExists — absent field",
			cond:   orktypes.Condition{Field: "spec.missing", Operator: orktypes.ConditionNotExists},
			expect: true,
		},
		{
			name:   "notExists — present field",
			cond:   orktypes.Condition{Field: "spec.image", Operator: orktypes.ConditionNotExists},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluateConditions(obj, []orktypes.Condition{tt.cond})
			if result != tt.expect {
				t.Errorf("expected %v, got %v", tt.expect, result)
			}
		})
	}
}

func TestEvaluateConditions_NestedField(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels": map[string]interface{}{
					"tier": "premium",
				},
			},
			"spec": map[string]interface{}{
				"database": map[string]interface{}{
					"engine": "postgres",
				},
			},
		},
	}

	tests := []struct {
		name   string
		field  string
		equals string
		expect bool
	}{
		{"nested metadata label", "metadata.labels.tier", "premium", true},
		{"nested spec field", "spec.database.engine", "postgres", true},
		{"nested field no match", "spec.database.engine", "mysql", false},
		{"nested field missing", "spec.database.version", "14", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := orktypes.Condition{Field: tt.field, Equals: tt.equals}
			result := evaluateConditions(obj, []orktypes.Condition{cond})
			if result != tt.expect {
				t.Errorf("expected %v, got %v", tt.expect, result)
			}
		})
	}
}

func TestResolveField_Types(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"stringVal":  "hello",
				"intVal":     int64(42),
				"floatVal":   float64(3.14),
				"boolTrue":   true,
				"boolFalse":  false,
				"intAsFloat": float64(10),
			},
		},
	}

	tests := []struct {
		field    string
		expected string
	}{
		{"spec.stringVal", "hello"},
		{"spec.intVal", "42"},
		{"spec.floatVal", "3.14"},
		{"spec.boolTrue", "true"},
		{"spec.boolFalse", "false"},
		{"spec.intAsFloat", "10"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			cond := orktypes.Condition{Field: tt.field, Equals: tt.expected}
			if !evaluateConditions(obj, []orktypes.Condition{cond}) {
				t.Errorf("field %q: expected value %q not matched", tt.field, tt.expected)
			}
		})
	}
}
