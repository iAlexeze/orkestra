package types_test

import (
	"testing"

	"github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestTokenAllowed(t *testing.T) {
	// Helper to build an IDPConfig with allowedTokens declared.
	cfg := func(tokens map[string]types.IDPTokenPermissions) *types.IDPConfig {
		return &types.IDPConfig{AllowedTokens: tokens}
	}

	tests := []struct {
		name       string
		idp        *types.IDPConfig
		token      string
		op         string
		namespace  string
		wantOK     bool
		wantReason types.IDPDenyReason
	}{
		{
			name:      "no restrictions — any token allowed",
			idp:       cfg(nil),
			token:     "any-token",
			op:        types.IDPOpDelete,
			namespace: "any-ns",
			wantOK:    true,
		},
		{
			name: "wildcard permission grants all ops",
			idp: cfg(map[string]types.IDPTokenPermissions{
				"control-center": {Permissions: []string{"*"}},
			}),
			token:     "control-center",
			op:        types.IDPOpDelete,
			namespace: "production",
			wantOK:    true,
		},
		{
			name: "specific permission — allowed",
			idp: cfg(map[string]types.IDPTokenPermissions{
				"ci-pipeline": {Permissions: []string{"create", "update"}},
			}),
			token:     "ci-pipeline",
			op:        types.IDPOpCreate,
			namespace: "staging",
			wantOK:    true,
		},
		{
			name: "specific permission — denied op",
			idp: cfg(map[string]types.IDPTokenPermissions{
				"ci-pipeline": {Permissions: []string{"create", "update"}},
			}),
			token:      "ci-pipeline",
			op:         types.IDPOpDelete,
			namespace:  "staging",
			wantOK:     false,
			wantReason: types.IDPDenyReasonOperation,
		},
		{
			name: "unknown token — denied",
			idp: cfg(map[string]types.IDPTokenPermissions{
				"ci-pipeline": {Permissions: []string{"*"}},
			}),
			token:      "rogue-token",
			op:         types.IDPOpGet,
			namespace:  "staging",
			wantOK:     false,
			wantReason: types.IDPDenyReasonUnknownToken,
		},
		{
			name: "namespace restriction — allowed ns",
			idp: cfg(map[string]types.IDPTokenPermissions{
				"ci-pipeline": {
					Permissions: []string{"create"},
					Namespaces:  []string{"team-staging"},
				},
			}),
			token:     "ci-pipeline",
			op:        types.IDPOpCreate,
			namespace: "team-staging",
			wantOK:    true,
		},
		{
			name: "namespace restriction — denied ns",
			idp: cfg(map[string]types.IDPTokenPermissions{
				"ci-pipeline": {
					Permissions: []string{"create"},
					Namespaces:  []string{"team-staging"},
				},
			}),
			token:      "ci-pipeline",
			op:         types.IDPOpCreate,
			namespace:  "team-production",
			wantOK:     false,
			wantReason: types.IDPDenyReasonNamespace,
		},
		{
			name: "wildcard permission with namespace restriction",
			idp: cfg(map[string]types.IDPTokenPermissions{
				"control-center": {
					Permissions: []string{"*"},
					Namespaces:  []string{"team-staging", "team-production"},
				},
			}),
			token:     "control-center",
			op:        types.IDPOpDelete,
			namespace: "team-production",
			wantOK:    true,
		},
		{
			name: "wildcard permission — namespace outside restriction",
			idp: cfg(map[string]types.IDPTokenPermissions{
				"control-center": {
					Permissions: []string{"*"},
					Namespaces:  []string{"team-staging"},
				},
			}),
			token:      "control-center",
			op:         types.IDPOpDelete,
			namespace:  "kube-system",
			wantOK:     false,
			wantReason: types.IDPDenyReasonNamespace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, reason := tt.idp.TokenAllowed(tt.token, tt.op, tt.namespace)
			assert.Equal(t, tt.wantOK, ok)
			if !tt.wantOK {
				assert.Equal(t, tt.wantReason, reason)
				// Reason.Message must produce a non-empty, actionable string.
				msg := reason.Message(tt.token, tt.op, "MyKind", tt.namespace)
				assert.NotEmpty(t, msg)
				assert.Contains(t, msg, tt.token)
			}
		})
	}
}

func TestHasTokenRestrictions(t *testing.T) {
	assert.False(t, (&types.IDPConfig{}).HasTokenRestrictions())
	assert.False(t, (&types.IDPConfig{AllowedTokens: map[string]types.IDPTokenPermissions{}}).HasTokenRestrictions())
	assert.True(t, (&types.IDPConfig{
		AllowedTokens: map[string]types.IDPTokenPermissions{
			"ci": {Permissions: []string{"get"}},
		},
	}).HasTokenRestrictions())
}
