package types

import "strings"

// ServeTarget returns the identifier callers use to reference this CRD in the
// Gateway API and schema API.
//
// Resolution order:
//  1. serve.target if explicitly set — platform team's chosen name.
//  2. Lowercased kind — "App" → "app", "DatabaseCluster" → "databasecluster".
//
// The default keeps things working with no config change. Platform teams set
// an explicit target when the lowercased kind is too verbose or ambiguous.
func (c *CRDEntry) ServeTarget() string {
	if c.Serve != nil && c.Serve.Target != "" {
		return c.Serve.Target
	}
	return strings.ToLower(c.APITypes.Kind)
}

// HasServeTarget reports whether this CRD can be addressed by target.
// Requires serve to be enabled and a kind to be declared.
func (c *CRDEntry) HasServeTarget() bool {
	return c.ServeEnabled() && c.APITypes.Kind != ""
}

// ServeTargetOrEmpty returns ServeTarget when HasServeTarget is true, "" otherwise.
// For API responses that expose target as an omitempty field to callers that
// shouldn't see a target for a CRD they can't actually address by one.
func (c *CRDEntry) ServeTargetOrEmpty() string {
	if !c.HasServeTarget() {
		return ""
	}
	return c.ServeTarget()
}

// AllServeFields returns a flat map of every serve-declared field — spec fields from
// serve.fields, label fields from serve.labels, and annotation
// fields from serve.annotations.
//
// Used by the schema API to expose the complete field contract to callers and
// by BuildCRFromTarget to route each submitted value to the right CR path.
//
// Fields from the same key in multiple sources are merged in this order:
// spec fields → label fields → annotation fields. In practice, the platform
// team should not declare the same key in more than one source; ork validate
// catches this.
func (c *CRDEntry) AllServeFields() map[string]ServeFieldConfig {
	if c.Serve == nil {
		return nil
	}
	out := make(map[string]ServeFieldConfig, len(c.Serve.Fields))

	// Spec fields — declared under serve.fields.
	for name, cfg := range c.Serve.Fields {
		out[name] = cfg
	}

	// Label fields — declared under serve.labels.
	for name, cfg := range c.ServeLabels() {
		out[name] = cfg
	}

	// Annotation fields — declared under serve.annotations.
	for name, cfg := range c.ServeAnnotations() {
		out[name] = cfg
	}

	return out
}
