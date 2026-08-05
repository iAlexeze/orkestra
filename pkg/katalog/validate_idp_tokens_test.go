package katalog

import (
	"strings"
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

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
				Enabled: true,
				Auth: orktypes.ApplyAPIAuth{
					Tokens: applyTokens,
				},
			},
		},
	}
}

// Helper to create token permissions with global list
func perms(ops ...string) orktypes.IDPTokenPermissions {
	return orktypes.IDPTokenPermissions{
		Permissions: orktypes.IDPPermissionSet{
			Global: ops,
		},
	}
}

// Helper to create token permissions with schema and resources lists
func permsWithScopes(schemaOps, resourceOps []string) orktypes.IDPTokenPermissions {
	return orktypes.IDPTokenPermissions{
		Permissions: orktypes.IDPPermissionSet{
			Schema:    schemaOps,
			Resources: resourceOps,
		},
	}
}

// Helper to create token permissions with all three lists
func permsWithAll(globalOps, schemaOps, resourceOps []string) orktypes.IDPTokenPermissions {
	return orktypes.IDPTokenPermissions{
		Permissions: orktypes.IDPPermissionSet{
			Global:    globalOps,
			Schema:    schemaOps,
			Resources: resourceOps,
		},
	}
}

// Helper to create token permissions with namespaces
func permsWithNamespaces(ops []string, namespaces ...string) orktypes.IDPTokenPermissions {
	return orktypes.IDPTokenPermissions{
		Permissions: orktypes.IDPPermissionSet{
			Global: ops,
		},
		Namespaces: namespaces,
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

func TestValidateIDPTokenRestrictions_TokenExists_Global(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline", "control-center")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": perms("get", "list"),
					},
				},
			},
		},
	}
	if err := k.validateIDPTokenRestrictions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_TokenExists_Scoped(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline", "control-center")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": permsWithScopes(
							[]string{"get"},              // schema
							[]string{"create", "update"}, // resources
						),
					},
				},
			},
		},
	}
	if err := k.validateIDPTokenRestrictions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_TokenExists_GlobalWithScopes(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline", "control-center")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": permsWithAll(
							[]string{"get", "list", "create", "update"}, // global
							[]string{"get"},              // schema (subset)
							[]string{"create", "update"}, // resources (subset)
						),
					},
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
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"unknown-token": perms("get"),
					},
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

func TestValidateIDPTokenRestrictions_InvalidOperation_Global(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": perms("get", "invalid-op"),
					},
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

func TestValidateIDPTokenRestrictions_InvalidOperation_Schema(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": permsWithScopes(
							[]string{"invalid-op"}, // schema
							[]string{"create"},     // resources
						),
					},
				},
			},
		},
	}
	err := k.validateIDPTokenRestrictions()
	if err == nil {
		t.Fatal("expected error for invalid schema operation")
	}
	if !strings.Contains(err.Error(), "invalid-op") {
		t.Errorf("error should mention the invalid operation, got: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_InvalidOperation_Resources(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": permsWithScopes(
							[]string{"get"},        // schema
							[]string{"invalid-op"}, // resources
						),
					},
				},
			},
		},
	}
	err := k.validateIDPTokenRestrictions()
	if err == nil {
		t.Fatal("expected error for invalid resources operation")
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
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": perms("get", "list", "get"),
					},
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

func TestValidateIDPTokenRestrictions_SchemaNotSubsetOfGlobal(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": permsWithAll(
							[]string{"get", "list"}, // global
							[]string{"delete"},      // schema (not in global)
							[]string{"get"},         // resources
						),
					},
				},
			},
		},
	}
	err := k.validateIDPTokenRestrictions()
	if err == nil {
		t.Fatal("expected error when schema is not subset of global")
	}
	if !strings.Contains(err.Error(), "delete") {
		t.Errorf("error should mention the missing operation, got: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_ResourcesNotSubsetOfGlobal(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": permsWithAll(
							[]string{"get", "list"}, // global
							[]string{"get"},         // schema
							[]string{"delete"},      // resources (not in global)
						),
					},
				},
			},
		},
	}
	err := k.validateIDPTokenRestrictions()
	if err == nil {
		t.Fatal("expected error when resources is not subset of global")
	}
	if !strings.Contains(err.Error(), "delete") {
		t.Errorf("error should mention the missing operation, got: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_ClassWildcardWithoutGlobalWildcard(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": permsWithAll(
							[]string{"get", "list"}, // global
							[]string{"*"},           // schema wildcard (broader than global)
							[]string{"get"},         // resources
						),
					},
				},
			},
		},
	}
	err := k.validateIDPTokenRestrictions()
	if err == nil {
		t.Fatal("expected error when class wildcard exceeds global")
	}
	if !strings.Contains(err.Error(), "*") {
		t.Errorf("error should mention wildcard, got: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_GlobalWildcard(t *testing.T) {
	k := katalogWithGatewayTokens("admin")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"admin": perms("*"),
					},
				},
			},
		},
	}
	if err := k.validateIDPTokenRestrictions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_GlobalWildcardWithScopes(t *testing.T) {
	// When global has "*", class-specific scopes should be ignored.
	// The validation should not error because global overrides everything.
	k := katalogWithGatewayTokens("admin")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"admin": permsWithAll(
							[]string{"*"},      // global wildcard
							[]string{"get"},    // schema (ignored)
							[]string{"create"}, // resources (ignored)
						),
					},
				},
			},
		},
	}
	if err := k.validateIDPTokenRestrictions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIDPTokenRestrictions_NamespaceAllowed(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": permsWithNamespaces(
							[]string{"get"},
							"staging",
						),
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
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": permsWithNamespaces(
							[]string{"get"},
							"unauthorized",
						),
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
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": permsWithNamespaces(
							[]string{"get"},
							"restricted-ns",
						),
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
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline":    perms("get", "list"),
						"control-center": perms("*"),
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

func TestValidateIDPTokenRestrictions_WildcardOperation(t *testing.T) {
	k := katalogWithGatewayTokens("admin")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"admin": perms("*"),
					},
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
					AllowedTokens: orktypes.IDPAllowedTokens{
						Tokens: map[string]orktypes.IDPTokenPermissions{
							"ci-pipeline": perms("get"),
						},
					},
				},
				Warnings: orktypes.Warnings{},
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
					AllowedTokens: orktypes.IDPAllowedTokens{
						Tokens: map[string]orktypes.IDPTokenPermissions{
							"ci-pipeline": perms("get"),
						},
					},
				},
				Warnings: orktypes.Warnings{},
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
				Enabled: false,
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

func TestValidateIDPTokenRestrictions_EmptyPermissionsWarning(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": {
							Permissions: orktypes.IDPPermissionSet{},
						},
					},
				},
			},
			Warnings: orktypes.Warnings{},
		},
	}
	if err := k.validateIDPTokenRestrictions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	crd := k.enabledCRDs["myresource"]
	if !crd.Warnings.HasWarnings() {
		t.Fatal("expected warning for empty permissions")
	}
}

func TestValidateIDPTokenRestrictions_ClusterScopedCRDWithNamespaces(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	falsePtr := false
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"clusterresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						"ci-pipeline": permsWithNamespaces(
							[]string{"get"},
							"staging",
						),
					},
				},
			},
			Namespaced: &falsePtr,
			Warnings:   orktypes.Warnings{},
		},
	}
	if err := k.validateIDPTokenRestrictions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	crd := k.enabledCRDs["clusterresource"]
	if !crd.Warnings.HasWarnings() {
		t.Fatal("expected warning for namespace restrictions on cluster-scoped CRD")
	}
}

func TestValidateIDPTokenRestrictions_SchemaInheritsInvalidGlobalOpWarning(t *testing.T) {
	k := katalogWithGatewayTokens("ci-pipeline")
	k.enabledCRDs = map[string]orktypes.CRDEntry{
		"myresource": {
			IDP: &orktypes.IDPConfig{
				AllowedTokens: orktypes.IDPAllowedTokens{
					Tokens: map[string]orktypes.IDPTokenPermissions{
						// No explicit schema list — schema inherits from global,
						// but "create" isn't valid for schema endpoints.
						"ci-pipeline": perms("get", "create"),
					},
				},
			},
			Warnings: orktypes.Warnings{},
		},
	}
	if err := k.validateIDPTokenRestrictions(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	crd := k.enabledCRDs["myresource"]
	if !crd.Warnings.HasWarnings() {
		t.Fatal("expected warning for global op not valid on schema endpoints")
	}
}
