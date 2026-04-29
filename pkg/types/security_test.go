// Tests for KatalogSecurity methods (security.go).
package types_test

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
)

// ── IsDeletionProtectionEnabled ───────────────────────────────────────────────

func TestIsDeletionProtectionEnabled_Nil(t *testing.T) {
	var s *orktypes.KatalogSecurity
	assert.False(t, s.IsDeletionProtectionEnabled())
}

func TestIsDeletionProtectionEnabled_NilBlock(t *testing.T) {
	s := &orktypes.KatalogSecurity{}
	assert.False(t, s.IsDeletionProtectionEnabled())
}

func TestIsDeletionProtectionEnabled_DeclaredNoEnabled(t *testing.T) {
	// Declared but no explicit enabled field = enabled by default
	s := &orktypes.KatalogSecurity{DeletionProtection: &orktypes.DeletionProtectionConfig{}}
	assert.True(t, s.IsDeletionProtectionEnabled())
}

func TestIsDeletionProtectionEnabled_ExplicitFalse(t *testing.T) {
	f := false
	s := &orktypes.KatalogSecurity{DeletionProtection: &orktypes.DeletionProtectionConfig{Enabled: &f}}
	assert.False(t, s.IsDeletionProtectionEnabled())
}

func TestIsDeletionProtectionEnabled_ExplicitTrue(t *testing.T) {
	tr := true
	s := &orktypes.KatalogSecurity{DeletionProtection: &orktypes.DeletionProtectionConfig{Enabled: &tr}}
	assert.True(t, s.IsDeletionProtectionEnabled())
}

// ── IsAdmissionEnabled ────────────────────────────────────────────────────────

func TestIsAdmissionEnabled_Nil(t *testing.T) {
	var s *orktypes.KatalogSecurity
	assert.False(t, s.IsAdmissionEnabled())
}

func TestIsAdmissionEnabled_NoWebhooks(t *testing.T) {
	s := &orktypes.KatalogSecurity{}
	assert.False(t, s.IsAdmissionEnabled())
}

func TestIsAdmissionEnabled_NoAdmission(t *testing.T) {
	s := &orktypes.KatalogSecurity{Webhooks: &orktypes.WebhooksConfig{}}
	assert.False(t, s.IsAdmissionEnabled())
}

func TestIsAdmissionEnabled_NilEnabledField(t *testing.T) {
	// Admission declared but Enabled not set → false (no default-on for webhooks)
	s := &orktypes.KatalogSecurity{
		Webhooks: &orktypes.WebhooksConfig{Admission: &orktypes.AdmissionWebhookToggle{}},
	}
	assert.False(t, s.IsAdmissionEnabled())
}

func TestIsAdmissionEnabled_ExplicitTrue(t *testing.T) {
	tr := true
	s := &orktypes.KatalogSecurity{
		Webhooks: &orktypes.WebhooksConfig{
			Admission: &orktypes.AdmissionWebhookToggle{Enabled: &tr},
		},
	}
	assert.True(t, s.IsAdmissionEnabled())
}

// ── IsConversionEnabled ───────────────────────────────────────────────────────

func TestIsConversionEnabled_Nil(t *testing.T) {
	var s *orktypes.KatalogSecurity
	assert.False(t, s.IsConversionEnabled())
}

func TestIsConversionEnabled_NilConversion(t *testing.T) {
	s := &orktypes.KatalogSecurity{}
	assert.False(t, s.IsConversionEnabled())
}

func TestIsConversionEnabled_ExplicitTrue(t *testing.T) {
	tr := true
	s := &orktypes.KatalogSecurity{Conversion: &orktypes.ConversionConfig{Enabled: &tr}}
	assert.True(t, s.IsConversionEnabled())
}

// ── DeletionProtectionFailurePolicy ──────────────────────────────────────────

func TestDeletionProtectionFailurePolicy_DefaultFail(t *testing.T) {
	var s *orktypes.KatalogSecurity
	assert.Equal(t, "Fail", s.DeletionProtectionFailurePolicy())
}

func TestDeletionProtectionFailurePolicy_Custom(t *testing.T) {
	s := &orktypes.KatalogSecurity{
		DeletionProtection: &orktypes.DeletionProtectionConfig{FailurePolicy: "Ignore"},
	}
	assert.Equal(t, "Ignore", s.DeletionProtectionFailurePolicy())
}

// ── IsNamespaceProtectionEnabled ──────────────────────────────────────────────

func TestIsNamespaceProtectionEnabled_Nil(t *testing.T) {
	var s *orktypes.KatalogSecurity
	assert.False(t, s.IsNamespaceProtectionEnabled())
}

func TestIsNamespaceProtectionEnabled_DeclaredNoEnabled(t *testing.T) {
	s := &orktypes.KatalogSecurity{NamespaceProtection: &orktypes.NamespaceProtectionConfig{}}
	assert.True(t, s.IsNamespaceProtectionEnabled())
}

func TestIsNamespaceProtectionEnabled_ExplicitFalse(t *testing.T) {
	f := false
	s := &orktypes.KatalogSecurity{
		NamespaceProtection: &orktypes.NamespaceProtectionConfig{Enabled: &f},
	}
	assert.False(t, s.IsNamespaceProtectionEnabled())
}

// ── NamespaceProtectionFailurePolicy ─────────────────────────────────────────

func TestNamespaceProtectionFailurePolicy_DefaultFail(t *testing.T) {
	var s *orktypes.KatalogSecurity
	assert.Equal(t, "Fail", s.NamespaceProtectionFailurePolicy())
}

func TestNamespaceProtectionFailurePolicy_Custom(t *testing.T) {
	s := &orktypes.KatalogSecurity{
		NamespaceProtection: &orktypes.NamespaceProtectionConfig{FailurePolicy: "Ignore"},
	}
	assert.Equal(t, "Ignore", s.NamespaceProtectionFailurePolicy())
}

// ── DeletionProtectionServiceName / OrkestraServiceName ──────────────────────

func TestDeletionProtectionServiceName_FallsBackToEnvDefault(t *testing.T) {
	var s *orktypes.KatalogSecurity
	assert.Equal(t, "orkestra-webhook", s.DeletionProtectionServiceName("orkestra-webhook"))
}

func TestDeletionProtectionServiceName_Custom(t *testing.T) {
	s := &orktypes.KatalogSecurity{
		DeletionProtection: &orktypes.DeletionProtectionConfig{ServiceName: "my-webhook"},
	}
	assert.Equal(t, "my-webhook", s.DeletionProtectionServiceName("default"))
}

func TestOrkestraServiceName_FallsBackToEnvDefault(t *testing.T) {
	var s *orktypes.KatalogSecurity
	assert.Equal(t, "orkestra", s.OrkestraServiceName("orkestra"))
}

func TestOrkestraServiceName_Custom(t *testing.T) {
	s := &orktypes.KatalogSecurity{ServiceName: "my-orkestra"}
	assert.Equal(t, "my-orkestra", s.OrkestraServiceName("orkestra"))
}
