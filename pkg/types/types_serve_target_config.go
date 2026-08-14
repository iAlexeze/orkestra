package types

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ServeTargetValue is the value of serve.target on a CRD entry.
//
// Two YAML forms are accepted:
//
//	# Scalar — shorthand for a primary-only target with no per-entry config.
//	serve:
//	  target: apifixture
//
//	# Map — one or more named entries; exactly one must have primary: true.
//	serve:
//	  target:
//	    apifixture:
//	      primary: true
//	      tokens: { ... }
//	    preview:
//	      tokens: { ... }
//
// The scalar form expands to {<name>: {Primary: true}} at load time.
// After expansion, Entries is always the canonical form.
type ServeTargetValue struct {
	// Shorthand holds the scalar string before load-time expansion.
	// After expansion this is always empty; Entries is the canonical form.
	Shorthand string

	// Entries is the map of named target/alias entries.
	// Populated directly from the map form, or synthesised from Shorthand at expansion.
	Entries map[string]*ServeTargetConfig
}

// IsZero reports whether no target has been set at all.
func (s ServeTargetValue) IsZero() bool {
	return s.Shorthand == "" && len(s.Entries) == 0
}

// UnmarshalYAML accepts both scalar and map forms.
func (s *ServeTargetValue) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		s.Shorthand = value.Value
		return nil
	case yaml.MappingNode:
		return value.Decode(&s.Entries)
	default:
		return fmt.Errorf("serve.target must be a string or a map")
	}
}

// MarshalYAML serialises as a plain string when there is exactly one entry
// that is primary with no additional config (shorthand round-trip).
func (s ServeTargetValue) MarshalYAML() (interface{}, error) {
	if s.Shorthand != "" {
		return s.Shorthand, nil
	}
	if len(s.Entries) == 1 {
		for name, cfg := range s.Entries {
			if cfg != nil && cfg.Primary && cfg.isDefaultPrimary() {
				return name, nil
			}
		}
	}
	return s.Entries, nil
}

// UnmarshalJSON accepts both string and object forms.
func (s *ServeTargetValue) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		s.Shorthand = str
		return nil
	}
	return json.Unmarshal(data, &s.Entries)
}

// MarshalJSON serialises as a plain string for the shorthand case.
func (s ServeTargetValue) MarshalJSON() ([]byte, error) {
	if s.Shorthand != "" {
		return json.Marshal(s.Shorthand)
	}
	if len(s.Entries) == 1 {
		for name, cfg := range s.Entries {
			if cfg != nil && cfg.Primary && cfg.isDefaultPrimary() {
				return json.Marshal(name)
			}
		}
	}
	return json.Marshal(s.Entries)
}

// ServeTargetConfig is one entry in the serve.target map.
// Used for both the primary entry (Primary: true) and aliases (Primary: false).
//
// The primary entry is identified by Primary: true. Exactly one entry per CRD
// must carry this flag — enforced by ork validate. The primary entry acts as
// config authority (source of CRD-level defaults for tokens and response) even
// when its Enabled flag is false (surface disabled but config still applies).
//
// Alias entries (Primary: false / omitted) override CRD-level defaults
// independently. A token not listed in an alias Tokens map is denied for that
// alias even when it is allowed at the CRD level.
type ServeTargetConfig struct {
	// Primary marks this as the primary target entry.
	// Exactly one entry in the map must be true.
	// Validated by ork validate — not enforced at parse time.
	Primary bool `yaml:"primary,omitempty" json:"primary,omitempty"`

	// Enabled controls whether this surface is reachable by callers.
	// nil and true are equivalent — the surface is active.
	// false hides the entry from callers, the schema catalog, and lookups.
	// The primary's config authority role is unaffected by Enabled.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Apply configures apply-time behaviour for this specific target.
	Apply *ServeApplyConfig `yaml:"apply,omitempty" json:"apply,omitempty"`

	// Modes controls which apply modes are allowed for this target.
	Modes *ServeModes `yaml:"modes,omitempty" json:"modes,omitempty"`

	// Include is a path (relative to the katalog file) to a YAML file with
	// tokens: and/or config: keys. Inline fields take precedence on merge.
	Include string `yaml:"include,omitempty" json:"include,omitempty"`

	// Tokens restricts which gateway tokens may access this surface and with
	// what permissions. When absent, falls back to serve.tokens (CRD default).
	Tokens map[string]ServeTokenPermissions `yaml:"tokens,omitempty" json:"tokens,omitempty"`

	// Config controls the shape of apply and GET responses for this surface.
	// When absent, falls back to serve.config (CRD default).
	Config *ServeAliasConfigSettings `yaml:"config,omitempty" json:"config,omitempty"`

	// Clusters scopes the fan-out for this target to a subset of serve.clusters.
	// Each entry is either a static name or a template expression resolved at apply time.
	// When absent, fan-out goes to all of serve.clusters.
	// Each name must appear in serve.clusters (validated by ork validate).
	Clusters []string `yaml:"clusters,omitempty" json:"clusters,omitempty"`
}

// IsEnabled reports whether this surface is reachable by callers.
// nil Enabled or Enabled=true → active. Enabled=false → disabled.
func (t *ServeTargetConfig) IsEnabled() bool {
	if t == nil || t.Enabled == nil {
		return true
	}
	return *t.Enabled
}

// HasTokenRestrictions reports whether this entry declares its own token restrictions.
func (t *ServeTargetConfig) HasTokenRestrictions() bool {
	if t == nil {
		return false
	}
	return len(t.Tokens) > 0
}

// ResponseConfig returns the response config for this entry, or nil.
func (t *ServeTargetConfig) ResponseConfig() *ServeResponseConfig {
	if t == nil || t.Config == nil {
		return nil
	}
	return t.Config.Response
}

// HasClusters reports whether this target entry declares its own cluster routing.
func (t *ServeTargetConfig) HasClusters() bool {
	return t != nil && len(t.Clusters) > 0
}

// TargetClusters returns the cluster list for this target entry, or nil.
func (t *ServeTargetConfig) TargetClusters() []string {
	if t == nil {
		return nil
	}
	return t.Clusters
}

// isDefaultPrimary reports whether this entry has no config beyond Primary: true
// — used to decide whether the YAML shorthand round-trip is safe.
func (t *ServeTargetConfig) isDefaultPrimary() bool {
	if t == nil {
		return true
	}
	return t.IsEnabled() && len(t.Tokens) == 0 && t.Config == nil && t.Include == "" && !t.HasClusters()
}
