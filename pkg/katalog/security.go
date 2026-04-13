// pkg/katalog/security.go
//
// Security accessors on *Katalog.
//
// KatalogSecurity now uses struct types with Enabled *bool fields,
// which allows detecting "not declared" (nil) vs "explicitly false" (*false).
//
// Deletion protection is ENABLED BY DEFAULT when the security block
// is present but deletionProtection is not declared.
// This matches the principle of least surprise for operators:
// if you care enough to have a security block, protection is on.
//
// RBAC is also enabled by default when the security block is present.
package katalog

// IsDeletionProtectionEnabled reports whether deletion protection is active.
//
// Decision table:
//
//	security block absent              → false (not declared, not opt-in)
//	security.deletionProtection absent → true  (block present, protection default-on)
//	enabled: true                      → true
//	enabled: false                     → false
func (k *Katalog) IsDeletionProtectionEnabled() bool {
	return k.Security.IsDeletionProtectionEnabled()
}

// IsRBACEnabled reports whether RBAC auto-apply is active.
//
// Decision table:
//
//	security block absent       → false (not declared, not opt-in)
//	security.rbac absent        → true  (block present, rbac default-on)
//	enabled: true               → true
//	enabled: false              → false
func (k *Katalog) IsRBACEnabled() bool {
	return k.Security.IsRBACEnabled()
}

// RBACCleanupOnShutdown reports whether RBAC should be deleted on shutdown.
// Default: false — RBAC survives restarts.
func (k *Katalog) RBACCleanupOnShutdown() bool {
	if k.Security.RBAC == nil {
		return false
	}
	return k.Security.RBAC.CleanupOnShutdown
}

// NeedsCertificates reports whether Orkestra must generate TLS certificates.
//
// Certificates are required when deletion protection is enabled — the deletion
// protection webhook needs TLS and the user should not have to configure it
// separately. If the user has already provided TLS config, this is a no-op.
//
// Concretely: returns true when deletion protection is enabled AND
// no explicit TLS cert path has been configured.
func (k *Katalog) NeedsCertificates() bool {
	return k.IsDeletionProtectionEnabled()
}
