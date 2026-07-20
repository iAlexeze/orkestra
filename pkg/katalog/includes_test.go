package katalog_test

import (
	"testing"
)

// ── idp.include ───────────────────────────────────────────────────────────────

func TestIDPInclude_ExpandsFields(t *testing.T) {
	k := mustParseTestdata(t, "include/idp.yaml")
	crd, _ := k.Get("platform")
	if crd == nil {
		t.Fatal("platform CRD not found")
	}
	if crd.IDP == nil {
		t.Fatal("IDP is nil")
	}
	// Included fields
	if _, ok := crd.IDP.Fields["team"]; !ok {
		t.Error("included field 'team' missing")
	}
	if _, ok := crd.IDP.Fields["environment"]; !ok {
		t.Error("included field 'environment' missing")
	}
	// Inline field merged on top
	if _, ok := crd.IDP.Fields["image"]; !ok {
		t.Error("inline field 'image' missing")
	}
}

func TestIDPInclude_InlineOverridesIncluded(t *testing.T) {
	k := mustParseTestdata(t, "include/idp.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.IDP == nil {
		t.Fatal("platform CRD or IDP is nil")
	}
	// 'team' from include has order 1; if an inline field with the same name
	// were present it would win. Here 'image' is inline-only with order 3.
	if f, ok := crd.IDP.Fields["image"]; !ok || f.Order != 3 {
		t.Errorf("inline 'image' field: got order %d, want 3", crd.IDP.Fields["image"].Order)
	}
}

func TestIDPInclude_ClearedAfterExpansion(t *testing.T) {
	k := mustParseTestdata(t, "include/idp.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.IDP == nil {
		t.Fatal("platform CRD or IDP is nil")
	}
	if crd.IDP.Include != "" {
		t.Errorf("IDP.Include not cleared after expansion, got %q", crd.IDP.Include)
	}
}

// ── validation.include ────────────────────────────────────────────────────────

func TestValidationInclude_ExpandsRules(t *testing.T) {
	k := mustParseTestdata(t, "include/validation.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.Validation == nil {
		t.Fatal("platform CRD or Validation is nil")
	}
	// 2 included + 1 inline = 3 total
	if len(crd.Validation.Rules) != 3 {
		t.Errorf("len(Rules) = %d, want 3", len(crd.Validation.Rules))
	}
}

func TestValidationInclude_IncludedRulesFirst(t *testing.T) {
	k := mustParseTestdata(t, "include/validation.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.Validation == nil {
		t.Fatal("platform CRD or Validation is nil")
	}
	// First rule must be the included one (spec.team exists)
	if got := crd.Validation.Rules[0].Field; got != "spec.team" {
		t.Errorf("Rules[0].Field = %q, want %q", got, "spec.team")
	}
	// Last rule is the inline one (spec.replicas lte)
	last := crd.Validation.Rules[len(crd.Validation.Rules)-1]
	if last.Field != "spec.replicas" {
		t.Errorf("last rule Field = %q, want %q", last.Field, "spec.replicas")
	}
}

func TestValidationInclude_ClearedAfterExpansion(t *testing.T) {
	k := mustParseTestdata(t, "include/validation.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.Validation == nil {
		t.Fatal("platform CRD or Validation is nil")
	}
	if crd.Validation.Include != "" {
		t.Errorf("Validation.Include not cleared after expansion, got %q", crd.Validation.Include)
	}
}

// ── mutation.include ──────────────────────────────────────────────────────────

func TestMutationInclude_ExpandsRules(t *testing.T) {
	k := mustParseTestdata(t, "include/mutation.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.Mutation == nil {
		t.Fatal("platform CRD or Mutation is nil")
	}
	// 2 included + 1 inline = 3 total
	if len(crd.Mutation.Rules) != 3 {
		t.Errorf("len(Rules) = %d, want 3", len(crd.Mutation.Rules))
	}
}

func TestMutationInclude_IncludedRulesFirst(t *testing.T) {
	k := mustParseTestdata(t, "include/mutation.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.Mutation == nil {
		t.Fatal("platform CRD or Mutation is nil")
	}
	// First rule must be the included one (spec.replicas default)
	if got := crd.Mutation.Rules[0].Field; got != "spec.replicas" {
		t.Errorf("Rules[0].Field = %q, want %q", got, "spec.replicas")
	}
	// Last rule is the inline one (spec.logLevel default)
	last := crd.Mutation.Rules[len(crd.Mutation.Rules)-1]
	if last.Field != "spec.logLevel" {
		t.Errorf("last rule Field = %q, want %q", last.Field, "spec.logLevel")
	}
}

func TestMutationInclude_ClearedAfterExpansion(t *testing.T) {
	k := mustParseTestdata(t, "include/mutation.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.Mutation == nil {
		t.Fatal("platform CRD or Mutation is nil")
	}
	if crd.Mutation.Include != "" {
		t.Errorf("Mutation.Include not cleared after expansion, got %q", crd.Mutation.Include)
	}
}

// ── conversion.include ────────────────────────────────────────────────────────

func TestConversionInclude_ExpandsPaths(t *testing.T) {
	k := mustParseTestdata(t, "include/conversion.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.Conversion == nil {
		t.Fatal("platform CRD or Conversion is nil")
	}
	// 2 included + 1 inline = 3 total
	if len(crd.Conversion.Paths) != 3 {
		t.Errorf("len(Paths) = %d, want 3", len(crd.Conversion.Paths))
	}
}

func TestConversionInclude_IncludedPathsFirst(t *testing.T) {
	k := mustParseTestdata(t, "include/conversion.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.Conversion == nil {
		t.Fatal("platform CRD or Conversion is nil")
	}
	// First path is from the include file (v1alpha1 → v1)
	if got := crd.Conversion.Paths[0].From; got != "v1alpha1" {
		t.Errorf("Paths[0].From = %q, want %q", got, "v1alpha1")
	}
	// Last path is the inline one (v1 → v1alpha1)
	last := crd.Conversion.Paths[len(crd.Conversion.Paths)-1]
	if last.From != "v1" {
		t.Errorf("last path From = %q, want %q", last.From, "v1")
	}
}

func TestConversionInclude_ClearedAfterExpansion(t *testing.T) {
	k := mustParseTestdata(t, "include/conversion.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.Conversion == nil {
		t.Fatal("platform CRD or Conversion is nil")
	}
	if crd.Conversion.Include != "" {
		t.Errorf("Conversion.Include not cleared after expansion, got %q", crd.Conversion.Include)
	}
}

// ── status.include ────────────────────────────────────────────────────────────

func TestStatusInclude_ExpandsFields(t *testing.T) {
	k := mustParseTestdata(t, "include/status.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.OperatorBox.Status == nil {
		t.Fatal("platform CRD or Status is nil")
	}
	// 2 included + 1 inline = 3 total
	if len(crd.OperatorBox.Status.Fields) != 3 {
		t.Errorf("len(Fields) = %d, want 3", len(crd.OperatorBox.Status.Fields))
	}
}

func TestStatusInclude_IncludedFieldsFirst(t *testing.T) {
	k := mustParseTestdata(t, "include/status.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.OperatorBox.Status == nil {
		t.Fatal("platform CRD or Status is nil")
	}
	// First field is the included one (phase)
	if got := crd.OperatorBox.Status.Fields[0].Path; got != "phase" {
		t.Errorf("Fields[0].Path = %q, want %q", got, "phase")
	}
	// Last field is the inline one (environment)
	last := crd.OperatorBox.Status.Fields[len(crd.OperatorBox.Status.Fields)-1]
	if last.Path != "environment" {
		t.Errorf("last field Path = %q, want %q", last.Path, "environment")
	}
}

func TestStatusInclude_ClearedAfterExpansion(t *testing.T) {
	k := mustParseTestdata(t, "include/status.yaml")
	crd, _ := k.Get("platform")
	if crd == nil || crd.OperatorBox.Status == nil {
		t.Fatal("platform CRD or Status is nil")
	}
	if crd.OperatorBox.Status.Include != "" {
		t.Errorf("Status.Include not cleared after expansion, got %q", crd.OperatorBox.Status.Include)
	}
}
