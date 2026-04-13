// pkg/types/security.go
//
// Security configuration at the Katalog level.
//
// YAML:
//
//	security:
//	  deletionProtection:
//	    enabled: true    # default: true when block is present
//	  rbac:
//	    enabled: true    # default: true when block is present
//	    cleanupOnShutdown: false
package types

// KatalogSecurity holds the security configuration for a Katalog.
type KatalogSecurity struct {
	// DeletionProtection controls whether Orkestra registers a webhook
	// that blocks deletion of its managed CRDs and its own deployment.
	//
	// When enabled (default when block is present):
	//   - Registers /deletion-protection endpoint on the HTTPS server
	//   - Creates a ValidatingWebhookConfiguration with failurePolicy: Fail
	//   - Intercepts DELETE on apiextensions.k8s.io/v1 customresourcedefinitions
	//   - Intercepts DELETE on apps/v1 deployments (Orkestra deployment only)
	//
	// To decommission an operator with deletion protection:
	//   1. Set enabled: false
	//   2. Redeploy Orkestra (webhook is removed on startup)
	//   3. Delete CRDs normally
	//
	// nil pointer: deletion protection is disabled (not declared in YAML).
	// *false: explicitly disabled.
	// *true: enabled.
	DeletionProtection *DeletionProtectionConfig `yaml:"deletionProtection,omitempty"`

	// RBAC controls whether Orkestra generates and applies RBAC resources
	// (ClusterRole, ClusterRoleBinding, ServiceAccount) at startup.
	//
	// When enabled: applies least-privilege RBAC using server-side apply.
	// Idempotent — safe to run on every startup.
	//
	// CleanupOnShutdown: false (default) — RBAC persists across restarts.
	// CleanupOnShutdown: true — RBAC is deleted on graceful shutdown.
	// Useful for test environments and ephemeral operators.
	RBAC *RBACConfig `yaml:"rbac,omitempty"`
}

// DeletionProtectionConfig controls deletion protection behaviour.
type DeletionProtectionConfig struct {
	// Enabled controls whether deletion protection is active.
	// Default: true when the deletionProtection block is declared.
	Enabled *bool `yaml:"enabled,omitempty"`
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

// IsDeletionProtectionEnabled returns the effective deletion protection setting.
// Consistent with the method on Katalog — usable without a Katalog reference.
func (s *KatalogSecurity) IsDeletionProtectionEnabled() bool {
	if s == nil || s.DeletionProtection == nil {
		return false // not declared = not enabled
	}
	if s.DeletionProtection.Enabled == nil {
		return true // declared but no explicit value = enabled
	}
	return *s.DeletionProtection.Enabled
}

// IsRBACEnabled returns the effective RBAC setting.
func (s *KatalogSecurity) IsRBACEnabled() bool {
	if s == nil || s.RBAC == nil {
		return false
	}
	if s.RBAC.Enabled == nil {
		return true
	}
	return *s.RBAC.Enabled
}
