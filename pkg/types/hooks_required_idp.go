package types

import (
	"fmt"
	"strings"
)

// idpFieldRef is one idp.fields / idp.additionalFields entry, resolved to
// the field expression a synthesized validation rule would target.
type idpFieldRef struct {
	field string // "spec.<name>" dot-path, or a getLabel/getAnnotation template
	label string
	cfg   IDPFieldConfig
}

// allIDPFieldRefs walks idp.fields, idp.additionalFields.labels, and
// idp.additionalFields.annotations, yielding one idpFieldRef per entry —
// the shared source both RequiredIDPFieldRules and EnumIDPFieldRules
// synthesize from.
//
// additionalFields entries resolve through the getLabel/getAnnotation notes
// rather than a raw "metadata.labels.<name>" dot-path: an annotation key
// following the Kubernetes-recommended prefix/name shape contains dots
// itself, which the dot-path resolver would misparse as extra path segments.
func (c *CRDEntry) allIDPFieldRefs() []idpFieldRef {
	if c == nil || c.IDP == nil {
		return nil
	}

	var refs []idpFieldRef
	for name, cfg := range c.IDP.Fields {
		refs = append(refs, idpFieldRef{"spec." + name, idpFieldLabel(cfg, name), cfg})
	}
	for name, cfg := range c.AdditionalLabelFields() {
		refs = append(refs, idpFieldRef{fmt.Sprintf(`{{ getLabel . %q }}`, name), idpFieldLabel(cfg, name), cfg})
	}
	for name, cfg := range c.AdditionalAnnotationFields() {
		refs = append(refs, idpFieldRef{fmt.Sprintf(`{{ getAnnotation . %q }}`, name), idpFieldLabel(cfg, name), cfg})
	}
	return refs
}

func idpFieldLabel(cfg IDPFieldConfig, name string) string {
	if cfg.Label != "" {
		return cfg.Label
	}
	return name
}

// RequiredIDPFieldRules synthesizes an implicit exists validation rule for
// every idp.fields / idp.additionalFields entry marked required: true —
// enforced server-side, for every Apply API client, not just the Control
// Center form. See "Required fields are enforced automatically" at
// https://orkestra.sh/docs/reference/schema/katalog/validation#required-fields-are-enforced-automatically
// for the full rationale, including why inheriting the field's own
// When/AnyOf matters for discriminator-routed CRDs.
func (c *CRDEntry) RequiredIDPFieldRules() []ValidationRule {
	var rules []ValidationRule
	for _, ref := range c.allIDPFieldRefs() {
		if !ref.cfg.Required {
			continue
		}
		rules = append(rules, ValidationRule{
			Field:    ref.field,
			Operator: ConditionExists,
			Message:  ref.label + " is required",
			Action:   ValidationActionDeny,
			When:     ref.cfg.When,
			AnyOf:    ref.cfg.AnyOf,
		})
	}
	return rules
}

// EnumIDPFieldRules synthesizes an implicit `in` validation rule for every
// idp.fields / idp.additionalFields entry declaring type: enum with a
// non-empty enum list. See "Enum fields are validated automatically" at
// https://orkestra.sh/docs/reference/schema/katalog/validation#enum-fields-are-validated-automatically for the full
// rationale.
//
// The exists gate is always added regardless of required: membership and
// presence are independent — RequiredIDPFieldRules alone decides whether a
// field is mandatory, this only decides whether its value (if any) is
// valid.
func (c *CRDEntry) EnumIDPFieldRules() []ValidationRule {
	var rules []ValidationRule
	for _, ref := range c.allIDPFieldRefs() {
		cfgType := strings.ToLower(ref.cfg.Type)
		if cfgType != "enum" || len(ref.cfg.Enum) == 0 {
			continue
		}
		when := append([]Condition{{Field: ref.field, Operator: ConditionExists}}, ref.cfg.When...)
		rules = append(rules, ValidationRule{
			Field:    ref.field,
			Operator: ConditionIn,
			Value:    strings.Join(ref.cfg.Enum, ","),
			Message:  ref.label + " must be one of: " + strings.Join(ref.cfg.Enum, ", "),
			Action:   ValidationActionDeny,
			When:     when,
			AnyOf:    ref.cfg.AnyOf,
		})
	}
	return rules
}
