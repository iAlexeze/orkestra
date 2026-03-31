//go:build integration

// tests/integration/reconciler/deployment_test.go
// Integration tests for validation rules applied to deployment-like CRD objects.
// These tests exercise RunValidation end-to-end across the reconciler pipeline
// without requiring a live Kubernetes cluster.
package reconciler_test

import (
	"testing"

	"github.com/ialexeze/orkestra/pkg/reconciler"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// deployObj creates an unstructured object simulating a deployment-like CR.
func deployObj(fields map[string]interface{}) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
	obj.SetName("test-deploy")
	obj.SetNamespace("default")
	for k, v := range fields {
		// Support simple top-level and nested keys via spec prefix
		obj.Object[k] = v
	}
	return obj
}

func TestDeployValidation_ImageMustHaveOrgPrefix(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.image", Prefix: "myorg/", Message: "image must come from myorg registry"},
		},
	}

	// Compliant: image from allowed registry
	ok := deployObj(map[string]interface{}{
		"spec": map[string]interface{}{"image": "myorg/app:v1"},
	})
	if r := reconciler.RunValidation(ok, cfg, "deployment"); !r.Passed {
		t.Errorf("expected pass for org-prefixed image, got: %v", r.ViolationSummary())
	}

	// Non-compliant: image from unapproved registry
	bad := deployObj(map[string]interface{}{
		"spec": map[string]interface{}{"image": "docker.io/some/app:v1"},
	})
	if r := reconciler.RunValidation(bad, cfg, "deployment"); r.Passed {
		t.Error("expected failure for non-org image")
	}
}

func TestDeployValidation_ReplicasMinMax(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.replicas", Min: "1", Message: "replicas must be >= 1"},
			{Field: "spec.replicas", Max: "10", Message: "replicas must be <= 10"},
		},
	}

	// Compliant: replicas within range
	good := deployObj(map[string]interface{}{
		"spec": map[string]interface{}{"replicas": "3"},
	})
	if r := reconciler.RunValidation(good, cfg, "deployment"); !r.Passed {
		t.Errorf("expected pass for replicas=3, got: %v", r.ViolationSummary())
	}

	// Too few
	tooFew := deployObj(map[string]interface{}{
		"spec": map[string]interface{}{"replicas": "0"},
	})
	if r := reconciler.RunValidation(tooFew, cfg, "deployment"); r.Passed {
		t.Error("expected failure for replicas=0")
	}

	// Too many
	tooMany := deployObj(map[string]interface{}{
		"spec": map[string]interface{}{"replicas": "11"},
	})
	if r := reconciler.RunValidation(tooMany, cfg, "deployment"); r.Passed {
		t.Error("expected failure for replicas=11")
	}
}

func TestDeployValidation_MultipleViolations_AllCollected(t *testing.T) {
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.image", Prefix: "myorg/", Message: "image prefix required"},
			{Field: "spec.replicas", Min: "1", Message: "replicas >= 1 required"},
		},
	}

	// Both fields violate their rules
	bad := deployObj(map[string]interface{}{
		"spec": map[string]interface{}{
			"image":    "docker.io/bad:latest",
			"replicas": "0",
		},
	})
	r := reconciler.RunValidation(bad, cfg, "deployment")
	if r.Passed {
		t.Fatal("expected validation failure")
	}
	if len(r.Violations) != 2 {
		t.Errorf("expected 2 violations, got %d: %v", len(r.Violations), r.ViolationSummary())
	}
}

func TestDeployValidation_ImageSuffix_RequiredTag(t *testing.T) {
	// Suffix: ":latest" means "field must end with :latest"
	// Use this pattern to require images to carry an explicit tag suffix.
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{Field: "spec.image", Suffix: ":latest", Message: "only latest builds allowed in this environment"},
		},
	}

	// Compliant: ends with :latest
	latestTag := deployObj(map[string]interface{}{
		"spec": map[string]interface{}{"image": "myorg/app:latest"},
	})
	r := reconciler.RunValidation(latestTag, cfg, "deployment")
	if !r.Passed {
		t.Errorf("image ending with :latest should pass: %v", r.ViolationSummary())
	}

	// Non-compliant: does not end with :latest
	pinnedTag := deployObj(map[string]interface{}{
		"spec": map[string]interface{}{"image": "myorg/app:v1.2.3"},
	})
	r = reconciler.RunValidation(pinnedTag, cfg, "deployment")
	if r.Passed {
		t.Error("expected failure: image without :latest suffix should be blocked")
	}
}

func TestDeployValidation_NilConfig_AlwaysPasses(t *testing.T) {
	obj := deployObj(map[string]interface{}{})
	r := reconciler.RunValidation(obj, nil, "deployment")
	if !r.Passed {
		t.Error("nil ValidationConfig must always pass")
	}
}
