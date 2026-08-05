package controlcenter

import (
	"encoding/json"
	"testing"
)

func TestBuildIDPField_InputTypes(t *testing.T) {
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
			f := buildIDPField("field", c.hint)
			if f.InputType != c.want {
				t.Errorf("InputType = %q, want %q", f.InputType, c.want)
			}
		})
	}
}

func TestBuildIDPField_LabelFallback(t *testing.T) {
	f := buildIDPField("workloadType", idpFieldHint{})
	if f.Label != "WorkloadType" {
		t.Errorf("Label = %q, want auto-capitalized field name %q", f.Label, "WorkloadType")
	}

	f = buildIDPField("workloadType", idpFieldHint{Label: "Workload Type"})
	if f.Label != "Workload Type" {
		t.Errorf("Label = %q, want hint.Label to override the fallback", f.Label)
	}
}

func TestBuildIDPField_Required(t *testing.T) {
	f := buildIDPField("team", idpFieldHint{Required: true})
	if !f.Required {
		t.Error("Required = false, want true when hint.Required is set")
	}

	f = buildIDPField("team", idpFieldHint{})
	if f.Required {
		t.Error("Required = true, want false when hint.Required is unset")
	}
}

func TestBuildIDPField_WhenAnyOfEncodedAsJSON(t *testing.T) {
	hint := idpFieldHint{
		When:  []json.RawMessage{json.RawMessage(`{"field":"spec.workloadType","equals":"app"}`)},
		AnyOf: []json.RawMessage{json.RawMessage(`{"field":"spec.workloadType","equals":"cert"}`)},
	}
	f := buildIDPField("repoURL", hint)

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

func TestBuildIDPField_CategoryAndDisabled(t *testing.T) {
	f := buildIDPField("team", idpFieldHint{Category: "Ownership", Disabled: "locked for maintenance"})
	if f.Category != "Ownership" {
		t.Errorf("Category = %q, want %q", f.Category, "Ownership")
	}
	if f.Disabled != "locked for maintenance" {
		t.Errorf("Disabled = %q, want %q", f.Disabled, "locked for maintenance")
	}
}
