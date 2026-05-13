// pkg/generate/pure_funcs_test.go
package generate

import (
	"testing"
)

// ── inferTypeFromValue ────────────────────────────────────────────────────────

func TestInferTypeFromValue_Int(t *testing.T) {
	if got := inferTypeFromValue(42); got != "integer" {
		t.Errorf("int must infer integer, got %q", got)
	}
}

func TestInferTypeFromValue_Int64(t *testing.T) {
	var v int64 = 100
	if got := inferTypeFromValue(v); got != "integer" {
		t.Errorf("int64 must infer integer, got %q", got)
	}
}

func TestInferTypeFromValue_Float64(t *testing.T) {
	if got := inferTypeFromValue(3.14); got != "integer" {
		t.Errorf("float64 must infer integer, got %q", got)
	}
}

func TestInferTypeFromValue_Bool(t *testing.T) {
	if got := inferTypeFromValue(true); got != "boolean" {
		t.Errorf("bool must infer boolean, got %q", got)
	}
}

func TestInferTypeFromValue_Slice(t *testing.T) {
	if got := inferTypeFromValue([]interface{}{"a"}); got != "array" {
		t.Errorf("slice must infer array, got %q", got)
	}
}

func TestInferTypeFromValue_Map(t *testing.T) {
	if got := inferTypeFromValue(map[string]interface{}{"k": "v"}); got != "object" {
		t.Errorf("map must infer object, got %q", got)
	}
}

func TestInferTypeFromValue_String(t *testing.T) {
	if got := inferTypeFromValue("hello"); got != "string" {
		t.Errorf("string must infer string, got %q", got)
	}
}

func TestInferTypeFromValue_Nil_FallsBackToString(t *testing.T) {
	if got := inferTypeFromValue(nil); got != "string" {
		t.Errorf("nil must fall back to string, got %q", got)
	}
}

// ── trimSpecPrefix ────────────────────────────────────────────────────────────

func TestTrimSpecPrefix_WithPrefix_RemovesIt(t *testing.T) {
	if got := trimSpecPrefix("spec.image"); got != "image" {
		t.Errorf("expected image, got %q", got)
	}
}

func TestTrimSpecPrefix_WithNestedPrefix(t *testing.T) {
	if got := trimSpecPrefix("spec.replicas"); got != "replicas" {
		t.Errorf("expected replicas, got %q", got)
	}
}

func TestTrimSpecPrefix_NoPrefix_ReturnedAsIs(t *testing.T) {
	if got := trimSpecPrefix("image"); got != "image" {
		t.Errorf("expected image unchanged, got %q", got)
	}
}

func TestTrimSpecPrefix_BareSpec_ReturnsEmpty(t *testing.T) {
	// "spec" (without dot) → the code checks HasPrefix("spec") and returns ""
	if got := trimSpecPrefix("spec"); got != "" {
		t.Errorf("bare spec must return empty, got %q", got)
	}
}

// ── placeholderFor ────────────────────────────────────────────────────────────

func TestPlaceholderFor_Image_ReturnsImagePlaceholder(t *testing.T) {
	got := placeholderFor("image", "string")
	if got != "my-image:latest" {
		t.Errorf("expected my-image:latest, got %v", got)
	}
}

func TestPlaceholderFor_Replicas_ReturnsInt(t *testing.T) {
	got := placeholderFor("replicas", "integer")
	if got != 1 {
		t.Errorf("expected 1, got %v", got)
	}
}

func TestPlaceholderFor_Port_ReturnsInt(t *testing.T) {
	got := placeholderFor("containerPort", "integer")
	if got != 8080 {
		t.Errorf("expected 8080 for port field, got %v", got)
	}
}

func TestPlaceholderFor_Region(t *testing.T) {
	got := placeholderFor("region", "string")
	if got != "us-east-1" {
		t.Errorf("expected us-east-1, got %v", got)
	}
}

func TestPlaceholderFor_Domain(t *testing.T) {
	got := placeholderFor("domain", "string")
	if got != "example.com" {
		t.Errorf("expected example.com, got %v", got)
	}
}

func TestPlaceholderFor_URL(t *testing.T) {
	got := placeholderFor("webhookUrl", "string")
	if got != "https://example.com" {
		t.Errorf("expected https://example.com, got %v", got)
	}
}

func TestPlaceholderFor_BoolType(t *testing.T) {
	got := placeholderFor("enabled", "boolean")
	if got != false {
		t.Errorf("expected false for boolean type, got %v", got)
	}
}

func TestPlaceholderFor_ArrayType(t *testing.T) {
	got := placeholderFor("tags", "array")
	arr, ok := got.([]interface{})
	if !ok || len(arr) != 0 {
		t.Errorf("expected empty array, got %v", got)
	}
}

func TestPlaceholderFor_IntegerType(t *testing.T) {
	got := placeholderFor("count", "integer")
	if got != 1 {
		t.Errorf("expected 1 for integer type, got %v", got)
	}
}

func TestPlaceholderFor_DefaultString(t *testing.T) {
	got := placeholderFor("tier", "string")
	if got != "my-tier" {
		t.Errorf("expected my-tier for unknown field, got %v", got)
	}
}

// ── toTitle ───────────────────────────────────────────────────────────────────

func TestToTitle_Empty_ReturnsEmpty(t *testing.T) {
	if got := toTitle(""); got != "" {
		t.Errorf("empty must return empty, got %q", got)
	}
}

func TestToTitle_Lowercase_CapitalizesFirst(t *testing.T) {
	if got := toTitle("hello"); got != "Hello" {
		t.Errorf("expected Hello, got %q", got)
	}
}

func TestToTitle_AlreadyCapitalized(t *testing.T) {
	if got := toTitle("World"); got != "World" {
		t.Errorf("expected World unchanged, got %q", got)
	}
}

func TestToTitle_SingleChar(t *testing.T) {
	if got := toTitle("a"); got != "A" {
		t.Errorf("expected A, got %q", got)
	}
}

// ── extractSpecPaths ──────────────────────────────────────────────────────────

func TestExtractSpecPaths_SingleMatch(t *testing.T) {
	// The regex captures only the field name after ".spec." — not the full path
	paths := extractSpecPaths([]string{"hello {{ .spec.image }} world"})
	if len(paths) != 1 || paths[0] != "image" {
		t.Errorf("expected [image], got %v", paths)
	}
}

func TestExtractSpecPaths_MultipleMatches_Sorted(t *testing.T) {
	paths := extractSpecPaths([]string{"{{ .spec.replicas }} and {{ .spec.image }}"})
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %v", paths)
	}
	// sorted alphabetically: image before replicas
	if paths[0] != "image" || paths[1] != "replicas" {
		t.Errorf("expected sorted [image replicas], got %v", paths)
	}
}

func TestExtractSpecPaths_Deduplicates(t *testing.T) {
	paths := extractSpecPaths([]string{"{{ .spec.image }}", "{{ .spec.image }}"})
	if len(paths) != 1 {
		t.Errorf("expected 1 unique path, got %v", paths)
	}
}

func TestExtractSpecPaths_NoMatch_ReturnsNil(t *testing.T) {
	paths := extractSpecPaths([]string{"no template here"})
	if len(paths) != 0 {
		t.Errorf("expected no paths, got %v", paths)
	}
}

func TestExtractSpecPaths_EmptySlice(t *testing.T) {
	paths := extractSpecPaths(nil)
	if len(paths) != 0 {
		t.Errorf("expected no paths from nil input, got %v", paths)
	}
}

// ── resolveAlias ──────────────────────────────────────────────────────────────

func TestResolveAlias_ExplicitAlias_ReturnsIt(t *testing.T) {
	got := resolveAlias("myalias", "prefix", "github.com/org/pkg/v1")
	if got != "myalias" {
		t.Errorf("expected explicit alias, got %q", got)
	}
}

func TestResolveAlias_NoExplicit_DerivesTwoSegments(t *testing.T) {
	got := resolveAlias("", "", "github.com/myorg/apis/project/v1alpha1")
	// last two parts: project + v1alpha1 → "projectv1alpha1"
	if got != "projectv1alpha1" {
		t.Errorf("expected projectv1alpha1, got %q", got)
	}
}

func TestResolveAlias_WithPrefix(t *testing.T) {
	got := resolveAlias("", "gen", "github.com/myorg/hooks")
	// last two parts: myorg + hooks → "myorghooks", with prefix "gen"
	if got != "genmyorghooks" {
		t.Errorf("expected genmyorghooks, got %q", got)
	}
}

func TestResolveAlias_DotInPath_Sanitized(t *testing.T) {
	got := resolveAlias("", "", "github.com/myorg/apis/v1.2.3")
	// dots removed in sanitization
	if got == "" {
		t.Error("expected non-empty alias")
	}
	// Should not contain dots
	for _, c := range got {
		if c == '.' {
			t.Errorf("alias must not contain dots, got %q", got)
		}
	}
}

// ── toRawExtension ────────────────────────────────────────────────────────────

func TestToRawExtension_String(t *testing.T) {
	j, err := toRawExtension("hello")
	if err != nil || string(j.Raw) != `"hello"` {
		t.Errorf("expected %q, got %s err=%v", `"hello"`, j.Raw, err)
	}
}

func TestToRawExtension_Int(t *testing.T) {
	j, err := toRawExtension(42)
	if err != nil || string(j.Raw) != "42" {
		t.Errorf("expected 42, got %s err=%v", j.Raw, err)
	}
}

func TestToRawExtension_Float64(t *testing.T) {
	j, err := toRawExtension(3.14)
	if err != nil || string(j.Raw) != "3.14" {
		t.Errorf("expected 3.14, got %s err=%v", j.Raw, err)
	}
}

func TestToRawExtension_BoolTrue(t *testing.T) {
	j, err := toRawExtension(true)
	if err != nil || string(j.Raw) != "true" {
		t.Errorf("expected true, got %s err=%v", j.Raw, err)
	}
}

func TestToRawExtension_BoolFalse(t *testing.T) {
	j, err := toRawExtension(false)
	if err != nil || string(j.Raw) != "false" {
		t.Errorf("expected false, got %s err=%v", j.Raw, err)
	}
}

func TestToRawExtension_Unknown_ReturnsNil(t *testing.T) {
	j, err := toRawExtension([]string{"a"})
	if err != nil || j != nil {
		t.Errorf("unknown type must return nil,nil — got %v err=%v", j, err)
	}
}

// ── dedupeImport ─────────────────────────────────────────────────────────────

func TestDedupeImport_NewAlias_NoError(t *testing.T) {
	seen := map[string]string{}
	if err := dedupeImport(seen, "myalias", "github.com/org/pkg", "website"); err != nil {
		t.Errorf("first occurrence must not error: %v", err)
	}
}

func TestDedupeImport_SameAliasAndLocation_NoError(t *testing.T) {
	seen := map[string]string{"myalias": "github.com/org/pkg"}
	if err := dedupeImport(seen, "myalias", "github.com/org/pkg", "website"); err != nil {
		t.Errorf("same alias+location must not error: %v", err)
	}
}

func TestDedupeImport_SameAliasDifferentLocation_Error(t *testing.T) {
	seen := map[string]string{"myalias": "github.com/org/pkg-a"}
	if err := dedupeImport(seen, "myalias", "github.com/org/pkg-b", "website"); err == nil {
		t.Error("conflicting alias must return error")
	}
}
