package types

// DeletionProtectionOverride controls per‑CRD behaviour when the global
// security.deletionProtection.enabled is true.
// Both fields default to true when omitted.
type DeletionProtectionOverride struct {
	// ProtectCRD determines whether the CRD definition itself (the type)
	// is protected from deletion. Default true.
	ProtectCRD *bool `yaml:"protectCRD,omitempty" json:"protectCRD,omitempty"`

	// ProtectCRs determines whether instances of this CRD are protected
	// from deletion (via the orkestra.io/deletion-protection label).
	// Default true.
	ProtectCRs *bool `yaml:"protectCRs,omitempty" json:"protectCRs,omitempty"`

	// StrictMode controls whether removing the deletion-protection label from a resource
	// is itself treated as a deletion attempt and blocked.
	// When true, the only way to remove the label (and thus unprotect a resource) is to
	// disable strictMode in the Katalog and restart Orkestra Gateway.
	// Default: katalog level strictMode.
	StrictMode *bool `yaml:"strictMode,omitempty" json:"strictMode,omitempty"`
}

// HasDeletionProtectionOverride reports whether deletion protection override is set for this CRD.
func (c *CRDEntry) HasDeletionProtectionOverride() bool {
	return c.DeletionProtection != nil
}

// ShouldProtectCRD reports whether the CRD *type definition* itself should be
// protected from deletion when global deletion protection is enabled.
//
// Semantics:
//   - When DeletionProtection is nil → default to true
//   - When ProtectCRD is nil         → default to true
//   - When ProtectCRD is false       → CRD deletion is allowed
//
// This controls whether DELETE operations on the CRD object
// (apiextensions.k8s.io/v1/customresourcedefinitions/<name>) are intercepted
// and blocked by the deletion‑protection webhook.
func (c *CRDEntry) ShouldProtectCRD() bool {
	if c.DeletionProtection == nil || c.DeletionProtection.ProtectCRD == nil {
		return true
	}
	return *c.DeletionProtection.ProtectCRD
}

// ShouldProtectCRs reports whether *instances* of this CRD should be protected
// from deletion when global deletion protection is enabled.
//
// Semantics:
//   - When DeletionProtection is nil → default to true
//   - When ProtectCRs is nil         → default to true
//   - When ProtectCRs is false       → CR deletion is allowed
//
// This controls whether DELETE operations on CR instances are blocked unless
// the object explicitly opts out via the orkestra.io/deletion-protection label.
func (c *CRDEntry) ShouldProtectCRs() bool {
	if c.DeletionProtection == nil || c.DeletionProtection.ProtectCRs == nil {
		return true
	}
	return *c.DeletionProtection.ProtectCRs
}

// IsStrictDeletionProtection returns whether strict deletion‑protection semantics
// apply to this CRD, taking into account the katalog‑level strict mode.
//
// Strict mode is only possible when the katalog‑level strictMode is true.
// If the katalog‑level strictMode is false, this function always returns false.
//
// When katalog‑level strictMode is true:
//   - If the CRD has its own StrictMode value (non‑nil), that value is returned.
//   - Otherwise, default to true.
//
// This override only applies when global deletion protection is enabled
// (security.deletionProtection.enabled = true).
func (c *CRDEntry) IsStrictDeletionProtection(katalogStrictMode bool) bool {
	if !katalogStrictMode {
		return false
	}
	if c.DeletionProtection == nil || c.DeletionProtection.StrictMode == nil {
		return true
	}
	return *c.DeletionProtection.StrictMode
}
