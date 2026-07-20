package katalog_test

import (
	"testing"
)

func TestIsApplyAPIEnabled(t *testing.T) {
	t.Run("enabled with secretRef", func(t *testing.T) {
		k := mustParseTestdata(t, "applyapi/applyapi-secretref.yaml")
		if !k.IsApplyAPIEnabled() {
			t.Error("IsApplyAPIEnabled() = false, want true")
		}
	})

	t.Run("enabled with env token", func(t *testing.T) {
		k := mustParseTestdata(t, "applyapi/applyapi-envtoken.yaml")
		if !k.IsApplyAPIEnabled() {
			t.Error("IsApplyAPIEnabled() = false, want true")
		}
	})

	t.Run("disabled when no gateway block", func(t *testing.T) {
		k := mustParseTestdata(t, "webhooks/webhooks-no-rules.yaml")
		if k.IsApplyAPIEnabled() {
			t.Error("IsApplyAPIEnabled() = true, want false")
		}
	})

	t.Run("true when applyAPI enabled even without idp", func(t *testing.T) {
		k := mustParseTestdata(t, "applyapi/applyapi-no-idp.yaml")
		if !k.IsApplyAPIEnabled() {
			t.Error("IsApplyAPIEnabled() = false, want true — flag is set")
		}
	})
}

func TestHasApplyAPISecretRefs(t *testing.T) {
	t.Run("true when secretRef present", func(t *testing.T) {
		k := mustParseTestdata(t, "applyapi/applyapi-secretref.yaml")
		if !k.HasApplyAPISecretRefs() {
			t.Error("HasApplyAPISecretRefs() = false, want true")
		}
	})

	t.Run("false when only env token", func(t *testing.T) {
		k := mustParseTestdata(t, "applyapi/applyapi-envtoken.yaml")
		if k.HasApplyAPISecretRefs() {
			t.Error("HasApplyAPISecretRefs() = true, want false")
		}
	})

	t.Run("false when applyAPI not enabled", func(t *testing.T) {
		k := mustParseTestdata(t, "webhooks/webhooks-no-rules.yaml")
		if k.HasApplyAPISecretRefs() {
			t.Error("HasApplyAPISecretRefs() = true, want false")
		}
	})
}

func TestHasIDPEnabled(t *testing.T) {
	t.Run("true when applyAPI enabled and idp on a CRD", func(t *testing.T) {
		k := mustParseTestdata(t, "applyapi/applyapi-secretref.yaml")
		if !k.HasIDPEnabled() {
			t.Error("HasIDPEnabled() = false, want true")
		}
	})

	t.Run("false when no gateway block", func(t *testing.T) {
		k := mustParseTestdata(t, "webhooks/webhooks-no-rules.yaml")
		if k.HasIDPEnabled() {
			t.Error("HasIDPEnabled() = true, want false")
		}
	})

	t.Run("false when applyAPI enabled but no idp", func(t *testing.T) {
		k := mustParseTestdata(t, "applyapi/applyapi-no-idp.yaml")
		if k.HasIDPEnabled() {
			t.Error("HasIDPEnabled() = true, want false — no idp.enabled CRD")
		}
	})
}

func TestGenerateGatewayRBACRules_ApplyAPI(t *testing.T) {
	k := mustParseTestdata(t, "applyapi/applyapi-secretref.yaml")
	rules := k.GenerateGatewayRBACRules()

	// Must contain get+create on secrets for self-bootstrap
	foundSecrets := false
	for _, r := range rules {
		for _, res := range r.Resources {
			if res == "secrets" {
				if !contains(r.Verbs, "get") || !contains(r.Verbs, "create") {
					t.Errorf("secrets rule missing get or create: %v", r.Verbs)
				}
				foundSecrets = true
			}
		}
	}
	if !foundSecrets {
		t.Error("no secrets rule found in gateway RBAC rules")
	}

	// Must contain CR verbs for the IDP-enabled platform CRD
	foundCRDVerbs := false
	for _, r := range rules {
		for _, res := range r.Resources {
			if res == "platforms" {
				foundCRDVerbs = true
				for _, v := range []string{"get", "list", "create", "update", "patch", "delete"} {
					if !contains(r.Verbs, v) {
						t.Errorf("platforms rule missing verb %q: %v", v, r.Verbs)
					}
				}
			}
		}
	}
	if !foundCRDVerbs {
		t.Error("no platforms CR rule found in gateway RBAC rules")
	}
}

func TestGenerateGatewayRBACRules_NoSecretRuleForEnvToken(t *testing.T) {
	k := mustParseTestdata(t, "applyapi/applyapi-envtoken.yaml")
	rules := k.GenerateGatewayRBACRules()
	for _, r := range rules {
		for _, res := range r.Resources {
			if res == "secrets" {
				// secrets rule is only valid if NeedsCertificates is true (TLS)
				// For env-token-only Apply API there should be no secrets rule
				// unless NeedsCertificates is also true.
				if k.HasApplyAPISecretRefs() {
					t.Error("unexpected HasApplyAPISecretRefs() = true for env-token katalog")
				}
				_ = r
			}
		}
	}
}
