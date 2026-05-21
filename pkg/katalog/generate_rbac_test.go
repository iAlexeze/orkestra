package katalog_test

import (
	"testing"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/types"
)

// helpers to build rule objects
func buildValidationRule() *types.ValidationConfig {
	return &types.ValidationConfig{
		Rules: []types.ValidationRule{
			{Field: "rule1", Message: "rule1 in validation"},
		},
	}
}

func buildMutationRule() *types.MutationConfig {
	return &types.MutationConfig{
		Rules: []types.MutationRule{
			{Field: "env", Default: "development"},
		},
	}
}

func enableAdmission(k *katalog.Katalog) *katalog.Katalog {
	enabled := true
	k.Security = types.KatalogSecurity{
		Webhooks: &types.WebhooksConfig{
			Admission: &types.AdmissionWebhookToggle{Enabled: &enabled},
		},
	}

	return k
}

// buildBaseKatalog returns a Katalog with a single CRD entry populated.
// Tests should clone the returned katalog before mutating it.
func buildBaseKatalog() *katalog.Katalog {
	entry := types.CRDEntry{
		Name: "admission-tester",
		APITypes: types.APITypes{
			Group:   "platform.orkestra.io",
			Version: "v1alpha1",
			Kind:    "Platform",
		},
	}

	return &katalog.Katalog{
		Spec: types.KatalogSpec{
			CRDs: map[string]types.CRDEntry{
				entry.Name: entry,
			},
		},
	}
}

// cloneKatalog builds a new Katalog from src, copying the CRD and Security
// fields that tests mutate. Direct struct copy is avoided because Katalog
// embeds sync/atomic.Bool (noCopy guard).
func cloneKatalog(src *katalog.Katalog) *katalog.Katalog {
	if src == nil {
		return &katalog.Katalog{}
	}

	dst := &katalog.Katalog{
		Security: src.Security,
	}

	if src.Spec.CRDs != nil {
		dst.Spec.CRDs = make(map[string]types.CRDEntry, len(src.Spec.CRDs))
		for k, v := range src.Spec.CRDs {
			entryCopy := v
			dst.Spec.CRDs[k] = entryCopy
		}
	}

	return dst
}

// contains helper
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// Individual tests that each operate on their own katalog clone.
func TestWebhookResources_NoRules(t *testing.T) {
	base := buildBaseKatalog()
	k := cloneKatalog(base)

	if got := k.WebhookResources(); len(got) != 0 {
		t.Fatalf("expected no webhook resources, got %v", got)
	}
}

func TestWebhookResources_EnabledNoRules(t *testing.T) {
	base := buildBaseKatalog()
	k := cloneKatalog(base)

	k = enableAdmission(k)

	if got := k.WebhookResources(); len(got) != 0 {
		t.Fatalf("expected no webhook resources, got %v", got)
	}
}

func TestWebhookResources_ValidationOnly(t *testing.T) {
	base := buildBaseKatalog()
	k := cloneKatalog(base)

	k = enableAdmission(k)

	// set validation on the CRD entry (must update the map entry)
	entry := k.Spec.CRDs["admission-tester"]
	entry.Validation = buildValidationRule()
	k.Spec.CRDs["admission-tester"] = entry

	got := k.WebhookResources()
	if !contains(got, "validatingwebhookconfigurations") {
		t.Fatalf("expected validatingwebhookconfigurations when validation rules present, got %v", got)
	}
	if contains(got, "mutatingwebhookconfigurations") {
		t.Fatalf("did not expect mutatingwebhookconfigurations for validation-only: %v", got)
	}
}

func TestWebhookResources_MutationOnly(t *testing.T) {
	base := buildBaseKatalog()
	k := cloneKatalog(base)

	k = enableAdmission(k)

	// set mutation on the CRD entry
	entry := k.Spec.CRDs["admission-tester"]
	entry.Mutation = buildMutationRule()
	k.Spec.CRDs["admission-tester"] = entry

	got := k.WebhookResources()
	if !contains(got, "mutatingwebhookconfigurations") {
		t.Fatalf("expected mutatingwebhookconfigurations when mutation rules present, got %v", got)
	}
	// mutation-only may or may not require validating resource depending on your logic;
	// assert that validating is not present if your design expects that.
	if contains(got, "validatingwebhookconfigurations") {
		t.Fatalf("did not expect validatingwebhookconfigurations for mutation-only: %v", got)
	}
}

func TestWebhookResources_BothValidationAndMutation(t *testing.T) {
	base := buildBaseKatalog()
	k := cloneKatalog(base)

	k = enableAdmission(k)

	entry := k.Spec.CRDs["admission-tester"]
	entry.Validation = buildValidationRule()
	entry.Mutation = buildMutationRule()
	k.Spec.CRDs["admission-tester"] = entry

	got := k.WebhookResources()
	if !contains(got, "validatingwebhookconfigurations") || !contains(got, "mutatingwebhookconfigurations") {
		t.Fatalf("expected both resources when both rule types present: %v", got)
	}
}
