// Tests for the validation and mutation type definitions in admission.go.
//
// These types form the public contract between the Katalog schema and the
// admission/reconcile enforcement engine. Tests verify that default behaviours
// and policy helper methods match the documented semantics.
package orktypes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// ── EffectiveAction ───────────────────────────────────────────────────────────

func TestEffectiveAction_EmptyDefaultsToDeny(t *testing.T) {
	// Omitting action in a Katalog rule must default to deny — new rules
	// should block by default so that adding a rule is always safe.
	assert.Equal(t, orktypes.ValidationActionDeny, orktypes.EffectiveAction(""))
}

func TestEffectiveAction_ExplicitDenyPassthrough(t *testing.T) {
	assert.Equal(t, orktypes.ValidationActionDeny, orktypes.EffectiveAction(orktypes.ValidationActionDeny))
}

func TestEffectiveAction_ExplicitWarnPassthrough(t *testing.T) {
	assert.Equal(t, orktypes.ValidationActionWarn, orktypes.EffectiveAction(orktypes.ValidationActionWarn))
}

// ── ValidationAction.IsDeny / IsWarn ─────────────────────────────────────────

func TestValidationAction_IsDeny(t *testing.T) {
	assert.True(t, orktypes.ValidationActionDeny.IsDeny())
	assert.False(t, orktypes.ValidationActionDeny.IsWarn())
}

func TestValidationAction_IsWarn(t *testing.T) {
	assert.True(t, orktypes.ValidationActionWarn.IsWarn())
	assert.False(t, orktypes.ValidationActionWarn.IsDeny())
}

func TestValidationAction_EmptyIsDenyByDefault(t *testing.T) {
	var a orktypes.ValidationAction
	assert.True(t, a.IsDeny(), "empty action must be treated as deny")
	assert.False(t, a.IsWarn())
}

// ── ValidationConfig.HasDenyRules ────────────────────────────────────────────

func TestValidationConfig_HasDenyRules(t *testing.T) {
	t.Run("nil config returns false", func(t *testing.T) {
		var cfg *orktypes.ValidationConfig
		assert.False(t, cfg.HasDenyRules())
	})

	t.Run("empty rules slice returns false", func(t *testing.T) {
		cfg := &orktypes.ValidationConfig{}
		assert.False(t, cfg.HasDenyRules())
	})

	t.Run("explicit deny rule", func(t *testing.T) {
		cfg := &orktypes.ValidationConfig{
			Rules: []orktypes.ValidationRule{
				{Field: "spec.image", Action: orktypes.ValidationActionDeny},
			},
		}
		assert.True(t, cfg.HasDenyRules())
	})

	t.Run("omitted action defaults to deny", func(t *testing.T) {
		// A rule without an action field is deny — this is the
		// common authoring pattern and must be correctly detected.
		cfg := &orktypes.ValidationConfig{
			Rules: []orktypes.ValidationRule{
				{Field: "spec.image", Prefix: "myorg/"},
			},
		}
		assert.True(t, cfg.HasDenyRules())
	})

	t.Run("only warn rules returns false", func(t *testing.T) {
		cfg := &orktypes.ValidationConfig{
			Rules: []orktypes.ValidationRule{
				{Field: "metadata.labels.team", Action: orktypes.ValidationActionWarn},
			},
		}
		assert.False(t, cfg.HasDenyRules())
	})

	t.Run("mixed deny and warn — returns true", func(t *testing.T) {
		cfg := &orktypes.ValidationConfig{
			Rules: []orktypes.ValidationRule{
				{Field: "spec.image", Action: orktypes.ValidationActionDeny},
				{Field: "metadata.labels.team", Action: orktypes.ValidationActionWarn},
			},
		}
		assert.True(t, cfg.HasDenyRules())
	})
}

// ── ValidationConfig.HasWarnRules ────────────────────────────────────────────

func TestValidationConfig_HasWarnRules(t *testing.T) {
	t.Run("nil config returns false", func(t *testing.T) {
		var cfg *orktypes.ValidationConfig
		assert.False(t, cfg.HasWarnRules())
	})

	t.Run("empty rules returns false", func(t *testing.T) {
		cfg := &orktypes.ValidationConfig{}
		assert.False(t, cfg.HasWarnRules())
	})

	t.Run("explicit warn rule", func(t *testing.T) {
		cfg := &orktypes.ValidationConfig{
			Rules: []orktypes.ValidationRule{
				{Field: "metadata.labels.team", Action: orktypes.ValidationActionWarn},
			},
		}
		assert.True(t, cfg.HasWarnRules())
	})

	t.Run("deny-only rules return false", func(t *testing.T) {
		cfg := &orktypes.ValidationConfig{
			Rules: []orktypes.ValidationRule{
				{Field: "spec.image", Action: orktypes.ValidationActionDeny},
				{Field: "spec.replicas"},
			},
		}
		assert.False(t, cfg.HasWarnRules())
	})

	t.Run("mixed deny and warn — returns true", func(t *testing.T) {
		cfg := &orktypes.ValidationConfig{
			Rules: []orktypes.ValidationRule{
				{Field: "spec.image", Action: orktypes.ValidationActionDeny},
				{Field: "metadata.labels.team", Action: orktypes.ValidationActionWarn},
			},
		}
		assert.True(t, cfg.HasWarnRules())
		assert.True(t, cfg.HasDenyRules())
	})
}

// ── AdmissionWebhookConfig.WebhookValidationEnabled ──────────────────────────

func boolPtr(v bool) *bool { return &v }

func TestWebhookValidationEnabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  *orktypes.AdmissionWebhookConfig
		want bool
	}{
		{
			name: "nil config — enabled by default",
			cfg:  nil,
			want: true,
		},
		{
			name: "config with nil Validation — enabled by default",
			cfg:  &orktypes.AdmissionWebhookConfig{},
			want: true,
		},
		{
			name: "Validation = true — enabled",
			cfg:  &orktypes.AdmissionWebhookConfig{Validation: boolPtr(true)},
			want: true,
		},
		{
			name: "Validation = false — disabled",
			cfg:  &orktypes.AdmissionWebhookConfig{Validation: boolPtr(false)},
			want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.cfg.WebhookValidationEnabled())
		})
	}
}

// ── AdmissionWebhookConfig.WebhookMutationEnabled ────────────────────────────

func TestWebhookMutationEnabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  *orktypes.AdmissionWebhookConfig
		want bool
	}{
		{
			name: "nil config — enabled by default",
			cfg:  nil,
			want: true,
		},
		{
			name: "config with nil Mutation — enabled by default",
			cfg:  &orktypes.AdmissionWebhookConfig{},
			want: true,
		},
		{
			name: "Mutation = true — enabled",
			cfg:  &orktypes.AdmissionWebhookConfig{Mutation: boolPtr(true)},
			want: true,
		},
		{
			name: "Mutation = false — disabled",
			cfg:  &orktypes.AdmissionWebhookConfig{Mutation: boolPtr(false)},
			want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.cfg.WebhookMutationEnabled())
		})
	}
}

// ── AdmissionWebhookConfig.EffectiveOperations ───────────────────────────────

func TestEffectiveOperations(t *testing.T) {
	t.Run("nil config returns CREATE and UPDATE", func(t *testing.T) {
		var cfg *orktypes.AdmissionWebhookConfig
		assert.Equal(t, []string{"CREATE", "UPDATE"}, cfg.EffectiveOperations())
	})

	t.Run("empty operations returns CREATE and UPDATE", func(t *testing.T) {
		cfg := &orktypes.AdmissionWebhookConfig{}
		assert.Equal(t, []string{"CREATE", "UPDATE"}, cfg.EffectiveOperations())
	})

	t.Run("custom operations returned as declared", func(t *testing.T) {
		cfg := &orktypes.AdmissionWebhookConfig{
			Operations: []string{"CREATE", "UPDATE", "DELETE"},
		}
		assert.Equal(t, []string{"CREATE", "UPDATE", "DELETE"}, cfg.EffectiveOperations())
	})

	t.Run("single operation override", func(t *testing.T) {
		cfg := &orktypes.AdmissionWebhookConfig{
			Operations: []string{"CREATE"},
		}
		assert.Equal(t, []string{"CREATE"}, cfg.EffectiveOperations())
	})
}

// ── MutationConfig.MutateFirst ────────────────────────────────────────────────

func TestMutationConfig_MutateFirstDefault(t *testing.T) {
	// MutateFirst defaults to false — validate before mutate at reconcile time.
	// Admission time always mutates first regardless of this setting.
	cfg := &orktypes.MutationConfig{
		Rules: []orktypes.MutationRule{
			{Field: "spec.replicas", Default: "2"},
		},
	}
	assert.False(t, cfg.MutateFirst)
}
