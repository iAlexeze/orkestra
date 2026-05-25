package types

// PDBBehavior configures PodDisruptionBudget disruption limits.
// Set Profile for a named preset, or MinAvailable/MaxUnavailable explicitly.
// Profile and explicit fields are mutually exclusive.
type PDBBehavior struct {
	// Profile — named preset. Expands into MinAvailable or MaxUnavailable at load time.
	// Allowed: zero-downtime, rolling, relaxed.
	Profile string `yaml:"profile,omitempty" json:"profile,omitempty"`

	// MinAvailable — minimum pods that must remain available during voluntary disruptions.
	// Accepts integer string ("1") or percentage string ("100%").
	// Mutually exclusive with MaxUnavailable.
	MinAvailable string `yaml:"minAvailable,omitempty" json:"minAvailable,omitempty"`

	// MaxUnavailable — maximum pods that may be unavailable during voluntary disruptions.
	// Accepts integer string ("1") or percentage string ("25%").
	// Mutually exclusive with MinAvailable.
	MaxUnavailable string `yaml:"maxUnavailable,omitempty" json:"maxUnavailable,omitempty"`
}
