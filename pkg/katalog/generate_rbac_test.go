package katalog_test

import (
	"testing"

	"github.com/orkspace/orkestra/pkg/katalog"
)

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func mustParseTestdata(t *testing.T, name string) *katalog.Katalog {
	t.Helper()
	k, err := katalog.ParseFile("testdata/" + name)
	if err != nil {
		t.Fatalf("ParseFile testdata/%s: %v", name, err)
	}
	return k
}

func TestWebhookResources_NoRules(t *testing.T) {
	k := mustParseTestdata(t, "webhooks/webhooks-no-rules.yaml")
	if got := k.WebhookResources(); len(got) != 0 {
		t.Fatalf("expected no webhook resources, got %v", got)
	}
}

func TestWebhookResources_EnabledNoRules(t *testing.T) {
	k := mustParseTestdata(t, "webhooks/webhooks-no-rules.yaml")
	if got := k.WebhookResources(); len(got) != 0 {
		t.Fatalf("expected no webhook resources with admission enabled but no rules, got %v", got)
	}
}

func TestWebhookResources_ValidationOnly(t *testing.T) {
	k := mustParseTestdata(t, "webhooks/webhooks-validation.yaml")
	got := k.WebhookResources()
	if !contains(got, "validatingwebhookconfigurations") {
		t.Fatalf("expected validatingwebhookconfigurations, got %v", got)
	}
	if contains(got, "mutatingwebhookconfigurations") {
		t.Fatalf("did not expect mutatingwebhookconfigurations for validation-only: %v", got)
	}
}

func TestWebhookResources_MutationOnly(t *testing.T) {
	k := mustParseTestdata(t, "webhooks/webhooks-mutation.yaml")
	got := k.WebhookResources()
	if !contains(got, "mutatingwebhookconfigurations") {
		t.Fatalf("expected mutatingwebhookconfigurations, got %v", got)
	}
	if contains(got, "validatingwebhookconfigurations") {
		t.Fatalf("did not expect validatingwebhookconfigurations for mutation-only: %v", got)
	}
}

func TestWebhookResources_BothValidationAndMutation(t *testing.T) {
	k := mustParseTestdata(t, "webhooks/webhooks-both.yaml")
	got := k.WebhookResources()
	if !contains(got, "validatingwebhookconfigurations") || !contains(got, "mutatingwebhookconfigurations") {
		t.Fatalf("expected both webhook resources, got %v", got)
	}
}

func TestGenerateRBACRules_CustomResources(t *testing.T) {
	k := mustParseTestdata(t, "rbac/custom-rbac.yaml")
	rules := k.GenerateRBACRules()

	want := map[string]string{
		"issuers":      "cert-manager.io",
		"certificates": "cert-manager.io",
	}
	for resource, group := range want {
		found := false
		for _, r := range rules {
			if contains(r.APIGroups, group) && contains(r.Resources, resource) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected RBAC rule for %s/%s — not found in generated rules", group, resource)
		}
	}
}

func TestGenerateRBACRules_CustomResources_Dedup(t *testing.T) {
	k := mustParseTestdata(t, "rbac/custom-rbac-dedup.yaml")
	rules := k.GenerateRBACRules()

	count := 0
	for _, r := range rules {
		if contains(r.APIGroups, "cert-manager.io") && contains(r.Resources, "certificates") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 RBAC rule for cert-manager.io/certificates, got %d", count)
	}
}
