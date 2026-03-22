// pkg/katalog/builtins_test.go
package katalog_test

import (
	"strings"
	"testing"

	"github.com/ialexeze/orkestra/pkg/katalog"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// ── LookupBuiltIn ─────────────────────────────────────────────────────────────

func TestLookupBuiltIn_CoreGroup(t *testing.T) {
	tests := []struct {
		kind           string
		expectedPlural string
		expectedGroup  string
		expectedPKG    string
		namespaced     bool
	}{
		{"Pod", "pods", "", "core", true},
		{"Service", "services", "", "core", true},
		{"ConfigMap", "configmaps", "", "core", true},
		{"Secret", "secrets", "", "core", true},
		{"Namespace", "namespaces", "", "core", false},
		{"ServiceAccount", "serviceaccounts", "", "core", true},
		{"PersistentVolumeClaim", "persistentvolumeclaims", "", "core", true},
		{"PersistentVolume", "persistentvolumes", "", "core", false},
		{"Node", "nodes", "", "core", false},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			result := katalog.LookupBuiltIn(tt.kind)
			if !result.Found {
				t.Fatalf("expected %q to be found in built-in registry", tt.kind)
			}
			if result.BuiltIn.Plural != tt.expectedPlural {
				t.Errorf("plural: expected %q, got %q", tt.expectedPlural, result.BuiltIn.Plural)
			}
			if result.BuiltIn.Group != tt.expectedGroup {
				t.Errorf("group: expected %q, got %q", tt.expectedGroup, result.BuiltIn.Group)
			}
			if result.DisplayGroup != tt.expectedPKG {
				t.Errorf("display group: expected %q, got %q", tt.expectedPKG, result.DisplayGroup)
			}
			if result.BuiltIn.Namespaced != tt.namespaced {
				t.Errorf("namespaced: expected %v, got %v", tt.namespaced, result.BuiltIn.Namespaced)
			}
		})
	}
}

func TestLookupBuiltIn_AppsGroup(t *testing.T) {
	tests := []struct {
		kind   string
		plural string
	}{
		{"Deployment", "deployments"},
		{"StatefulSet", "statefulsets"},
		{"DaemonSet", "daemonsets"},
		{"ReplicaSet", "replicasets"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			result := katalog.LookupBuiltIn(tt.kind)
			if !result.Found {
				t.Fatalf("expected %q to be found", tt.kind)
			}
			if result.BuiltIn.Group != "apps" {
				t.Errorf("expected group=apps, got %q", result.BuiltIn.Group)
			}
			if result.BuiltIn.Version != "v1" {
				t.Errorf("expected version=v1, got %q", result.BuiltIn.Version)
			}
			if result.BuiltIn.Plural != tt.plural {
				t.Errorf("expected plural=%q, got %q", tt.plural, result.BuiltIn.Plural)
			}
		})
	}
}

func TestLookupBuiltIn_CaseInsensitive(t *testing.T) {
	// All of these should resolve to the same Deployment entry
	variants := []string{"Deployment", "deployment", "DEPLOYMENT", "dEpLoyMeNt"}
	for _, v := range variants {
		t.Run(v, func(t *testing.T) {
			result := katalog.LookupBuiltIn(v)
			if !result.Found {
				t.Errorf("expected %q to resolve to Deployment", v)
			}
			if result.Kind != "Deployment" {
				t.Errorf("expected canonical Kind=Deployment, got %q", result.Kind)
			}
		})
	}
}

func TestLookupBuiltIn_NotFound(t *testing.T) {
	unknowns := []string{"Website", "MyApp", "CustomCRD", ""}
	for _, kind := range unknowns {
		t.Run(kind, func(t *testing.T) {
			result := katalog.LookupBuiltIn(kind)
			if result.Found {
				t.Errorf("expected %q to NOT be found in built-in registry", kind)
			}
		})
	}
}

func TestLookupBuiltIn_CanonicalKindName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"horizontalpodautoscaler", "HorizontalPodAutoscaler"},
		{"persistentvolumeclaim", "PersistentVolumeClaim"},
		{"clusterrolebinding", "ClusterRoleBinding"},
		{"customresourcedefinition", "CustomResourceDefinition"},
		{"poddisruptionbudget", "PodDisruptionBudget"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := katalog.LookupBuiltIn(tt.input)
			if !result.Found {
				t.Fatalf("expected %q to be found", tt.input)
			}
			if result.Kind != tt.expected {
				t.Errorf("canonical name: expected %q, got %q", tt.expected, result.Kind)
			}
		})
	}
}

func TestAllBuiltInKinds_Sorted(t *testing.T) {
	kinds := katalog.AllBuiltInKinds()

	if len(kinds) == 0 {
		t.Fatal("expected non-empty built-in kind list")
	}

	// Verify sorted
	for i := 1; i < len(kinds); i++ {
		if kinds[i] < kinds[i-1] {
			t.Errorf("kinds not sorted: %q comes before %q", kinds[i], kinds[i-1])
		}
	}
}

func TestAllBuiltInKinds_NoInternalAliases(t *testing.T) {
	kinds := katalog.AllBuiltInKinds()
	for _, k := range kinds {
		if strings.Contains(k, "_") {
			t.Errorf("internal alias leaked into AllBuiltInKinds: %q", k)
		}
	}
}

// ── EnrichCRDEntry ────────────────────────────────────────────────────────────

func TestEnrichCRDEntry_KindOnly_Deployment(t *testing.T) {
	entry := &orktypes.CRDEntry{
		Name: "deployment-governance",
		APITypes: orktypes.APITypes{
			Kind: "Deployment", // only kind set
		},
	}

	outcome, err := katalog.EnrichCRDEntry(entry)

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
	if !entry.Namespaced {
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

	outcome, err := katalog.EnrichCRDEntry(entry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome != orktypes.EnrichmentApplied {
		t.Errorf("expected EnrichmentApplied")
	}
	// Pod is core group — empty string
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

	_, err := katalog.EnrichCRDEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Namespaced {
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

	outcome, err := katalog.EnrichCRDEntry(entry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome != orktypes.EnrichmentNotNeeded {
		t.Errorf("expected EnrichmentNotNeeded for fully specified entry")
	}
	// Values should be unchanged
	if entry.APITypes.Group != "demo.orkestra.io" {
		t.Errorf("group should not have changed")
	}
}

func TestEnrichCRDEntry_UnknownKind_Error(t *testing.T) {
	entry := &orktypes.CRDEntry{
		Name:     "my-custom-crd",
		APITypes: orktypes.APITypes{Kind: "MyCustomResource"},
		// group, version, plural all empty — looks like kind-only but not a built-in
	}

	outcome, err := katalog.EnrichCRDEntry(entry)

	if err == nil {
		t.Fatal("expected error for unknown kind with missing apiTypes fields")
	}
	if outcome != orktypes.EnrichmentFailed {
		t.Errorf("expected EnrichmentFailed")
	}
	// Error should mention the kind and give guidance
	errStr := err.Error()
	if !strings.Contains(errStr, "MyCustomResource") {
		t.Errorf("error should mention the kind: %q", errStr)
	}
	if !strings.Contains(errStr, "group") {
		t.Errorf("error should mention group: %q", errStr)
	}
}

func TestEnrichCRDEntry_PartiallySpecified_Error(t *testing.T) {
	// group set but version and plural missing — not kind-only, not fully specified
	entry := &orktypes.CRDEntry{
		Name: "partial",
		APITypes: orktypes.APITypes{
			Kind:  "Website",
			Group: "demo.orkestra.io",
			// Version and Plural missing
		},
	}

	outcome, err := katalog.EnrichCRDEntry(entry)

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
	// Lowercase kind should be normalized to canonical form
	entry := &orktypes.CRDEntry{
		Name:     "dep-gov",
		APITypes: orktypes.APITypes{Kind: "deployment"},
	}

	_, err := katalog.EnrichCRDEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.APITypes.Kind != "Deployment" {
		t.Errorf("expected Kind=Deployment after normalization, got %q", entry.APITypes.Kind)
	}
}
