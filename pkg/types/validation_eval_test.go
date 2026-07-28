package types

import "testing"

// ── ScalarToString ──────────────────────────────────────────────────────────

func TestScalarToString(t *testing.T) {
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
			if got := ScalarToString(tc.input); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// ── ResolveScalarField ──────────────────────────────────────────────────────

func TestResolveScalarField(t *testing.T) {
	data := map[string]interface{}{
		"apiVersion": "demo.orkestra.io/v1alpha1",
		"metadata": map[string]interface{}{
			"name":      "my-site",
			"namespace": "default",
			"labels": map[string]interface{}{
				"team": "platform",
			},
		},
		"spec": map[string]interface{}{
			"stringVal":  "hello",
			"intVal":     int64(42),
			"floatVal":   float64(3.14),
			"boolTrue":   true,
			"boolFalse":  false,
			"intAsFloat": float64(10),
			"image":      "myorg/nginx:1.25",
			"replicas":   float64(3),
			"enabled":    true,
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
		{"top-level string", "apiVersion", "demo.orkestra.io/v1alpha1", true},
		{"nested string", "spec.stringVal", "hello", true},
		{"nested int", "spec.intVal", "42", true},
		{"nested float", "spec.floatVal", "3.14", true},
		{"nested bool true", "spec.boolTrue", "true", true},
		{"nested bool false", "spec.boolFalse", "false", true},
		{"float that is a whole number", "spec.intAsFloat", "10", true},
		{"nested float64 (int)", "spec.replicas", "3", true},
		{"nested bool", "spec.enabled", "true", true},
		{"deeply nested", "spec.db.engine", "postgres", true},
		{"nested under metadata", "metadata.name", "my-site", true},
		{"three levels deep", "metadata.labels.team", "platform", true},
		{"missing top-level", "status", "", false},
		{"missing nested", "spec.port", "", false},
		{"missing deeply nested", "spec.db.host", "", false},
		{"intermediate is a leaf not a map", "spec.image.subfield", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := ResolveScalarField(data, tt.path)
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v", found, tt.wantFound)
			}
			if got != tt.wantValue {
				t.Errorf("value = %q, want %q", got, tt.wantValue)
			}
		})
	}
}

func TestResolveScalarField_EmptyObject(t *testing.T) {
	if val, found := ResolveScalarField(map[string]interface{}{}, "spec.image"); found || val != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", val, found)
	}
}

// ── ResolveValidationOp ──────────────────────────────────────────────────────

func TestResolveValidationOp_Shorthands(t *testing.T) {
	tests := []struct {
		name    string
		rule    ValidationRule
		wantOp  ConditionOperator
		wantVal string
	}{
		{"equals", ValidationRule{Equals: "production"}, ConditionEquals, "production"},
		{"notEquals", ValidationRule{NotEquals: "test"}, ConditionNotEquals, "test"},
		{"prefix", ValidationRule{Prefix: "myorg/"}, ConditionPrefix, "myorg/"},
		{"suffix", ValidationRule{Suffix: ":latest"}, ConditionSuffix, ":latest"},
		{"contains", ValidationRule{Contains: "nginx"}, ConditionContains, "nginx"},
		{"min maps to gt", ValidationRule{Min: "1"}, ConditionGt, "1"},
		{"max maps to lt", ValidationRule{Max: "10"}, ConditionLt, "10"},
		{"greaterThan maps to gt", ValidationRule{GreaterThan: "5"}, ConditionGt, "5"},
		{"lessThan maps to lt", ValidationRule{LessThan: "5"}, ConditionLt, "5"},
		{"explicit operator", ValidationRule{Operator: ConditionIn, Value: "a,b"}, ConditionIn, "a,b"},
		{"no shorthand defaults to exists", ValidationRule{Field: "spec.image"}, ConditionExists, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, val := ResolveValidationOp(tt.rule)
			if op != tt.wantOp || val != tt.wantVal {
				t.Errorf("got (%q, %q), want (%q, %q)", op, val, tt.wantOp, tt.wantVal)
			}
		})
	}
}

// ── EvaluateValidationRule ────────────────────────────────────────────────────

func specData(fields map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"spec": fields}
}

func TestEvaluateValidationRule_Operators(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		rule     ValidationRule
		wantPass bool
	}{
		// ── exists ──────────────────────────────────────────────────────────
		{"exists: field present with value — passes", specData(map[string]interface{}{"image": "myorg/app:v1"}), ValidationRule{Field: "spec.image", Operator: ConditionExists, Message: "required"}, true},
		{"exists: field missing — fails", specData(map[string]interface{}{}), ValidationRule{Field: "spec.image", Operator: ConditionExists, Message: "required"}, false},
		{"exists: field present but empty string — fails", specData(map[string]interface{}{"image": ""}), ValidationRule{Field: "spec.image", Operator: ConditionExists, Message: "required"}, false},

		// ── notExists ───────────────────────────────────────────────────────
		{"notExists: field absent — passes", specData(map[string]interface{}{}), ValidationRule{Field: "spec.debug", Operator: ConditionNotExists, Message: "debug not allowed"}, true},
		{"notExists: field present with value — fails", specData(map[string]interface{}{"debug": "true"}), ValidationRule{Field: "spec.debug", Operator: ConditionNotExists, Message: "debug not allowed"}, false},
		{"notExists: field present but empty — passes", specData(map[string]interface{}{"debug": ""}), ValidationRule{Field: "spec.debug", Operator: ConditionNotExists, Message: "debug not allowed"}, true},

		// ── equals ──────────────────────────────────────────────────────────
		{"equals: exact match — passes", specData(map[string]interface{}{"env": "production"}), ValidationRule{Field: "spec.env", Equals: "production", Message: "must be production"}, true},
		{"equals: no match — fails", specData(map[string]interface{}{"env": "staging"}), ValidationRule{Field: "spec.env", Equals: "production", Message: "must be production"}, false},
		{"equals: field missing — fails", specData(map[string]interface{}{}), ValidationRule{Field: "spec.env", Equals: "production", Message: "must be production"}, false},

		// ── notEquals ───────────────────────────────────────────────────────
		{"notEquals: different value — passes", specData(map[string]interface{}{"env": "staging"}), ValidationRule{Field: "spec.env", NotEquals: "production", Message: "must not be production"}, true},
		{"notEquals: matching value — fails", specData(map[string]interface{}{"env": "production"}), ValidationRule{Field: "spec.env", NotEquals: "production", Message: "must not be production"}, false},
		{"notEquals: field missing — passes", specData(map[string]interface{}{}), ValidationRule{Field: "spec.env", NotEquals: "production", Message: "must not be production"}, true},

		// ── prefix ──────────────────────────────────────────────────────────
		{"prefix: correct prefix — passes", specData(map[string]interface{}{"image": "myorg/nginx:1.25"}), ValidationRule{Field: "spec.image", Prefix: "myorg/", Message: "must use myorg registry"}, true},
		{"prefix: wrong prefix — fails", specData(map[string]interface{}{"image": "docker.io/nginx:1.25"}), ValidationRule{Field: "spec.image", Prefix: "myorg/", Message: "must use myorg registry"}, false},
		{"prefix: field missing — fails", specData(map[string]interface{}{}), ValidationRule{Field: "spec.image", Prefix: "myorg/", Message: "must use myorg registry"}, false},

		// ── suffix ──────────────────────────────────────────────────────────
		{"suffix: correct suffix — passes", specData(map[string]interface{}{"image": "myorg/app:v1.0"}), ValidationRule{Field: "spec.image", Suffix: ":v1.0", Message: "must use v1.0 tag"}, true},
		{"suffix: wrong suffix — fails", specData(map[string]interface{}{"image": "myorg/app:latest"}), ValidationRule{Field: "spec.image", Suffix: ":v1.0", Message: "must use v1.0 tag"}, false},

		// ── contains ────────────────────────────────────────────────────────
		{"contains: substring present — passes", specData(map[string]interface{}{"image": "registry.myorg.io/nginx:1.25"}), ValidationRule{Field: "spec.image", Contains: "myorg", Message: "must be from myorg"}, true},
		{"contains: substring absent — fails", specData(map[string]interface{}{"image": "docker.io/nginx:1.25"}), ValidationRule{Field: "spec.image", Contains: "myorg", Message: "must be from myorg"}, false},

		// ── in ──────────────────────────────────────────────────────────────
		// The exact regression this suite guards: operator: in was defined
		// for when:/anyOf: conditions but missing from the validation-rule
		// evaluation switch, so a rule using it always passed silently, in
		// both the reconciler and the webhook, at once.
		{"in: value in list — passes", specData(map[string]interface{}{"workloadType": "app"}), ValidationRule{Field: "spec.workloadType", Operator: ConditionIn, Value: "app,cert,monitoring,infra", Message: "invalid"}, true},
		{"in: value not in list — fails", specData(map[string]interface{}{"workloadType": "bogus"}), ValidationRule{Field: "spec.workloadType", Operator: ConditionIn, Value: "app,cert,monitoring,infra", Message: "invalid"}, false},
		{"in: field missing — fails", specData(map[string]interface{}{}), ValidationRule{Field: "spec.workloadType", Operator: ConditionIn, Value: "app,cert", Message: "invalid"}, false},
		{"in: whitespace around list entries is trimmed", specData(map[string]interface{}{"env": "prod"}), ValidationRule{Field: "spec.env", Operator: ConditionIn, Value: "dev, staging, prod", Message: "invalid"}, true},

		// ── min (ConditionGt) ───────────────────────────────────────────────
		{"min: value exactly at minimum — passes", specData(map[string]interface{}{"replicas": float64(1)}), ValidationRule{Field: "spec.replicas", Min: "1", Message: "at least 1 replica"}, true},
		{"min: value above minimum — passes", specData(map[string]interface{}{"replicas": float64(5)}), ValidationRule{Field: "spec.replicas", Min: "1", Message: "at least 1 replica"}, true},
		{"min: value below minimum — fails", specData(map[string]interface{}{"replicas": float64(0)}), ValidationRule{Field: "spec.replicas", Min: "1", Message: "at least 1 replica"}, false},
		{"min: field missing — fails", specData(map[string]interface{}{}), ValidationRule{Field: "spec.replicas", Min: "1", Message: "at least 1 replica"}, false},
		{"min: non-numeric field value — fails", specData(map[string]interface{}{"replicas": "lots"}), ValidationRule{Field: "spec.replicas", Min: "1", Message: "at least 1 replica"}, false},
		{"min: non-numeric config value — rule skipped (passes)", specData(map[string]interface{}{"replicas": float64(5)}), ValidationRule{Field: "spec.replicas", Min: "not-a-number", Message: "invalid config"}, true},

		// ── max (ConditionLt) ───────────────────────────────────────────────
		{"max: value at maximum — passes", specData(map[string]interface{}{"replicas": float64(10)}), ValidationRule{Field: "spec.replicas", Max: "10", Message: "no more than 10"}, true},
		{"max: value above maximum — fails", specData(map[string]interface{}{"replicas": float64(15)}), ValidationRule{Field: "spec.replicas", Max: "10", Message: "no more than 10"}, false},
		{"max: value below maximum — passes", specData(map[string]interface{}{"replicas": float64(3)}), ValidationRule{Field: "spec.replicas", Max: "10", Message: "no more than 10"}, true},
		{"max: field missing — fails", specData(map[string]interface{}{}), ValidationRule{Field: "spec.replicas", Max: "10", Message: "no more than 10"}, false},
		{"max: non-numeric field value — fails", specData(map[string]interface{}{"replicas": "many"}), ValidationRule{Field: "spec.replicas", Max: "10", Message: "no more than 10"}, false},
		{"max: non-numeric config value — rule skipped (passes)", specData(map[string]interface{}{"replicas": float64(5)}), ValidationRule{Field: "spec.replicas", Max: "not-a-number", Message: "invalid config"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := EvaluateValidationRule(tt.data, nil, tt.rule)
			if tt.wantPass && v != nil {
				t.Errorf("expected rule to pass — got violation: %+v", v)
			}
			if !tt.wantPass && v == nil {
				t.Error("expected rule to fail — got nil violation")
			}
		})
	}
}

func TestEvaluateValidationRule_ViolationFieldsArePopulated(t *testing.T) {
	data := specData(map[string]interface{}{"image": "docker.io/nginx:1.25"})
	rule := ValidationRule{
		Field:   "spec.image",
		Prefix:  "myorg/",
		Message: "image must be from the myorg registry",
		Action:  ValidationActionDeny,
	}

	v := EvaluateValidationRule(data, nil, rule)

	if v == nil {
		t.Fatal("expected a violation")
	}
	if v.Field != "spec.image" {
		t.Errorf("Field = %q, want %q", v.Field, "spec.image")
	}
	if v.Message != "image must be from the myorg registry" {
		t.Errorf("Message = %q, want %q", v.Message, "image must be from the myorg registry")
	}
	if v.Value != "docker.io/nginx:1.25" {
		t.Errorf("Value = %q, want %q", v.Value, "docker.io/nginx:1.25")
	}
	if v.Rule != string(ConditionPrefix) {
		t.Errorf("Rule = %q, want %q", v.Rule, string(ConditionPrefix))
	}
}
