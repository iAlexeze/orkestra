// pkg/types/security.go
//
// Security configuration at the Katalog level.
//
// The unified security block covers four concerns:
//
//  1. Deletion protection — webhook that blocks deletion of managed CRDs and the operator itself.
//  2. Admission webhooks  — ValidatingWebhookConfiguration and MutatingWebhookConfiguration.
//  3. Conversion          — CRD version conversion via /convert (separate from webhooks).
//  4. RBAC auto-apply     — ClusterRole, ClusterRoleBinding, ServiceAccount at startup.
//
// YAML shape:
//
//	security:
//	  deletionProtection:
//	    enabled: true            # default: true when block is present
//	    serviceName: orkestra    # default: ORKESTRA_SERVICE_NAME env / "orkestra"
//	    failurePolicy: Fail      # default: Fail
//
//	  webhooks:
//	    admission:
//	      enabled: true          # default: ENABLE_ADMISSION_WEBHOOK env / false
//	    failurePolicy: Ignore    # default: WEBHOOKS_FAILURE_POLICY env / "Ignore"
//	    serviceName: orkestra    # default: ORKESTRA_SERVICE_NAME env / "orkestra"
//
//	  conversion:
//	    enabled: true            # default: ENABLE_CONVERSION env / false
//	    conversionWindow: 100    # default: CONVERSION_WINDOW env / 100
//
// Precedence: katalog YAML value > ENV value > hard default.
// ENV values populate SecurityConfig during Init() and act as defaults.
// Katalog values are merged on top in KomposeKatalogFromYaml.
package types

// KatalogSecurity holds the full security configuration for a Katalog.
type KatalogSecurity struct {
	// DeletionProtection controls whether Orkestra registers a webhook that
	// blocks deletion of its managed CRDs, deployment, service, etc.
	//
	// When enabled (default when block is present):
	//   - Registers /deletion-protection endpoint on the HTTPS server
	//   - Creates ValidatingWebhookConfiguration "orkestra-delete-protection"
	//   - Entry 1: broad rule for CRDs; handler filters by ProtectedCRDNames()
	//   - Entry 2: ObjectSelector-gated rule for deployment, service, etc
	//
	// To decommission an operator with deletion protection:
	//   1. Set enabled: false
	//   2. Redeploy Orkestra (webhook removed on startup)
	//   3. Delete resources normally
	//
	// nil pointer: not enabled (not declared in YAML).
	DeletionProtection *DeletionProtectionConfig `yaml:"deletionProtection,omitempty"`

	// Webhooks controls the admission webhook settings (ValidatingWebhookConfiguration,
	// MutatingWebhookConfiguration). These apply globally — there is no per-CRD switch.
	// CRDs declare validation/mutation rules; the webhook is enabled or not globally.
	//
	// nil pointer: admission webhooks not configured; ENV vars drive behavior.
	Webhooks *WebhooksConfig `yaml:"webhooks,omitempty"`

	// Conversion controls the /convert endpoint and CRD version conversion.
	// Separate from admission webhooks — conversion has its own endpoint and
	// configuration (window size for stats, etc.).
	//
	// nil pointer: conversion not configured; ENV vars drive behavior.
	Conversion *ConversionConfig `yaml:"conversion,omitempty"`

	// NamespaceProtection controls the optional validating webhook that prevents
	// Orkestra-managed CRs from being created or updated in forbidden namespaces.
	//
	// This is an admission-time safeguard only. When enabled:
	//   - Registers /namespace-protection endpoint on the HTTPS server
	//   - Creates ValidatingWebhookConfiguration "orkestra-namespace-protection"
	//   - Enforces allowedNamespaces / restrictedNamespaces declared by each CRD
	//
	// If disabled or omitted, namespace rules are not enforced at apply time.
	// Existing CRs in forbidden namespaces will still be reconciled normally.
	//
	// nil pointer: namespace protection not configured; ENV vars drive behavior.
	NamespaceProtection *NamespaceProtectionConfig `yaml:"namespaceProtection,omitempty"`
}

// DeletionProtectionConfig controls deletion protection behaviour.
type DeletionProtectionConfig struct {
	// Enabled controls whether deletion protection is active.
	// Default: true when the deletionProtection block is declared.
	Enabled *bool `yaml:"enabled,omitempty"`

	// ServiceName is the Kubernetes Service fronting Orkestra's HTTPS server.
	// The API server sends webhook requests to this Service.
	// Default: ORKESTRA_SERVICE_NAME env / "orkestra".
	ServiceName string `yaml:"serviceName,omitempty"`

	// FailurePolicy controls what the API server does when Orkestra is unreachable.
	// "Fail" — reject the DELETE (recommended for protection; this is the default).
	// "Ignore" — allow the DELETE through when Orkestra cannot be reached.
	// Default: "Fail".
	FailurePolicy string `yaml:"failurePolicy,omitempty"`

	// CleanupOnShutdown controls whether Deletion protection webhook is deleted on graceful shutdown.
	// Default: false — Deletion protection webhook persists across restarts.
	CleanupOnShutdown bool `yaml:"cleanupOnShutdown,omitempty"`
}

// NamespaceProtectionConfig controls namespace-protection webhook behaviour.
type NamespaceProtectionConfig struct {
	// Enabled controls whether namespace protection is active.
	// Default: true when the namespaceProtection block is declared.
	Enabled *bool `yaml:"enabled,omitempty"`

	// ServiceName is the Kubernetes Service fronting Orkestra's HTTPS server.
	// The API server sends webhook requests to this Service.
	// Default: ORKESTRA_SERVICE_NAME env / "orkestra".
	ServiceName string `yaml:"serviceName,omitempty"`

	// FailurePolicy controls what the API server does when Orkestra is unreachable.
	// "Fail"   — reject the CREATE/UPDATE (recommended; this is the default).
	// "Ignore" — allow the request through when Orkestra cannot be reached.
	// Default: "Fail".
	FailurePolicy string `yaml:"failurePolicy,omitempty"`

	// RestrictedNamespaces — deny-list applied to every CRD in this Katalog.
	// Merged additively with per-CRD restrictedNamespaces — more specific levels
	// add to, not replace, broader levels.
	RestrictedNamespaces RestrictedNamespaces `yaml:"restrictedNamespaces,omitempty" json:"restrictedNamespaces,omitempty"`

	// AllowedNamespaces — allow-list applied to every CRD in this Katalog.
	// Merged additively with per-CRD allowedNamespaces.
	AllowedNamespaces AllowedNamespaces `yaml:"allowedNamespaces,omitempty" json:"allowedNamespaces,omitempty"`
}

// WebhooksConfig controls global admission webhook settings.
// Conversion is declared separately under security.conversion.
type WebhooksConfig struct {
	// Admission controls the ValidatingWebhookConfiguration and MutatingWebhookConfiguration.
	Admission *AdmissionWebhookToggle `yaml:"admission,omitempty"`

	// FailurePolicy controls what the API server does when Orkestra is unreachable
	// for admission calls. "Fail" or "Ignore".
	// Default: WEBHOOKS_FAILURE_POLICY env / "Ignore".
	FailurePolicy string `yaml:"failurePolicy,omitempty"`

	// ServiceName is the Kubernetes Service fronting Orkestra's HTTPS server.
	// Shared with deletion protection when both are enabled.
	// Default: ORKESTRA_SERVICE_NAME env / "orkestra".
	ServiceName string `yaml:"serviceName,omitempty"`

	// CleanupOnShutdown controls whether Admission webhook is deleted on graceful shutdown.
	// Default: false — Admission webhook persists across restarts.
	CleanupOnShutdown bool `yaml:"cleanupOnShutdown,omitempty"`
}

// AdmissionWebhookToggle controls whether admission webhooks are globally enabled.
type AdmissionWebhookToggle struct {
	// Enabled controls whether ValidatingWebhookConfiguration and
	// MutatingWebhookConfiguration are registered at startup.
	// Default: ENABLE_ADMISSION_WEBHOOK env / false.
	Enabled *bool `yaml:"enabled,omitempty"`
}

// ConversionConfig controls the /convert endpoint and CRD version conversion.
type ConversionConfig struct {
	// Enabled controls whether the /convert endpoint is registered and the
	// CRD conversion webhook is active.
	// Default: ENABLE_CONVERSION env / false.
	Enabled *bool `yaml:"enabled,omitempty"`

	// ConversionWindow is the rolling window size for latency/throughput stats.
	// Default: CONVERSION_WINDOW env / 100.
	ConversionWindow int `yaml:"conversionWindow,omitempty"`
}

// RBACConfig controls RBAC auto-apply behaviour.
type RBACConfig struct {
	// Enabled controls whether RBAC is applied at startup.
	// Default: true when the rbac block is declared.
	Enabled *bool `yaml:"enabled,omitempty"`

	// CleanupOnShutdown controls whether RBAC is deleted on graceful shutdown.
	// Default: false — RBAC persists across restarts.
	CleanupOnShutdown bool `yaml:"cleanupOnShutdown,omitempty"`
}

// ── Effective value helpers ───────────────────────────────────────────────────

// IsDeletionProtectionEnabled returns the effective deletion protection setting.
func (s *KatalogSecurity) IsDeletionProtectionEnabled() bool {
	if s == nil || s.DeletionProtection == nil {
		return false
	}
	if s.DeletionProtection.Enabled == nil {
		return true // declared but no explicit value = enabled
	}
	return *s.DeletionProtection.Enabled
}

// IsAdmissionEnabled returns true when admission webhooks are globally enabled.
func (s *KatalogSecurity) IsAdmissionEnabled() bool {
	if s == nil || s.Webhooks == nil || s.Webhooks.Admission == nil {
		return false
	}
	if s.Webhooks.Admission.Enabled == nil {
		return false // no default-on for webhooks — must be explicit
	}
	return *s.Webhooks.Admission.Enabled
}

// IsConversionEnabled returns true when the conversion webhook is globally enabled.
func (s *KatalogSecurity) IsConversionEnabled() bool {
	if s == nil || s.Conversion == nil {
		return false
	}
	if s.Conversion.Enabled == nil {
		return false
	}
	return *s.Conversion.Enabled
}

// DeletionProtectionServiceName returns the effective service name for deletion protection.
// Falls back to the provided ENV default.
func (s *KatalogSecurity) DeletionProtectionServiceName(envDefault string) string {
	if s != nil && s.DeletionProtection != nil && s.DeletionProtection.ServiceName != "" {
		return s.DeletionProtection.ServiceName
	}
	return envDefault
}

// DeletionProtectionFailurePolicy returns the effective failure policy string.
// Falls back to "Fail" when not configured — protecting by default.
func (s *KatalogSecurity) DeletionProtectionFailurePolicy() string {
	if s != nil && s.DeletionProtection != nil && s.DeletionProtection.FailurePolicy != "" {
		return s.DeletionProtection.FailurePolicy
	}
	return "Fail"
}

// IsNamespaceProtectionEnabled returns the effective namespace protection setting.
func (s *KatalogSecurity) IsNamespaceProtectionEnabled() bool {
	if s == nil || s.NamespaceProtection == nil {
		return false
	}
	if s.NamespaceProtection.Enabled == nil {
		return true // declared but no explicit value = enabled
	}
	return *s.NamespaceProtection.Enabled
}

// NamespaceProtectionServiceName returns the effective service name for namespace protection.
// Falls back to the provided ENV default.
func (s *KatalogSecurity) NamespaceProtectionServiceName(envDefault string) string {
	if s != nil && s.NamespaceProtection != nil && s.NamespaceProtection.ServiceName != "" {
		return s.NamespaceProtection.ServiceName
	}
	return envDefault
}

// NamespaceProtectionFailurePolicy returns the effective failure policy string.
// Falls back to "Fail" when not configured — protecting by default.
func (s *KatalogSecurity) NamespaceProtectionFailurePolicy() string {
	if s != nil && s.NamespaceProtection != nil && s.NamespaceProtection.FailurePolicy != "" {
		return s.NamespaceProtection.FailurePolicy
	}
	return "Fail"
}

// WebhooksServiceName returns the effective service name for admission webhooks.
// Falls back to the provided ENV default.
func (s *KatalogSecurity) WebhooksServiceName(envDefault string) string {
	if s != nil && s.Webhooks != nil && s.Webhooks.ServiceName != "" {
		return s.Webhooks.ServiceName
	}
	return envDefault
}

// WebhooksFailurePolicy returns the effective failure policy for admission webhooks.
// Falls back to "Ignore" — not blocking when Orkestra is unreachable.
func (s *KatalogSecurity) WebhooksFailurePolicy(envDefault string) string {
	if s != nil && s.Webhooks != nil && s.Webhooks.FailurePolicy != "" {
		return s.Webhooks.FailurePolicy
	}
	if envDefault != "" {
		return envDefault
	}
	return "Ignore"
}
