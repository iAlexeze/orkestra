package types

import "strings"

// IDPTarget returns the identifier callers use to reference this CRD in the
// Apply API and schema API.
//
// Resolution order:
//  1. idp.target if explicitly set — platform team's chosen name.
//  2. Lowercased kind — "App" → "app", "DatabaseCluster" → "databasecluster".
//
// The default keeps things working with no config change. Platform teams set
// an explicit target when the lowercased kind is too verbose or ambiguous.
func (c *CRDEntry) IDPTarget() string {
	if c.IDP != nil && c.IDP.Target != "" {
		return c.IDP.Target
	}
	return strings.ToLower(c.APITypes.Kind)
}

// HasIDPTarget reports whether this CRD can be addressed by target.
// Requires IDP to be enabled and a kind to be declared.
func (c *CRDEntry) HasIDPTarget() bool {
	return c.IDPEnabled() && c.APITypes.Kind != ""
}

// IDPFields returns a flat map of every IDP-declared field — spec fields from
// idp.fields, label fields from idp.additionalFields.labels, and annotation
// fields from idp.additionalFields.annotations.
//
// Used by the schema API to expose the complete field contract to callers and
// by BuildCRFromTarget to route each submitted value to the right CR path.
//
// Fields from the same key in multiple sources are merged in this order:
// spec fields → label fields → annotation fields. In practice, the platform
// team should not declare the same key in more than one source; ork validate
// catches this.
func (c *CRDEntry) IDPFields() map[string]IDPFieldConfig {
	if c.IDP == nil {
		return nil
	}
	out := make(map[string]IDPFieldConfig, len(c.IDP.Fields))

	// Spec fields — declared under idp.fields.
	for name, cfg := range c.IDP.Fields {
		out[name] = cfg
	}

	// Label fields — declared under idp.additionalFields.labels.
	for name, cfg := range c.AdditionalLabelFields() {
		out[name] = cfg
	}

	// Annotation fields — declared under idp.additionalFields.annotations.
	for name, cfg := range c.AdditionalAnnotationFields() {
		out[name] = cfg
	}

	return out
}
