package katalog

import (
	"strings"
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// Helper to create a Katalog with gateway tokens
// Helper to create a Katalog with gateway tokens
func katalogWithGatewayTokens(tokens ...string) *Katalog {
	applyTokens := make([]orktypes.ApplyAPIToken, len(tokens))
	for i, name := range tokens {
		applyTokens[i] = orktypes.ApplyAPIToken{Name: name}
	}

	return &Katalog{
		Gateway: &orktypes.GatewayConfig{
			Enabled: true,
			ApplyAPI: &orktypes.ApplyAPIConfig{
				Enabled: true, // This is required for HasApplyAPI()
				Auth: orktypes.ApplyAPIAuth{
					Tokens: applyTokens,
				},
			},
		},
	}
}

func TestValidateIDPTokenRestrictions_NilIDP(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"myresource": {IDP: nil},
		},
	}
	if err := k.validateIDPTokenRestrictions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_NoTokenRestrictions(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"myresource": {IDP: &orktypes.IDPConfig{Enabled: true}},
		},
	}
	if err := k.validateIDPTokenRestrictions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_TokenExists(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline", "control-center")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: map[string]orktypes.IDPTokenPermissions{
					"ci-pipeline": {Permissions: []string{"get", "list"}},
				},
			},
		},
	}
	if err := k.validateIDPTokenRestrictions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_TokenNotFound(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: map[string]orktypes.IDPTokenPermissions{
					"unknown-token": {Permissions: []string{"get"}},
				},
			},
		},
	}
	err := k.validateIDPTokenRestrictions()
	if err == nil {
		t.Fatal("expected error for unknown token")
	}
	if !strings.Contains(err.Error(), "unknown-token") {
		t.Errorf("error should mention the unknown token, got: %v", err)
	}
	if !strings.Contains(err.Error(), "ci-pipeline") {
		t.Errorf("error should list available tokens, got: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_InvalidOperation(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: map[string]orktypes.IDPTokenPermissions{
					"ci-pipeline": {Permissions: []string{"get", "invalid-op"}},
				},
			},
		},
	}
	err := k.validateIDPTokenRestrictions()
	if err == nil {
		t.Fatal("expected error for invalid operation")
	}
	if !strings.Contains(err.Error(), "invalid-op") {
		t.Errorf("error should mention the invalid operation, got: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_DuplicateOperations(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: map[string]orktypes.IDPTokenPermissions{
					"ci-pipeline": {Permissions: []string{"get", "list", "get"}},
				},
			},
		},
	}
	err := k.validateIDPTokenRestrictions()
	if err == nil {
		t.Fatal("expected error for duplicate operations")
	}
	if !strings.Contains(err.Error(), "get") {
		t.Errorf("error should mention the duplicate operation, got: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_NamespaceAllowed(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: map[string]orktypes.IDPTokenPermissions{
					"ci-pipeline": {
						Permissions: []string{"get"},
						Namespaces:  []string{"staging"},
					},
				},
			},
			AllowedNamespaces: []string{"staging", "production"},
		},
	}
	if err := k.validateIDPTokenRestrictions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_NamespaceNotAllowed(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: map[string]orktypes.IDPTokenPermissions{
					"ci-pipeline": {
						Permissions: []string{"get"},
						Namespaces:  []string{"unauthorized"},
					},
				},
			},
			AllowedNamespaces: []string{"staging", "production"},
		},
	}
	err := k.validateIDPTokenRestrictions()
	if err == nil {
		t.Fatal("expected error for unauthorized namespace")
	}
	if !strings.Contains(err.Error(), "unauthorized") {
		t.Errorf("error should mention the unauthorized namespace, got: %v", err)
	}
	if !strings.Contains(err.Error(), "staging") {
		t.Errorf("error should list allowed namespaces, got: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_RestrictedNamespace(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: map[string]orktypes.IDPTokenPermissions{
					"ci-pipeline": {
						Permissions: []string{"get"},
						Namespaces:  []string{"restricted-ns"},
					},
				},
			},
			RestrictedNamespaces: []string{"restricted-ns", "blocked"},
		},
	}
	err := k.validateIDPTokenRestrictions()
	if err == nil {
		t.Fatal("expected error for restricted namespace")
	}
	if !strings.Contains(err.Error(), "restricted-ns") {
		t.Errorf("error should mention the restricted namespace, got: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_MultipleTokens(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline", "control-center")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: map[string]orktypes.IDPTokenPermissions{
					"ci-pipeline":    {Permissions: []string{"get", "list"}},
					"control-center": {Permissions: []string{"*"}},
				},
			},
			AllowedNamespaces: []string{"staging", "production"},
		},
	}
	if err := k.validateIDPTokenRestrictions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_WildcardOperation(t *testing.T) {
	k := katalogWithGatewayTokens("admin")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: map[string]orktypes.IDPTokenPermissions{
					"admin": {Permissions: []string{"*"}},
				},
			},
		},
	}
	if err := k.validateIDPTokenRestrictions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_GatewayNotConfigured(t *testing.T) {
	k := &Katalog{
		Gateway: nil,
		enabledCRDs: map[string]orktypes.CRDEntry{
			"myresource": {
				IDP: &orktypes.IDPConfig{
					AllowedTokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": {Permissions: []string{"get"}},
					},
				},
			},
		},
	}
	err := k.validateIDPTokenRestrictions()
	if err == nil {
		t.Fatal("expected error when gateway is not configured but IDP tokens are defined")
	}
}

func TestValidateIDPTokenRestrictions_GatewayAuthEmpty(t *testing.T) {
	k := &Katalog{
		Gateway: &orktypes.GatewayConfig{
			ApplyAPI: &orktypes.ApplyAPIConfig{
				Auth: orktypes.ApplyAPIAuth{
					Tokens: []orktypes.ApplyAPIToken{},
				},
			},
		},
		enabledCRDs: map[string]orktypes.CRDEntry{
			"myresource": {
				IDP: &orktypes.IDPConfig{
					AllowedTokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": {Permissions: []string{"get"}},
					},
				},
			},
		},
	}
	err := k.validateIDPTokenRestrictions()
	if err == nil {
		t.Fatal("expected error when gateway auth tokens are empty but IDP tokens are defined")
	}
}

func TestGatewayTokenNames(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline", "control-center")
	names := k.gatewayTokenNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 tokens, got %d: %v", len(names), names)
	}
	if names[0] != "ci-pipeline" || names[1] != "control-center" {
		t.Errorf("unexpected token names: %v", names)
	}
}

func TestGatewayTokenNames_WithTokens(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline", "control-center")
	names := k.gatewayTokenNames()

	if len(names) != 2 {
		t.Fatalf("expected 2 tokens, got %d: %v", len(names), names)
	}
	expected := []string{"ci-pipeline", "control-center"}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("expected %q, got %q", name, names[i])
		}
	}
}

func TestGatewayTokenNames_NoTokens(t *testing.T) {
	k := katalogWithGatewayTokens() // no tokens
	names := k.gatewayTokenNames()
	if names != nil {
		t.Errorf("expected nil, got %v", names)
	}
}

func TestGatewayTokenNames_GatewayDisabled(t *testing.T) {
	// Gateway exists but ApplyAPI is disabled
	k := &Katalog{
		Gateway: &orktypes.GatewayConfig{
			ApplyAPI: &orktypes.ApplyAPIConfig{
				Enabled: false, // ApplyAPI disabled
				Auth: orktypes.ApplyAPIAuth{
					Tokens: []orktypes.ApplyAPIToken{
						{Name: "ci-pipeline"},
					},
				},
			},
		},
	}
	names := k.gatewayTokenNames()
	if names != nil {
		t.Errorf("expected nil when ApplyAPI disabled, got %v", names)
	}
}
