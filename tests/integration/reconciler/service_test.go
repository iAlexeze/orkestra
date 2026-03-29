//go:build integration

// tests/integration/reconciler/service_test.go
// Integration tests for validation rules applied to service-like CRD objects.
package reconciler_test

import (
	"testing"

	"github.com/ialexeze/orkestra/pkg/reconciler"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func svcObj(fields map[string]interface{}) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
	obj.SetName("test-svc")
	obj.SetNamespace("default")
	for k, v := range fields {
		obj.Object[k] = v
	}
	return obj
}

func TestServiceValidation_TypeMustBeClusterIP(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.type", Equals: "ClusterIP", Message: "only ClusterIP services are allowed"},
		},
	}

	clusterIP := svcObj(map[string]interface{}{
		"spec": map[string]interface{}{"type": "ClusterIP"},
	})
	if r := reconciler.RunValidation(clusterIP, cfg, "service"); !r.Passed {
		t.Errorf("ClusterIP should pass: %v", r.ViolationSummary())
	}

	nodePort := svcObj(map[string]interface{}{
		"spec": map[string]interface{}{"type": "NodePort"},
	})
	if r := reconciler.RunValidation(nodePort, cfg, "service"); r.Passed {
		t.Error("NodePort should be rejected")
	}

	loadBalancer := svcObj(map[string]interface{}{
		"spec": map[string]interface{}{"type": "LoadBalancer"},
	})
	if r := reconciler.RunValidation(loadBalancer, cfg, "service"); r.Passed {
		t.Error("LoadBalancer should be rejected")
	}
}

func TestServiceValidation_PortRange(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.port", Min: "1024", Message: "port must be >= 1024 (reserved ports not allowed)"},
			{Field: "spec.port", Max: "65535", Message: "port must be <= 65535"},
		},
	}

	validPort := svcObj(map[string]interface{}{
		"spec": map[string]interface{}{"port": "8080"},
	})
	if r := reconciler.RunValidation(validPort, cfg, "service"); !r.Passed {
		t.Errorf("port 8080 should pass: %v", r.ViolationSummary())
	}

	reservedPort := svcObj(map[string]interface{}{
		"spec": map[string]interface{}{"port": "80"},
	})
	if r := reconciler.RunValidation(reservedPort, cfg, "service"); r.Passed {
		t.Error("port 80 should be rejected (reserved)")
	}
}

func TestServiceValidation_NameMustContainEnvironment(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:    "spec.name",
				Operator: orktypes.ConditionContains,
				Value:    "prod",
				Message:  "service name must contain environment tag",
			},
		},
	}

	withEnv := svcObj(map[string]interface{}{
		"spec": map[string]interface{}{"name": "api-prod-v1"},
	})
	if r := reconciler.RunValidation(withEnv, cfg, "service"); !r.Passed {
		t.Errorf("name containing 'prod' should pass: %v", r.ViolationSummary())
	}

	noEnv := svcObj(map[string]interface{}{
		"spec": map[string]interface{}{"name": "api-v1"},
	})
	if r := reconciler.RunValidation(noEnv, cfg, "service"); r.Passed {
		t.Error("name without environment tag should be rejected")
	}
}

func TestServiceValidation_ExistenceRule_TargetPortRequired(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:    "spec.targetPort",
				Operator: orktypes.ConditionExists,
				Message:  "targetPort is required",
			},
		},
	}

	withTargetPort := svcObj(map[string]interface{}{
		"spec": map[string]interface{}{"targetPort": "8080"},
	})
	if r := reconciler.RunValidation(withTargetPort, cfg, "service"); !r.Passed {
		t.Errorf("spec.targetPort present should pass: %v", r.ViolationSummary())
	}

	withoutTargetPort := svcObj(map[string]interface{}{
		"spec": map[string]interface{}{},
	})
	if r := reconciler.RunValidation(withoutTargetPort, cfg, "service"); r.Passed {
		t.Error("missing spec.targetPort should fail")
	}
}

func TestServiceValidation_ViolationSummary_IsHumanReadable(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.type", Equals: "ClusterIP", Message: "only ClusterIP allowed"},
			{Field: "spec.targetPort", Operator: orktypes.ConditionExists, Message: "targetPort required"},
		},
	}

	bad := svcObj(map[string]interface{}{
		"spec": map[string]interface{}{"type": "NodePort"},
	})
	r := reconciler.RunValidation(bad, cfg, "service")
	if r.Passed {
		t.Fatal("expected failures")
	}
	summary := r.ViolationSummary()
	if summary == "" {
		t.Error("ViolationSummary should not be empty when violations exist")
	}
}
