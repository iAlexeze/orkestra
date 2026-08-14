package types

// ServeApplyOverrides configures override behaviours for apply operations.
type ServeApplyOverrides struct {
	// TargetConflict, when true, allows callers to change the target/alias
	// of an existing CR via the ?override=true query parameter.
	// Default: false — target/alias is immutable after creation.
	TargetConflict *bool `yaml:"targetConflict,omitempty" json:"targetConflict,omitempty"`

	// ResourceConflict, when true, sets Force: true on every server-side apply
	// for this CRD — the gateway takes ownership of any conflicting fields
	// rather than surfacing a conflict error. Equivalent to helm --force-conflict.
	// Can be overridden per-request with ?overwrite=true regardless of this setting.
	// Default: false.
	ResourceConflict *bool `yaml:"resourceConflict,omitempty" json:"resourceConflict,omitempty"`
}

// ServeApplyConfig configures apply-time behaviour for a CRD or target.
type ServeApplyConfig struct {
	// Overrides contains override settings for apply operations.
	Overrides *ServeApplyOverrides `yaml:"overrides,omitempty" json:"overrides,omitempty"`
}

// ─── HasOverride methods ─────────────────────────────────────────────────────

// HasOverride reports whether the ServeConfig has any override fields set.
// Used to avoid unnecessary config blocks in the Katalog.
func (s *ServeConfig) HasOverride() bool {
	if s == nil || s.Apply == nil || s.Apply.Overrides == nil {
		return false
	}
	return s.Apply.Overrides.TargetConflict != nil || s.Apply.Overrides.ResourceConflict != nil
}

// HasOverride reports whether the ServeTargetConfig has any override fields set.
func (t *ServeTargetConfig) HasOverride() bool {
	if t == nil || t.Apply == nil || t.Apply.Overrides == nil {
		return false
	}
	return t.Apply.Overrides.TargetConflict != nil || t.Apply.Overrides.ResourceConflict != nil
}

// HasOverride reports whether the CRDEntry has any override fields set
// at the CRD level or on any target.
func (c *CRDEntry) HasOverride() bool {
	if !c.ServeEnabled() {
		return false
	}

	// Check CRD-level override
	if c.Serve.HasOverride() {
		return true
	}

	// Check any target-level override
	if c.Serve.Target.Entries != nil {
		for _, cfg := range c.Serve.Target.Entries {
			if cfg.HasOverride() {
				return true
			}
		}
	}

	return false
}

// HasTargetOverride reports whether the CRDEntry has any targetConflict set
// at the CRD level or on any target.
func (c *CRDEntry) HasTargetOverride() bool {
	if !c.ServeEnabled() {
		return false
	}

	if c.Serve.Apply != nil && c.Serve.Apply.Overrides != nil && c.Serve.Apply.Overrides.TargetConflict != nil {
		return true
	}

	if c.Serve.Target.Entries != nil {
		for _, cfg := range c.Serve.Target.Entries {
			if cfg.Apply != nil && cfg.Apply.Overrides != nil && cfg.Apply.Overrides.TargetConflict != nil {
				return true
			}
		}
	}

	return false
}

// HasResourceConflict reports whether the CRDEntry has any resourceConflict set
// at the CRD level or on any target.
func (c *CRDEntry) HasResourceConflict() bool {
	if !c.ServeEnabled() {
		return false
	}

	if c.Serve.Apply != nil && c.Serve.Apply.Overrides != nil && c.Serve.Apply.Overrides.ResourceConflict != nil {
		return true
	}

	if c.Serve.Target.Entries != nil {
		for _, cfg := range c.Serve.Target.Entries {
			if cfg.Apply != nil && cfg.Apply.Overrides != nil && cfg.Apply.Overrides.ResourceConflict != nil {
				return true
			}
		}
	}

	return false
}
