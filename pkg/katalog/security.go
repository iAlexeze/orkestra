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
package katalog

import "time"

// securityEnvDefaults returns the SecurityConfig from konfig, or a zero
// value when konfig is not wired (e.g. NewEmptyKatalog in tests).
func (k *Katalog) securityEnvDefaults() interface {
	OrkestraServiceName() string
	DeletionProtectionEnabled() bool
	DeletionProtectionSvcName() string
	DeletionProtectionPolicy() string
	DeletionProtectionCleanup() bool
	AdmissionEnabled() bool
	ConversionEnabled() bool
	WebhooksSvcName() string
	WebhooksPolicy() string
	WebhookCleanup() bool
	ConversionWindowVal() int
	WebhookControllerEnabled() bool
	WebhookControllerSyncInterval() time.Duration
	NamespaceProtectionEnabled() bool
	NamespaceProtectionSvcName() string
	NamespaceProtectionPolicy() string
	NamespaceProtectionCleanup() bool
} {
	return &envSecurityReader{k: k}
}

const (
	ork = "orkestra"
)

// envSecurityReader adapts *konfig.SecurityConfig through a small interface
// so that security.go does not import konfig directly.
type envSecurityReader struct{ k *Katalog }

func (r *envSecurityReader) OrkestraServiceName() string {
	if r.k.konfig == nil {
		return "orkestra-runtime"
	}
	return r.k.konfig.Security().ServiceName
}

func (r *envSecurityReader) DeletionProtectionEnabled() bool {
	if r.k.konfig == nil {
		return false
	}
	return r.k.konfig.Security().DeletionProtection.Enabled
}
func (r *envSecurityReader) DeletionProtectionSvcName() string {
	if r.k.konfig == nil {
		return ork
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
		return ork
	}
	return r.k.konfig.Security().Webhooks.ServiceName
}
func (r *envSecurityReader) WebhooksPolicy() string {
	if r.k.konfig == nil {
		return "Ignore"
	}
	return r.k.konfig.Security().Webhooks.FailurePolicy
}

func (r *envSecurityReader) WebhookCleanup() bool {
	if r.k.konfig == nil {
		return false
	}
	return r.k.konfig.Security().Webhooks.CleanupOnShutdown
}
func (r *envSecurityReader) DeletionProtectionCleanup() bool {
	if r.k.konfig == nil {
		return false
	}
	return r.k.konfig.Security().DeletionProtection.CleanupOnShutdown
}
func (r *envSecurityReader) NamespaceProtectionCleanup() bool {
	if r.k.konfig == nil {
		return false
	}
	return r.k.konfig.Security().NamespaceProtection.CleanupOnShutdown
}

func (r *envSecurityReader) ConversionWindowVal() int {
	if r.k.konfig == nil {
		return 100
	}
	return r.k.konfig.Security().Conversion.ConversionWindow
}
func (r *envSecurityReader) WebhookControllerEnabled() bool {
	if r.k.konfig == nil {
		return false
	}
	return r.k.konfig.Security().Webhooks.Controller.Enabled
}
func (r *envSecurityReader) WebhookControllerSyncInterval() time.Duration {
	if r.k.konfig == nil {
		return 30 * time.Second
	}
	return r.k.konfig.Security().Webhooks.Controller.SyncInterval
}
func (r *envSecurityReader) NamespaceProtectionEnabled() bool {
	if r.k.konfig == nil {
		return false
	}
	return r.k.konfig.Security().NamespaceProtection.Enabled
}
func (r *envSecurityReader) NamespaceProtectionPolicy() string {
	if r.k.konfig == nil {
		return "Fail"
	}
	return r.k.konfig.Security().NamespaceProtection.FailurePolicy
}
func (r *envSecurityReader) NamespaceProtectionSvcName() string {
	if r.k.konfig == nil {
		return ork
	}
	return r.k.konfig.Security().NamespaceProtection.ServiceName
}

//

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

// Orkestra Service Name returns the effective service name for orkestra.
// YAML value takes precedence over ENV.
func (k *Katalog) OrkestraServiceName() string {
	env := k.securityEnvDefaults()
	return k.Security.OrkestraServiceName(env.OrkestraServiceName())
}

// DeletionProtectionFailurePolicy returns the effective failure policy string.
// YAML value takes precedence over ENV.
func (k *Katalog) DeletionProtectionFailurePolicy() string {
	if k.Security.DeletionProtection != nil && k.Security.DeletionProtection.FailurePolicy != "" {
		return k.Security.DeletionProtection.FailurePolicy
	}
	return k.securityEnvDefaults().DeletionProtectionPolicy()
}

// IsStrictModeEnabled reports whether strict mode is active for deletion protection.
// Strict mode treats removal of the deletion-protection label as a deletion attempt.
// Only valid when deletion protection is also enabled.
func (k *Katalog) IsStrictModeEnabled() bool {
	if k.Security.DeletionProtection == nil {
		return false
	}
	return k.Security.DeletionProtection.StrictMode
}

// ── Namespace protection ───────────────────────────────────────────────────────

// IsNamespaceProtectionEnabled reports whether namespace protection is active.
//
// Precedence:
//
//	YAML security.namespaceProtection block present → use YAML value (default-on when block declared)
//	YAML block absent                               → fall back to ENABLE_NAMESPACE_PROTECTION env
func (k *Katalog) IsNamespaceProtectionEnabled() bool {
	if k.Security.NamespaceProtection != nil {
		// YAML declared the block — apply its enabled semantics.
		return k.Security.IsNamespaceProtectionEnabled()
	}
	return k.securityEnvDefaults().NamespaceProtectionEnabled()
}

// NamespaceProtectionServiceName returns the effective service name for namespace protection.
// YAML value takes precedence over ENV.
func (k *Katalog) NamespaceProtectionServiceName() string {
	env := k.securityEnvDefaults()
	return k.Security.NamespaceProtectionServiceName(env.NamespaceProtectionSvcName())
}

// NamespaceProtectionFailurePolicy returns the effective failure policy string.
// YAML value takes precedence over ENV.
func (k *Katalog) NamespaceProtectionFailurePolicy() string {
	if k.Security.NamespaceProtection != nil && k.Security.NamespaceProtection.FailurePolicy != "" {
		return k.Security.NamespaceProtection.FailurePolicy
	}
	return k.securityEnvDefaults().NamespaceProtectionPolicy()
}

// NamespaceProtectionCleanupOnShutdown reports whether Deletion Protection should be deleted on shutdown.
//
// Precedence:
//
//	YAML security.namespaceProtection.cleanupOnShutdown present → use YAML value
//	YAML block absent                                      → fall back to NAMESPACE_PROTECTION_CLEANUP_ON_SHUTDOWN env
func (k *Katalog) NamespaceProtectionCleanupOnShutdown() bool {
	if k.Security.NamespaceProtection != nil {
		return k.Security.NamespaceProtection.CleanupOnShutdown
	}
	return k.securityEnvDefaults().NamespaceProtectionCleanup()
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

// DeletionProtectionCleanupOnShutdown reports whether Deletion Protection should be deleted on shutdown.
//
// Precedence:
//
//	YAML security.deletionProtection.cleanupOnShutdown present → use YAML value
//	YAML block absent                                      → fall back to DELETION_PROTECTION_CLEANUP_ON_SHUTDOWN env
func (k *Katalog) DeletionProtectionCleanupOnShutdown() bool {
	if k.Security.DeletionProtection != nil {
		return k.Security.DeletionProtection.CleanupOnShutdown
	}
	return k.securityEnvDefaults().DeletionProtectionCleanup()
}

// WebhookCleanupOnShutdown reports whether admission webhooks should be deleted on shutdown.
//
// Precedence:
//
//	YAML security.webhooks.cleanupOnShutdown present → use YAML value
//	YAML block absent                                → fall back to WEBHOOK_CLEANUP_ON_SHUTDOWN env
func (k *Katalog) WebhookCleanupOnShutdown() bool {
	if k.Security.Webhooks != nil {
		return k.Security.Webhooks.CleanupOnShutdown
	}
	return k.securityEnvDefaults().WebhookCleanup()
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
// or conversion webhooks are enabled with valid usecases
// configured in at least 1 CRD— all three use the same TLS cert.
func (k *Katalog) NeedsCertificates() bool {
	return k.IsDeletionProtectionEnabled() ||
		k.IsNamespaceProtectionEnabled() ||
		k.HasValidationOrMutationRules() || // Valid admission rule
		k.HasConversionPaths()
}

// IsWebhookControllerEnabled reports whether the webhook controller is enabled.
func (k *Katalog) IsWebhookControllerEnabled() bool {
	return k.securityEnvDefaults().WebhookControllerEnabled()
}

// WebhookControllerSyncInterval returns the webhook controller sync interval.
func (k *Katalog) WebhookControllerSyncInterval() time.Duration {
	return k.securityEnvDefaults().WebhookControllerSyncInterval()
}
