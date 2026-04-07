//go:build integration

// tests/integration/reconciler/secret_test.go
// Integration tests for validation rules applied to secret-like CRD objects.
package reconciler_test

import (
	"testing"

	"github.com/ialexeze/orkestra/pkg/reconciler"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func secretObj(fields map[string]interface{}) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
	obj.SetName("test-secret")
	obj.SetNamespace("default")
	for k, v := range fields {
		obj.Object[k] = v
	}
	return obj
}

func TestSecretValidation_SecretNameRequired(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:    "spec.secretName",
				Operator: orktypes.ConditionExists,
				Message:  "spec.secretName is required",
			},
		},
	}

	withSecret := secretObj(map[string]interface{}{
		"spec": map[string]interface{}{"secretName": "db-credentials"},
	})
	if r := reconciler.RunValidation(withSecret, cfg, "secret"); !r.Passed {
		t.Errorf("spec.secretName present should pass: %v", r.ViolationSummary())
	}

	withoutSecret := secretObj(map[string]interface{}{
		"spec": map[string]interface{}{},
	})
	if r := reconciler.RunValidation(withoutSecret, cfg, "secret"); r.Passed {
		t.Error("missing spec.secretName should fail")
	}
}

func TestSecretValidation_TypeMustNotBeOpaque(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.type", NotEquals: "Opaque", Message: "Opaque secrets are not allowed; use a typed secret"},
		},
	}

	opaque := secretObj(map[string]interface{}{
		"spec": map[string]interface{}{"type": "Opaque"},
	})
	if r := reconciler.RunValidation(opaque, cfg, "secret"); r.Passed {
		t.Error("Opaque type should be rejected")
	}

	typed := secretObj(map[string]interface{}{
		"spec": map[string]interface{}{"type": "kubernetes.io/tls"},
	})
	if r := reconciler.RunValidation(typed, cfg, "secret"); !r.Passed {
		t.Errorf("typed secret should pass: %v", r.ViolationSummary())
	}
}

func TestSecretValidation_RotationIntervalMin(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.rotationDays", Min: "7", Message: "rotation interval must be at least 7 days"},
		},
	}

	validRotation := secretObj(map[string]interface{}{
		"spec": map[string]interface{}{"rotationDays": "30"},
	})
	if r := reconciler.RunValidation(validRotation, cfg, "secret"); !r.Passed {
		t.Errorf("30-day rotation should pass: %v", r.ViolationSummary())
	}

	tooFrequent := secretObj(map[string]interface{}{
		"spec": map[string]interface{}{"rotationDays": "1"},
	})
	if r := reconciler.RunValidation(tooFrequent, cfg, "secret"); r.Passed {
		t.Error("1-day rotation should be rejected (< 7 day minimum)")
	}
}

func TestSecretValidation_NameMustNotContainPassword(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:    "spec.secretName",
				Operator: orktypes.ConditionNotEquals,
				Value:    "password",
				Message:  "secretName 'password' is not descriptive enough",
			},
		},
	}

	badName := secretObj(map[string]interface{}{
		"spec": map[string]interface{}{"secretName": "password"},
	})
	if r := reconciler.RunValidation(badName, cfg, "secret"); r.Passed {
		t.Error("secretName 'password' should be rejected")
	}

	goodName := secretObj(map[string]interface{}{
		"spec": map[string]interface{}{"secretName": "db-admin-credentials"},
	})
	if r := reconciler.RunValidation(goodName, cfg, "secret"); !r.Passed {
		t.Errorf("descriptive secretName should pass: %v", r.ViolationSummary())
	}
}

func TestSecretValidation_MultipleRules_AllEvaluated(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.secretName", Operator: orktypes.ConditionExists, Message: "secretName required"},
			{Field: "spec.type", NotEquals: "Opaque", Message: "Opaque not allowed"},
			{Field: "spec.rotationDays", Min: "7", Message: "rotation >= 7 days"},
		},
	}

	// All three rules violated
	bad := secretObj(map[string]interface{}{
		"spec": map[string]interface{}{
			"type":         "Opaque",
			"rotationDays": "3",
		},
	})
	r := reconciler.RunValidation(bad, cfg, "secret")
	if r.Passed {
		t.Fatal("expected failures across all rules")
	}
	if len(r.Violations) < 2 {
		t.Errorf("expected at least 2 violations, got %d: %v", len(r.Violations), r.ViolationSummary())
	}
}

func TestSecretValidation_ErrorMethod_DescribesAllViolations(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.secretName", Operator: orktypes.ConditionExists, Message: "secretName required"},
			{Field: "spec.type", NotEquals: "Opaque", Message: "Opaque not allowed"},
		},
	}

	bad := secretObj(map[string]interface{}{
		"spec": map[string]interface{}{"type": "Opaque"},
	})
	r := reconciler.RunValidation(bad, cfg, "secret")
	if err := r.Error(); err == nil {
		t.Error("Error() should return non-nil when validation fails")
	}
}
