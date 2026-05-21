// pkg/katalog/enrichment_test.go
package katalog

import (
	"strings"
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func TestEnrichCRDEntry_KindOnly_Deployment(t *testing.T) {
	entry := &orktypes.CRDEntry{
		Name: "deployment-governance",
		APITypes: orktypes.APITypes{
			Kind: "Deployment", // only kind set
		},
	}

	outcome, err := EnrichCRDEntry(entry)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if outcome != orktypes.EnrichmentApplied {
		t.Errorf("expected EnrichmentApplied, got %v", outcome)
	}
	if entry.APITypes.Group != "apps" {
		t.Errorf("expected group=apps, got %q", entry.APITypes.Group)
	}
	if entry.APITypes.Version != "v1" {
		t.Errorf("expected version=v1, got %q", entry.APITypes.Version)
	}
	if entry.APITypes.Plural != "deployments" {
		t.Errorf("expected plural=deployments, got %q", entry.APITypes.Plural)
	}
	if !entry.IsNamespaced() {
		t.Error("expected Namespaced=true for Deployment")
	}
	if !entry.IsBuiltIn {
		t.Error("expected IsBuiltIn=true after enrichment")
	}
}

func TestEnrichCRDEntry_KindOnly_Pod(t *testing.T) {
	entry := &orktypes.CRDEntry{
		Name:     "pod-governance",
		APITypes: orktypes.APITypes{Kind: "Pod"},
	}

	outcome, err := EnrichCRDEntry(entry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome != orktypes.EnrichmentApplied {
		t.Errorf("expected EnrichmentApplied")
	}
	if entry.APITypes.Group != "" {
		t.Errorf("expected empty group for Pod, got %q", entry.APITypes.Group)
	}
	if entry.APITypes.Version != "v1" {
		t.Errorf("expected version=v1, got %q", entry.APITypes.Version)
	}
	if entry.APITypes.APIPath != "/api" {
		t.Errorf("expected APIPath=/api for core resource, got %q", entry.APITypes.APIPath)
	}
	if entry.BuiltInGroup != "core" {
		t.Errorf("expected BuiltInGroup=core, got %q", entry.BuiltInGroup)
	}
}

func TestEnrichCRDEntry_KindOnly_Namespace_ClusterScoped(t *testing.T) {
	entry := &orktypes.CRDEntry{
		Name:     "namespace-governance",
		APITypes: orktypes.APITypes{Kind: "Namespace"},
	}

	_, err := EnrichCRDEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.IsNamespaced() {
		t.Error("Namespace is cluster-scoped — Namespaced should be false")
	}
}

func TestEnrichCRDEntry_FullySpecified_NotNeeded(t *testing.T) {
	entry := &orktypes.CRDEntry{
		Name: "my-website",
		APITypes: orktypes.APITypes{
			Kind:    "Website",
			Group:   "demo.orkestra.io",
			Version: "v1alpha1",
			Plural:  "websites",
		},
	}

	outcome, err := EnrichCRDEntry(entry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome != orktypes.EnrichmentNotNeeded {
		t.Errorf("expected EnrichmentNotNeeded for fully specified entry")
	}
	if entry.APITypes.Group != "demo.orkestra.io" {
		t.Errorf("group should not have changed")
	}
}

func TestEnrichCRDEntry_UnknownKind_Error(t *testing.T) {
	entry := &orktypes.CRDEntry{
		Name:     "my-custom-crd",
		APITypes: orktypes.APITypes{Kind: "MyCustomResource"},
	}

	outcome, err := EnrichCRDEntry(entry)

	if err == nil {
		t.Fatal("expected error for unknown kind with missing apiTypes fields")
	}
	if outcome != orktypes.EnrichmentFailed {
		t.Errorf("expected EnrichmentFailed")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "MyCustomResource") {
		t.Errorf("error should mention the kind: %q", errStr)
	}
	if !strings.Contains(errStr, "group") {
		t.Errorf("error should mention group: %q", errStr)
	}
}

func TestEnrichCRDEntry_PartiallySpecified_Error(t *testing.T) {
	entry := &orktypes.CRDEntry{
		Name: "partial",
		APITypes: orktypes.APITypes{
			Kind:  "Website",
			Group: "demo.orkestra.io",
		},
	}

	outcome, err := EnrichCRDEntry(entry)

	if err == nil {
		t.Fatal("expected error for partially specified apiTypes")
	}
	if outcome != orktypes.EnrichmentFailed {
		t.Errorf("expected EnrichmentFailed")
	}
	if !strings.Contains(err.Error(), "partially specified") {
		t.Errorf("error should mention partial specification: %q", err.Error())
	}
}

func TestEnrichCRDEntry_CaseNormalization(t *testing.T) {
	entry := &orktypes.CRDEntry{
		Name:     "dep-gov",
		APITypes: orktypes.APITypes{Kind: "deployment"},
	}

	_, err := EnrichCRDEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.APITypes.Kind != "Deployment" {
		t.Errorf("expected Kind=Deployment after normalization, got %q", entry.APITypes.Kind)
	}
}
