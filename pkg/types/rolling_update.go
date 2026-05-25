package types

// RollingUpdateBehavior configures a Deployment's rolling update strategy.
// Set Profile for a named preset, or MaxSurge/MaxUnavailable explicitly.
// Profile and explicit fields are mutually exclusive.
type RollingUpdateBehavior struct {
	// Profile — named preset. Expands into MaxSurge and MaxUnavailable at load time.
	// Allowed: safe, fast, blue-green.
	Profile string `yaml:"profile,omitempty" json:"profile,omitempty"`

	// MaxSurge — maximum pods that can be scheduled above the desired replica count
	// during a rolling update. Accepts integer string ("1") or percentage string ("25%").
	MaxSurge string `yaml:"maxSurge,omitempty" json:"maxSurge,omitempty"`

	// MaxUnavailable — maximum pods that can be unavailable during a rolling update.
	// Accepts integer string ("0") or percentage string ("25%").
	MaxUnavailable string `yaml:"maxUnavailable,omitempty" json:"maxUnavailable,omitempty"`
}
