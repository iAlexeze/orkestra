package types

import "fmt"

// RequiredIDPFieldRules synthesizes an implicit exists validation rule for
// every idp.fields / idp.additionalFields entry marked required: true.
//
// This makes "required" actually required, for every client of the Apply
// API — curl, CI, a custom UI, the Control Center form — not just the one
// that happens to render an HTML `required` attribute. Without synthesis, a
// katalog author would have to separately hand-write a matching
// validation.rules entry, easy to forget, and easy for the message text to
// drift from the field's label since the two are declared in different
// places. Synthesizing the rule here keeps the message matching the label
// automatically.
//
// The synthesized rule inherits the same When/AnyOf already declared on the
// field for form visibility. This is what makes required: true correct for a
// discriminator-routed CRD (one CRD, a workloadType-style field selecting
// between several shapes): a field only visible when workloadType: app
// becomes required only under that same condition, not unconditionally — a
// plain CRD schema's static `required: [...]` list has no way to express
// that, since it can't vary by another field's value.
//
// additionalFields entries use the getLabel/getAnnotation notes rather than
// a raw "metadata.labels.<name>" dot-path: an annotation key following the
// Kubernetes-recommended prefix/name shape contains dots itself, which the
// dot-path resolver would misparse as extra path segments.
func (c *CRDEntry) RequiredIDPFieldRules() []ValidationRule {
	if c == nil || c.IDP == nil {
		return nil
	}

	var rules []ValidationRule

	for name, cfg := range c.IDP.Fields {
		if !cfg.Required {
			continue
		}
		rules = append(rules, requiredRule("spec."+name, requiredLabel(cfg, name), cfg))
	}
	for name, cfg := range c.AdditionalLabelFields() {
		if !cfg.Required {
			continue
		}
		rules = append(rules, requiredRule(fmt.Sprintf(`{{ getLabel . %q }}`, name), requiredLabel(cfg, name), cfg))
	}
	for name, cfg := range c.AdditionalAnnotationFields() {
		if !cfg.Required {
			continue
		}
		rules = append(rules, requiredRule(fmt.Sprintf(`{{ getAnnotation . %q }}`, name), requiredLabel(cfg, name), cfg))
	}

	return rules
}

func requiredLabel(cfg IDPFieldConfig, name string) string {
	if cfg.Label != "" {
		return cfg.Label
	}
	return name
}

func requiredRule(field, label string, cfg IDPFieldConfig) ValidationRule {
	return ValidationRule{
		Field:    field,
		Operator: ConditionExists,
		Message:  label + " is required",
		Action:   ValidationActionDeny,
		When:     cfg.When,
		AnyOf:    cfg.AnyOf,
	}
}
