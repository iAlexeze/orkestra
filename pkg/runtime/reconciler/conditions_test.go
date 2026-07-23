// pkg/reconciler/conditions_test.go
package reconciler

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// buildData builds a minimal CR data map for condition tests.
func buildData(spec map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": "demo.orkestra.io/v1alpha1",
		"kind":       "Website",
		"metadata": map[string]interface{}{
			"name":      "my-website",
			"namespace": "default",
		},
		"spec": spec,
	}
}

func eval(data map[string]interface{}, conds []orktypes.Condition) bool {
	return orktypes.EvaluateWhen(data, conds, nil, nil)
}

func TestEvaluateConditions_Empty(t *testing.T) {
	data := buildData(map[string]interface{}{})
	if !eval(data, nil) {
		t.Error("nil conditions should always return true")
	}
	if !eval(data, []orktypes.Condition{}) {
		t.Error("empty slice should always return true")
	}
}

func TestEvaluateConditions_EqualsShorthand(t *testing.T) {
	data := buildData(map[string]interface{}{"environment": "production"})

	if !eval(data, []orktypes.Condition{{Field: "spec.environment", Equals: "production"}}) {
		t.Error("equals shorthand should pass when value matches")
	}
	if eval(data, []orktypes.Condition{{Field: "spec.environment", Equals: "staging"}}) {
		t.Error("equals shorthand should fail when value does not match")
	}
}

func TestEvaluateConditions_AllMustPass(t *testing.T) {
	data := buildData(map[string]interface{}{
		"environment": "production",
		"enabled":     "true",
	})

	if !eval(data, []orktypes.Condition{
		{Field: "spec.environment", Equals: "production"},
		{Field: "spec.enabled", Equals: "true"},
	}) {
		t.Error("all conditions pass — should return true")
	}
	if eval(data, []orktypes.Condition{
		{Field: "spec.environment", Equals: "production"},
		{Field: "spec.enabled", Equals: "false"},
	}) {
		t.Error("one condition fails — should return false")
	}
}

func TestEvaluateConditions_Operators(t *testing.T) {
	data := buildData(map[string]interface{}{
		"image":    "myorg/nginx:1.25",
		"replicas": int64(5),
		"logLevel": "info",
	})

	tests := []struct {
		name   string
		cond   orktypes.Condition
		expect bool
	}{
		{"prefix matches", orktypes.Condition{Field: "spec.image", Operator: orktypes.ConditionPrefix, Value: "myorg/"}, true},
		{"prefix no match", orktypes.Condition{Field: "spec.image", Operator: orktypes.ConditionPrefix, Value: "other/"}, false},
		{"suffix matches", orktypes.Condition{Field: "spec.image", Operator: orktypes.ConditionSuffix, Value: ":1.25"}, true},
		{"contains matches", orktypes.Condition{Field: "spec.image", Operator: orktypes.ConditionContains, Value: "nginx"}, true},
		{"notEquals matches", orktypes.Condition{Field: "spec.logLevel", Operator: orktypes.ConditionNotEquals, Value: "debug"}, true},
		{"gt matches — 5 > 3", orktypes.Condition{Field: "spec.replicas", Operator: orktypes.ConditionGt, Value: "3"}, true},
		{"gt fails — 5 not > 10", orktypes.Condition{Field: "spec.replicas", Operator: orktypes.ConditionGt, Value: "10"}, false},
		{"lt matches — 5 < 10", orktypes.Condition{Field: "spec.replicas", Operator: orktypes.ConditionLt, Value: "10"}, true},
		{"exists — present field", orktypes.Condition{Field: "spec.image", Operator: orktypes.ConditionExists}, true},
		{"exists — absent field", orktypes.Condition{Field: "spec.missing", Operator: orktypes.ConditionExists}, false},
		{"notExists — absent field", orktypes.Condition{Field: "spec.missing", Operator: orktypes.ConditionNotExists}, true},
		{"notExists — present field", orktypes.Condition{Field: "spec.image", Operator: orktypes.ConditionNotExists}, false},
		{"in — value in list", orktypes.Condition{Field: "spec.logLevel", Operator: orktypes.ConditionIn, Value: "debug,info,warn"}, true},
		{"in — value not in list", orktypes.Condition{Field: "spec.logLevel", Operator: orktypes.ConditionIn, Value: "debug,warn"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if eval(data, []orktypes.Condition{tt.cond}) != tt.expect {
				t.Errorf("expected %v, got %v", tt.expect, !tt.expect)
			}
		})
	}
}

func TestEvaluateConditions_NestedField(t *testing.T) {
	data := map[string]interface{}{
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
			if eval(data, []orktypes.Condition{cond}) != tt.expect {
				t.Errorf("expected %v, got %v", tt.expect, !tt.expect)
			}
		})
	}
}

func TestEvaluateConditions_AnyOf(t *testing.T) {
	data := buildData(map[string]interface{}{"phase": "Failed"})

	// anyOf — OR semantics
	anyOf := []orktypes.Condition{
		{Field: "spec.phase", Equals: "Failed"},
		{Field: "spec.phase", Equals: "Succeeded"},
	}
	if !orktypes.EvaluateWhen(data, nil, anyOf, nil) {
		t.Error("anyOf should pass when at least one condition matches")
	}

	anyOf = []orktypes.Condition{
		{Field: "spec.phase", Equals: "Pending"},
		{Field: "spec.phase", Equals: "Running"},
	}
	if orktypes.EvaluateWhen(data, nil, anyOf, nil) {
		t.Error("anyOf should fail when no conditions match")
	}
}

func TestResolveField_Types(t *testing.T) {
	data := map[string]interface{}{
		"spec": map[string]interface{}{
			"stringVal":  "hello",
			"intVal":     int64(42),
			"floatVal":   float64(3.14),
			"boolTrue":   true,
			"boolFalse":  false,
			"intAsFloat": float64(10),
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
			got, found := resolveField(data, tt.field)
			if !found {
				t.Fatalf("field %q not found", tt.field)
			}
			if got != tt.expected {
				t.Errorf("field %q: got %q, want %q", tt.field, got, tt.expected)
			}
		})
	}
}
