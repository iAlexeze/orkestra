package types

import (
	"fmt"
	"sort"
	"strings"
)

// idpFieldRef is one idp.fields / idp.additionalFields entry, resolved to
// the field expression a synthesized validation rule would target.
type idpFieldRef struct {
	name  string // the plain idp.fields / additionalFields key, for sorting
	field string // "spec.<name>" dot-path, or a getLabel/getAnnotation template
	link  string // same as name, for ValidationRule.Link — set only when field isn't already "spec.<name>"
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
// That template expression isn't a valid display name either, which is what
// idpFieldRef.link is for — see ValidationRule.Link.
//
// Returned sorted by cfg.Order (0 = unset, sorts last), then name — the same
// rule the Control Center uses to arrange the rendered form
// (buildIDPFormFields). idp.fields/additionalFields are Go maps, so without
// this the synthesis order — and therefore which rule's violation lands in
// Violations[0] when several required fields are missing at once — would be
// nondeterministic across reconciles. Sorting to match the form means the
// field a developer sees first is also the one whose violation wins.
func (c *CRDEntry) allIDPFieldRefs() []idpFieldRef {
	if c == nil || c.IDP == nil {
		return nil
	}

	var refs []idpFieldRef
	for name, cfg := range c.IDP.Fields {
		// "spec." + name is already a clean display name on its own —
		// no link needed, and setting one would trip the "redundant link"
		// validation error.
		refs = append(refs, idpFieldRef{name: name, field: "spec." + name, label: idpFieldLabel(cfg, name), cfg: cfg})
	}
	for name, cfg := range c.AdditionalLabelFields() {
		refs = append(refs, idpFieldRef{name: name, field: fmt.Sprintf(`{{ getLabel . %q }}`, name), link: name, label: idpFieldLabel(cfg, name), cfg: cfg})
	}
	for name, cfg := range c.AdditionalAnnotationFields() {
		refs = append(refs, idpFieldRef{name: name, field: fmt.Sprintf(`{{ getAnnotation . %q }}`, name), link: name, label: idpFieldLabel(cfg, name), cfg: cfg})
	}

	sort.Slice(refs, func(i, j int) bool {
		oi, oj := refs[i].cfg.Order, refs[j].cfg.Order
		if oi == 0 && oj != 0 {
			return false
		}
		if oi != 0 && oj == 0 {
			return true
		}
		if oi != oj {
			return oi < oj
		}
		return refs[i].name < refs[j].name
	})
	return refs
}

// DuplicateIDPFieldOrders reports every explicit (non-zero) idp.fields /
// idp.additionalFields order: value shared by more than one field on this
// CRD, keyed by the colliding order number, field names sorted for a
// deterministic error message. order: 0 (unset) is never a collision — any
// number of fields can leave it unset at once, see allIDPFieldRefs.
//
// order used to be purely cosmetic (form layout). It no longer is:
// allIDPFieldRefs sorts by it to decide synthesized validation-rule
// priority, so two fields silently sharing an order value now means an
// arbitrary (name-based) tiebreak decides which one's violation gets
// reported when both fail at once — worth catching at load time instead.
func (c *CRDEntry) DuplicateIDPFieldOrders() map[int][]string {
	if c == nil || c.IDP == nil {
		return nil
	}
	byOrder := map[int][]string{}
	for _, ref := range c.allIDPFieldRefs() {
		if ref.cfg.Order == 0 {
			continue
		}
		byOrder[ref.cfg.Order] = append(byOrder[ref.cfg.Order], ref.name)
	}
	var dups map[int][]string
	for order, names := range byOrder {
		if len(names) > 1 {
			if dups == nil {
				dups = map[int][]string{}
			}
			sort.Strings(names)
			dups[order] = names
		}
	}
	return dups
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
			Link:     ref.link,
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
			Link:     ref.link,
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
