package controlcenter

import (
	"encoding/json"
	"testing"
)

// Regression test for the bug where every CRD-schema enum field silently
// rendered as a plain text box instead of a dropdown: buildSpecIDPField's
// switch once required prop.Type == "enum" before treating a field as a
// select, but CRD OpenAPI schemas never set type: enum — enum is always a
// constraint alongside type: string. The check must depend only on
// len(prop.Enum) > 0.
func TestBuildSpecIDPField_SchemaEnumRendersAsSelect(t *testing.T) {
	prop := idpSchemaProperty{
		Type: "string",
		Enum: []string{"app", "cert", "monitoring", "infra"},
	}
	f := buildSpecIDPField("workloadType", prop, idpFieldHint{}, false)

	if f.InputType != "select" {
		t.Fatalf("InputType = %q, want %q (prop.Type=%q with Enum set must still render as a dropdown)", f.InputType, "select", prop.Type)
	}
	if len(f.Enum) != 4 || f.Enum[0] != "app" {
		t.Fatalf("Enum = %v, want the schema's enum list carried through", f.Enum)
	}
}

func TestBuildSpecIDPField_InputTypes(t *testing.T) {
	cases := []struct {
		name string
		prop idpSchemaProperty
		want string
	}{
		{"enum wins even with a base type", idpSchemaProperty{Type: "string", Enum: []string{"a", "b"}}, "select"},
		{"boolean", idpSchemaProperty{Type: "boolean"}, "checkbox"},
		{"integer", idpSchemaProperty{Type: "integer"}, "number"},
		{"number", idpSchemaProperty{Type: "number"}, "number"},
		{"plain string", idpSchemaProperty{Type: "string"}, "text"},
		{"unknown/empty type", idpSchemaProperty{}, "text"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := buildSpecIDPField("field", c.prop, idpFieldHint{}, false)
			if f.InputType != c.want {
				t.Errorf("InputType = %q, want %q", f.InputType, c.want)
			}
		})
	}
}

func TestBuildSpecIDPField_LabelFallback(t *testing.T) {
	f := buildSpecIDPField("workloadType", idpSchemaProperty{}, idpFieldHint{}, false)
	if f.Label != "WorkloadType" {
		t.Errorf("Label = %q, want auto-capitalized field name %q", f.Label, "WorkloadType")
	}

	f = buildSpecIDPField("workloadType", idpSchemaProperty{}, idpFieldHint{Label: "Workload Type"}, false)
	if f.Label != "Workload Type" {
		t.Errorf("Label = %q, want hint.Label to override the fallback", f.Label)
	}
}

func TestBuildSpecIDPField_HintFallsBackToSchemaDescription(t *testing.T) {
	prop := idpSchemaProperty{Description: "Team that owns this resource"}

	f := buildSpecIDPField("team", prop, idpFieldHint{}, false)
	if f.Hint != prop.Description {
		t.Errorf("Hint = %q, want schema Description %q when hint.Hint is empty", f.Hint, prop.Description)
	}

	f = buildSpecIDPField("team", prop, idpFieldHint{Hint: "custom hint"}, false)
	if f.Hint != "custom hint" {
		t.Errorf("Hint = %q, want hint.Hint to take precedence over schema Description", f.Hint)
	}
}

func TestBuildSpecIDPField_Required(t *testing.T) {
	// required via the CRD schema's required: list
	f := buildSpecIDPField("team", idpSchemaProperty{}, idpFieldHint{}, true)
	if !f.Required {
		t.Error("Required = false, want true when the schema marks the field required")
	}

	// required via idp.fields.<name>.required, independent of the schema
	f = buildSpecIDPField("team", idpSchemaProperty{}, idpFieldHint{Required: true}, false)
	if !f.Required {
		t.Error("Required = false, want true when hint.Required is set")
	}

	f = buildSpecIDPField("team", idpSchemaProperty{}, idpFieldHint{}, false)
	if f.Required {
		t.Error("Required = true, want false when neither source marks it required")
	}
}

func TestBuildSpecIDPField_DefaultAndSource(t *testing.T) {
	prop := idpSchemaProperty{Default: 3}
	f := buildSpecIDPField("replicas", prop, idpFieldHint{}, false)
	if f.Default != "3" {
		t.Errorf("Default = %q, want %q", f.Default, "3")
	}
	if f.Source != IDPFieldSourceSpec {
		t.Errorf("Source = %q, want %q", f.Source, IDPFieldSourceSpec)
	}
}

func TestBuildSpecIDPField_WhenAnyOfEncodedAsJSON(t *testing.T) {
	hint := idpFieldHint{
		When:  []json.RawMessage{json.RawMessage(`{"field":"spec.workloadType","equals":"app"}`)},
		AnyOf: []json.RawMessage{json.RawMessage(`{"field":"spec.workloadType","equals":"cert"}`)},
	}
	f := buildSpecIDPField("repoURL", idpSchemaProperty{}, hint, false)

	var when []map[string]string
	if err := json.Unmarshal([]byte(f.WhenJSON), &when); err != nil {
		t.Fatalf("WhenJSON did not round-trip as JSON: %v", err)
	}
	if len(when) != 1 || when[0]["equals"] != "app" {
		t.Errorf("WhenJSON decoded to %v, want the single when: condition", when)
	}

	var anyOf []map[string]string
	if err := json.Unmarshal([]byte(f.AnyOfJSON), &anyOf); err != nil {
		t.Fatalf("AnyOfJSON did not round-trip as JSON: %v", err)
	}
	if len(anyOf) != 1 || anyOf[0]["equals"] != "cert" {
		t.Errorf("AnyOfJSON decoded to %v, want the single anyOf: condition", anyOf)
	}
}

func TestBuildAdditionalIDPField_InputTypes(t *testing.T) {
	cases := []struct {
		name string
		hint idpFieldHint
		want string
	}{
		{"enum with values", idpFieldHint{Type: "enum", Enum: []string{"a", "b"}}, "select"},
		{"enum type but no values falls through", idpFieldHint{Type: "enum"}, "text"},
		{"boolean", idpFieldHint{Type: "boolean"}, "checkbox"},
		{"integer", idpFieldHint{Type: "integer"}, "number"},
		{"number", idpFieldHint{Type: "number"}, "number"},
		{"string", idpFieldHint{Type: "string"}, "text"},
		{"type omitted defaults to text", idpFieldHint{}, "text"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := buildAdditionalIDPField("field", c.hint, IDPFieldSourceLabel)
			if f.InputType != c.want {
				t.Errorf("InputType = %q, want %q", f.InputType, c.want)
			}
		})
	}
}

func TestBuildAdditionalIDPField_LabelFallbackAndSource(t *testing.T) {
	f := buildAdditionalIDPField("team", idpFieldHint{}, IDPFieldSourceLabel)
	if f.Label != "team" {
		t.Errorf("Label = %q, want the raw key %q (additionalFields keys are not auto-capitalized)", f.Label, "team")
	}
	if f.Source != IDPFieldSourceLabel {
		t.Errorf("Source = %q, want %q", f.Source, IDPFieldSourceLabel)
	}

	f = buildAdditionalIDPField("platform.myorg.io/expose", idpFieldHint{Label: "Expose externally"}, IDPFieldSourceAnnotation)
	if f.Label != "Expose externally" {
		t.Errorf("Label = %q, want hint.Label to override the raw key", f.Label)
	}
	if f.Source != IDPFieldSourceAnnotation {
		t.Errorf("Source = %q, want %q", f.Source, IDPFieldSourceAnnotation)
	}
}

func TestBuildAdditionalIDPField_RequiredCategoryDisabled(t *testing.T) {
	f := buildAdditionalIDPField("team", idpFieldHint{Required: true, Category: "Ownership", Disabled: "locked for maintenance"}, IDPFieldSourceLabel)
	if !f.Required {
		t.Error("Required = false, want true")
	}
	if f.Category != "Ownership" {
		t.Errorf("Category = %q, want %q", f.Category, "Ownership")
	}
	if f.Disabled != "locked for maintenance" {
		t.Errorf("Disabled = %q, want %q", f.Disabled, "locked for maintenance")
	}
}
