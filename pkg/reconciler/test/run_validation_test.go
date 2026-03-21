// pkg/reconciler/test/run_validation_test.go
package reconciler_test

import (
	"testing"

	"github.com/ialexeze/orkestra/pkg/reconciler"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func validationObj(spec map[string]interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": "test-cr",
			},
			"spec": spec,
		},
	}
}

func TestRunValidation_NilConfig(t *testing.T) {
	obj := validationObj(map[string]interface{}{"image": "nginx"})
	result := reconciler.RunValidation(obj, nil, "website")
	if !result.Passed {
		t.Error("nil config should always pass")
	}
	if result.Error() != nil {
		t.Errorf("nil config should produce no error, got: %v", result.Error())
	}
}

func TestRunValidation_EmptyRules(t *testing.T) {
	obj := validationObj(map[string]interface{}{})
	cfg := &orktypes.ValidationConfig{Rules: []orktypes.ValidationRule{}}
	result := reconciler.RunValidation(obj, cfg, "website")
	if !result.Passed {
		t.Error("empty rules should always pass")
	}
}

func TestRunValidation_PrefixRule_Passes(t *testing.T) {
	obj := validationObj(map[string]interface{}{"image": "myorg/nginx:1.25"})
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:   "spec.image",
				Prefix:  "myorg/",
				Message: "image must be from myorg registry",
			},
		},
	}
	result := reconciler.RunValidation(obj, cfg, "website")
	if !result.Passed {
		t.Errorf("expected pass, got violations: %v", result.Violations)
	}
}

func TestRunValidation_PrefixRule_Fails(t *testing.T) {
	obj := validationObj(map[string]interface{}{"image": "nginx:1.25"})
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:   "spec.image",
				Prefix:  "myorg/",
				Message: "image must be from myorg registry",
			},
		},
	}
	result := reconciler.RunValidation(obj, cfg, "website")
	if result.Passed {
		t.Error("expected failure for non-myorg image")
	}
	if len(result.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(result.Violations))
	}
	if result.Violations[0].Field != "spec.image" {
		t.Errorf("expected violation on spec.image, got %q", result.Violations[0].Field)
	}
	if result.Error() == nil {
		t.Error("expected non-nil error for failed validation")
	}
}

func TestRunValidation_MaxRule_Passes(t *testing.T) {
	obj := validationObj(map[string]interface{}{"replicas": int64(5)})
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:   "spec.replicas",
				Max:     "10",
				Message: "replicas cannot exceed 10",
			},
		},
	}
	result := reconciler.RunValidation(obj, cfg, "website")
	if !result.Passed {
		t.Errorf("expected pass for replicas=5, max=10: %v", result.Violations)
	}
}

func TestRunValidation_MaxRule_Fails(t *testing.T) {
	obj := validationObj(map[string]interface{}{"replicas": int64(15)})
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:   "spec.replicas",
				Max:     "10",
				Message: "replicas cannot exceed 10",
			},
		},
	}
	result := reconciler.RunValidation(obj, cfg, "website")
	if result.Passed {
		t.Error("expected failure for replicas=15, max=10")
	}
}

func TestRunValidation_MultipleRules_AllFail(t *testing.T) {
	obj := validationObj(map[string]interface{}{
		"image":    "nginx:1.25", // fails prefix check
		"replicas": int64(20),    // fails max check
	})
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.image", Prefix: "myorg/", Message: "bad image"},
			{Field: "spec.replicas", Max: "10", Message: "too many replicas"},
		},
	}
	result := reconciler.RunValidation(obj, cfg, "website")
	if result.Passed {
		t.Error("expected failure when multiple rules fail")
	}
	if len(result.Violations) != 2 {
		t.Errorf("expected 2 violations, got %d: %v", len(result.Violations), result.Violations)
	}
}

func TestRunValidation_MultipleRules_SomePass(t *testing.T) {
	obj := validationObj(map[string]interface{}{
		"image":    "myorg/nginx:1.25", // passes
		"replicas": int64(20),          // fails
	})
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.image", Prefix: "myorg/", Message: "bad image"},
			{Field: "spec.replicas", Max: "10", Message: "too many replicas"},
		},
	}
	result := reconciler.RunValidation(obj, cfg, "website")
	if result.Passed {
		t.Error("expected overall failure when any rule fails")
	}
	if len(result.Violations) != 1 {
		t.Errorf("expected 1 violation (replicas), got %d", len(result.Violations))
	}
}

func TestRunValidation_EqualsShorthand(t *testing.T) {
	obj := validationObj(map[string]interface{}{"environment": "production"})
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:   "spec.environment",
				Equals:  "production",
				Message: "only production is supported",
			},
		},
	}
	result := reconciler.RunValidation(obj, cfg, "website")
	if !result.Passed {
		t.Errorf("equals shorthand should pass: %v", result.Violations)
	}
}

func TestRunValidation_MissingField_Fails(t *testing.T) {
	// Field doesn't exist — validation requiring it to equal something should fail
	obj := validationObj(map[string]interface{}{})
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:   "spec.image",
				Prefix:  "myorg/",
				Message: "image required and must be from myorg",
			},
		},
	}
	result := reconciler.RunValidation(obj, cfg, "website")
	if result.Passed {
		t.Error("missing required field should fail validation")
	}
}

func TestRunValidation_ErrorMessage(t *testing.T) {
	obj := validationObj(map[string]interface{}{"image": "nginx:1.25"})
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:   "spec.image",
				Prefix:  "myorg/",
				Message: "image must be from myorg registry",
			},
		},
	}
	result := reconciler.RunValidation(obj, cfg, "website")
	err := result.Error()
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	errStr := err.Error()
	if !contains(errStr, "spec.image") {
		t.Errorf("error should mention field name, got: %q", errStr)
	}
	if !contains(errStr, "image must be from myorg registry") {
		t.Errorf("error should include the user message, got: %q", errStr)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
