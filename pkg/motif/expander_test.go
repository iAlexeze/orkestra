package motif_test

import (
	"testing"

	"github.com/orkspace/orkestra/pkg/motif"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func loadTestMotif(t *testing.T, path string) *orktypes.Motif {
	t.Helper()
	m, err := motif.Load(path)
	if err != nil {
		t.Fatalf("loading motif %s: %v", path, err)
	}
	return m
}

func TestExpand_RequiredInputProvided(t *testing.T) {
	m := loadTestMotif(t, "testdata/simple.yaml")

	expanded, err := motif.Expand(m, map[string]string{
		"image": "nginx",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expanded.OnReconcile == nil {
		t.Fatal("expected OnReconcile to be non-nil")
	}
	if len(expanded.OnReconcile.Deployments) != 1 {
		t.Fatalf("deployments len = %d, want 1", len(expanded.OnReconcile.Deployments))
	}
}

func TestExpand_DefaultFilled(t *testing.T) {
	m := loadTestMotif(t, "testdata/simple.yaml")

	expanded, err := motif.Expand(m, map[string]string{
		"image": "nginx",
		// "tag" not provided — should use default "latest"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expanded.OnReconcile == nil {
		t.Fatal("expected OnReconcile to be non-nil")
	}
	if len(expanded.OnReconcile.Deployments) != 1 {
		t.Fatalf("deployments len = %d, want 1", len(expanded.OnReconcile.Deployments))
	}
}

func TestExpand_MissingRequiredInput(t *testing.T) {
	m := loadTestMotif(t, "testdata/simple.yaml")

	_, err := motif.Expand(m, map[string]string{
		// "image" is required but not provided
	})
	if err == nil {
		t.Fatal("expected error for missing required input, got nil")
	}
}

func TestExpand_UnknownInput(t *testing.T) {
	m := loadTestMotif(t, "testdata/simple.yaml")

	_, err := motif.Expand(m, map[string]string{
		"image":   "nginx",
		"unknown": "value",
	})
	if err == nil {
		t.Fatal("expected error for unknown input, got nil")
	}
}

func TestValidateMotifTemplates_AllDeclared(t *testing.T) {
	m := loadTestMotif(t, "testdata/simple.yaml")
	errs := motif.ValidateMotifTemplates(m)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestMergeHookTemplates_Appends(t *testing.T) {
	dst := &orktypes.HookTemplates{}
	src := &orktypes.HookTemplates{
		Deployments: []orktypes.DeploymentTemplateSource{{Name: "app"}},
		Services:    []orktypes.ServiceTemplateSource{{Name: "svc"}},
	}

	motif.MergeHookTemplates(dst, src)

	if len(dst.Deployments) != 1 {
		t.Errorf("deployments = %d, want 1", len(dst.Deployments))
	}
	if len(dst.Services) != 1 {
		t.Errorf("services = %d, want 1", len(dst.Services))
	}
}

func TestMergeHookTemplates_NilSafe(t *testing.T) {
	dst := &orktypes.HookTemplates{}
	// Should not panic
	motif.MergeHookTemplates(dst, nil)
	motif.MergeHookTemplates(nil, dst)
}
