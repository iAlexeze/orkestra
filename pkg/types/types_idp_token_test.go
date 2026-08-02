package types_test

import (
	"testing"

	"github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestTokenAllowed_PermissionClasses(t *testing.T) {
	cfg := func(tokens map[string]types.IDPTokenPermissions) *types.IDPConfig {
		return &types.IDPConfig{AllowedTokens: types.IDPAllowedTokens{Tokens: tokens}}
	}

	tests := []struct {
		name       string
		idp        *types.IDPConfig
		token      string
		op         string
		namespace  string
		class      types.IDPEndpointClass
		wantOK     bool
		wantReason types.IDPDenyReason
	}{
		{
			name:  "global wildcard allows all classes",
			idp:   cfg(map[string]types.IDPTokenPermissions{"cc": {Permissions: types.IDPPermissionSet{Global: []string{"*"}}}}),
			token: "cc", op: types.IDPOpDelete, class: types.IDPClassResources,
			wantOK: true,
		},
		{
			name: "schema class used for schema op, not resources",
			idp: cfg(map[string]types.IDPTokenPermissions{
				"audit": {Permissions: types.IDPPermissionSet{
					Schema:    []string{"get"},
					Resources: []string{"get", "list"},
				}},
			}),
			token: "audit", op: types.IDPOpGet, class: types.IDPClassSchema,
			wantOK: true,
		},
		{
			name: "schema class denies create — not in schema perms",
			idp: cfg(map[string]types.IDPTokenPermissions{
				"audit": {Permissions: types.IDPPermissionSet{
					Schema:    []string{"get"},
					Resources: []string{"get", "list"},
				}},
			}),
			// create is a resources operation, not schema — but we check class
			// correctly here: schema class, create op → schema has only get.
			token: "audit", op: types.IDPOpCreate, class: types.IDPClassSchema,
			wantOK:     false,
			wantReason: types.IDPDenyReasonOperation,
		},
		{
			name: "falls back to global when class list is empty",
			idp: cfg(map[string]types.IDPTokenPermissions{
				"ci": {Permissions: types.IDPPermissionSet{
					Global: []string{"create", "update", "get"},
					// schema not set → inherits global
				}},
			}),
			token: "ci", op: types.IDPOpGet, class: types.IDPClassSchema,
			wantOK: true,
		},
		{
			name: "empty permissions denies all",
			idp: cfg(map[string]types.IDPTokenPermissions{
				"empty": {Permissions: types.IDPPermissionSet{}},
			}),
			token: "empty", op: types.IDPOpGet, class: types.IDPClassResources,
			wantOK:     false,
			wantReason: types.IDPDenyReasonOperation,
		},
		{
			name: "namespace restriction respected regardless of class",
			idp: cfg(map[string]types.IDPTokenPermissions{
				"ci": {
					Namespaces:  []string{"staging"},
					Permissions: types.IDPPermissionSet{Global: []string{"*"}},
				},
			}),
			token: "ci", op: types.IDPOpCreate, namespace: "production",
			class:      types.IDPClassResources,
			wantOK:     false,
			wantReason: types.IDPDenyReasonNamespace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, reason := tt.idp.TokenAllowed(tt.token, tt.op, tt.namespace, tt.class)
			assert.Equal(t, tt.wantOK, ok)
			if !tt.wantOK {
				assert.Equal(t, tt.wantReason, reason)
				assert.NotEmpty(t, reason.Message(tt.token, tt.op, "MyKind", tt.namespace))
			}
		})
	}
}

func TestIDPPermissionSetIsEmpty(t *testing.T) {
	assert.True(t, types.IDPPermissionSet{}.IsEmpty())
	assert.False(t, types.IDPPermissionSet{Global: []string{"get"}}.IsEmpty())
	assert.False(t, types.IDPPermissionSet{Schema: []string{"get"}}.IsEmpty())
	assert.False(t, types.IDPPermissionSet{Resources: []string{"get"}}.IsEmpty())
}
