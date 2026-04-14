// pkg/katalog/security.go
//
// Security accessors on *Katalog.
//
// Precedence (highest → lowest):
//  1. Katalog YAML (k.Security)
//  2. ENV vars via SecurityConfig (k.konfig.Security())
//  3. Hard defaults coded below
//
// KatalogSecurity uses *bool fields which allows detecting
// "not declared" (nil) vs "explicitly false" (*false).
//
// Deletion protection is ENABLED BY DEFAULT when the security block
// is present but deletionProtection is not declared.
// RBAC is also enabled by default when the rbac block is present.
package katalog

// securityEnvDefaults returns the SecurityConfig from konfig, or a zero
// value when konfig is not wired (e.g. NewEmptyKatalog in tests).
func (k *Katalog) securityEnvDefaults() interface {
	DeletionProtectionEnabled() bool
	DeletionProtectionSvcName() string
	DeletionProtectionPolicy() string
	AdmissionEnabled() bool
	ConversionEnabled() bool
	WebhooksSvcName() string
	WebhooksPolicy() string
	RBACEnabled() bool
	RBACCleanup() bool
	ConversionWindowVal() int
} {
	return &envSecurityReader{k: k}
}

// envSecurityReader adapts *konfig.SecurityConfig through a small interface
// so that security.go does not import konfig directly.
type envSecurityReader struct{ k *Katalog }

func (r *envSecurityReader) DeletionProtectionEnabled() bool {
	if r.k.konfig == nil {
		return false
	}
	return r.k.konfig.Security().DeletionProtection.Enabled
}
func (r *envSecurityReader) DeletionProtectionSvcName() string {
	if r.k.konfig == nil {
		return "orkestra"
	}
	return r.k.konfig.Security().DeletionProtection.ServiceName
}
func (r *envSecurityReader) DeletionProtectionPolicy() string {
	if r.k.konfig == nil {
		return "Fail"
	}
	return r.k.konfig.Security().DeletionProtection.FailurePolicy
}
func (r *envSecurityReader) AdmissionEnabled() bool {
	if r.k.konfig == nil {
		return false
	}
	return r.k.konfig.Security().Webhooks.Admission.Enabled
}
func (r *envSecurityReader) ConversionEnabled() bool {
	if r.k.konfig == nil {
		return false
	}
	return r.k.konfig.Security().Conversion.Enabled
}
func (r *envSecurityReader) WebhooksSvcName() string {
	if r.k.konfig == nil {
		return "orkestra"
	}
	return r.k.konfig.Security().Webhooks.ServiceName
}
func (r *envSecurityReader) WebhooksPolicy() string {
	if r.k.konfig == nil {
		return "Ignore"
	}
	return r.k.konfig.Security().Webhooks.FailurePolicy
}
func (r *envSecurityReader) RBACEnabled() bool {
	if r.k.konfig == nil {
		return false
	}
	return r.k.konfig.Security().RBAC.Enabled
}
func (r *envSecurityReader) RBACCleanup() bool {
	if r.k.konfig == nil {
		return false
	}
	return r.k.konfig.Security().RBAC.CleanupOnShutdown
}
func (r *envSecurityReader) ConversionWindowVal() int {
	if r.k.konfig == nil {
		return 100
	}
	return r.k.konfig.Security().Conversion.ConversionWindow
}

// ── Deletion protection ───────────────────────────────────────────────────────

// IsDeletionProtectionEnabled reports whether deletion protection is active.
//
// Precedence:
//
//	YAML security.deletionProtection block present → use YAML value (default-on when block declared)
//	YAML block absent                              → fall back to ENABLE_DELETION_PROTECTION env
func (k *Katalog) IsDeletionProtectionEnabled() bool {
	if k.Security.DeletionProtection != nil {
		// YAML declared the block — apply its enabled semantics.
		return k.Security.IsDeletionProtectionEnabled()
	}
	return k.securityEnvDefaults().DeletionProtectionEnabled()
}

// DeletionProtectionServiceName returns the effective service name for deletion protection.
// YAML value takes precedence over ENV.
func (k *Katalog) DeletionProtectionServiceName() string {
	env := k.securityEnvDefaults()
	return k.Security.DeletionProtectionServiceName(env.DeletionProtectionSvcName())
}

// DeletionProtectionFailurePolicy returns the effective failure policy string.
// YAML value takes precedence over ENV.
func (k *Katalog) DeletionProtectionFailurePolicy() string {
	if k.Security.DeletionProtection != nil && k.Security.DeletionProtection.FailurePolicy != "" {
		return k.Security.DeletionProtection.FailurePolicy
	}
	return k.securityEnvDefaults().DeletionProtectionPolicy()
}

// ── Admission webhooks ────────────────────────────────────────────────────────

// IsAdmissionEnabled reports whether admission webhooks are globally enabled.
//
// Precedence:
//
//	YAML security.webhooks.admission block present → use YAML value
//	YAML block absent                              → fall back to ENABLE_ADMISSION_WEBHOOK env
func (k *Katalog) IsAdmissionEnabled() bool {
	if k.Security.Webhooks != nil && k.Security.Webhooks.Admission != nil {
		return k.Security.IsAdmissionEnabled()
	}
	return k.securityEnvDefaults().AdmissionEnabled()
}

// ── Conversion webhook ────────────────────────────────────────────────────────

// IsConversionEnabled reports whether the conversion webhook is globally enabled.
//
// Precedence:
//
//	YAML security.conversion block present → use YAML value
//	YAML block absent                      → fall back to ENABLE_CONVERSION env
func (k *Katalog) IsConversionEnabled() bool {
	if k.Security.Conversion != nil {
		return k.Security.IsConversionEnabled()
	}
	return k.securityEnvDefaults().ConversionEnabled()
}

// WebhooksServiceName returns the effective service name for admission/conversion webhooks.
// YAML value takes precedence over ENV.
func (k *Katalog) WebhooksServiceName() string {
	env := k.securityEnvDefaults()
	return k.Security.WebhooksServiceName(env.WebhooksSvcName())
}

// WebhooksFailurePolicy returns the effective failure policy for admission webhooks.
// YAML value takes precedence over ENV.
func (k *Katalog) WebhooksFailurePolicy() string {
	env := k.securityEnvDefaults()
	return k.Security.WebhooksFailurePolicy(env.WebhooksPolicy())
}

// ── RBAC ──────────────────────────────────────────────────────────────────────

// IsRBACEnabled reports whether RBAC auto-apply is active.
//
// Precedence:
//
//	YAML security.rbac block present → use YAML value (default-on when block declared)
//	YAML block absent                → fall back to RBAC_AUTO_APPLY env
func (k *Katalog) IsRBACEnabled() bool {
	if k.Security.RBAC != nil {
		return k.Security.IsRBACEnabled()
	}
	return k.securityEnvDefaults().RBACEnabled()
}

// RBACCleanupOnShutdown reports whether RBAC should be deleted on shutdown.
//
// Precedence:
//
//	YAML security.rbac.cleanupOnShutdown present → use YAML value
//	YAML block absent                            → fall back to RBAC_CLEANUP_ON_SHUTDOWN env
func (k *Katalog) RBACCleanupOnShutdown() bool {
	if k.Security.RBAC != nil {
		return k.Security.RBAC.CleanupOnShutdown
	}
	return k.securityEnvDefaults().RBACCleanup()
}

// ── Conversion stats window ───────────────────────────────────────────────────

// ConversionWindow returns the effective rolling window size for conversion/admission stats.
//
// Precedence:
//
//	YAML security.conversion.conversionWindow > 0 → use YAML value
//	YAML absent or zero                           → fall back to CONVERSION_WINDOW env
func (k *Katalog) ConversionWindow() int {
	if k.Security.Conversion != nil && k.Security.Conversion.ConversionWindow > 0 {
		return k.Security.Conversion.ConversionWindow
	}
	return k.securityEnvDefaults().ConversionWindowVal()
}

// ── Certificates ──────────────────────────────────────────────────────────────

// NeedsCertificates reports whether Orkestra must generate TLS certificates.
//
// Certificates are required when deletion protection, admission webhooks,
// or conversion webhooks are enabled — all three use the same TLS cert.
func (k *Katalog) NeedsCertificates() bool {
	return k.IsDeletionProtectionEnabled() ||
		k.IsAdmissionEnabled() ||
		k.IsConversionEnabled()
}
