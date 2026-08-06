package types

import (
	"fmt"
	"slices"
	"strings"
)

// ServeDenyReason is returned by TokenAllowed to let the caller compose a
// precise 403 message without duplicating the check logic.
type ServeDenyReason int

const (
	ServeDenyReasonNone         ServeDenyReason = iota // allowed
	ServeDenyReasonUnknownToken                        // token not in tokens map
	ServeDenyReasonNamespace                           // namespace not in token's namespaces
	ServeDenyReasonOperation                           // operation not in token's permissions
)

// ServeConfig declares serve exposure settings for a CRD entry.
type ServeConfig struct {
	// Enabled surfaces this CRD in the Control Center as a self-service form.
	// Requires gateway.api.enabled: true on the Katalog.
	// Default: false.
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Include is a path (relative to the katalog file) to a YAML file with a
	// "fields:" map and/or an "additionalFields:" block (same shape as the
	// inline equivalents below). Expanded at load time — the result is merged
	// into Fields and AdditionalFields respectively, with inline entries
	// taking precedence per key.
	Include string `yaml:"include,omitempty" json:"include,omitempty"`

	// Fields provides presentation hints layered on top of the CRD's OpenAPI
	// schema. Each key matches a field path in spec. Hints are merged with the
	// schema at GET /api/v1/schema/{kind} time — they do not replace the schema.
	Fields map[string]ServeFieldConfig `yaml:"fields,omitempty" json:"fields,omitempty"`

	// Labels exposes label keys as self-service form fields, written to
	// metadata.labels on apply. Declare type explicitly — no CRD schema to infer from.
	Labels map[string]ServeFieldConfig `yaml:"labels,omitempty" json:"labels,omitempty"`

	// Annotations exposes annotation keys as self-service form fields, written to
	// metadata.annotations on apply. Declare type explicitly — no CRD schema to infer from.
	Annotations map[string]ServeFieldConfig `yaml:"annotations,omitempty" json:"annotations,omitempty"`

	// Ignore lists spec field names hidden from the serve form.
	Ignore []string `yaml:"ignore,omitempty" json:"ignore,omitempty"`

	// Title is the human-readable name shown in the Control Center catalog.
	// Defaults to kind when not set.
	Title string `yaml:"title,omitempty" json:"title,omitempty"`

	// Category is a catalog label used when listing available schemas
	// via GET /api/v1/schema/. Example: "Compute", "Data", "Security".
	Category string `yaml:"category,omitempty" json:"category,omitempty"`

	// Description is a short human-readable summary shown in the service catalog.
	// Falls back to the CRD-level description when not set.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Target is the caller-facing identifier for this CRD in the Gateway API.
	// Defaults to the lowercased kind when not set.
	// Must be lowercase alphanumeric with optional hyphens.
	// Must be unique across all serve-enabled CRDs in this Katalog.
	Target string `yaml:"target,omitempty" json:"target,omitempty"`

	// ForceConflict, when true, sets Force: true on every server-side apply
	// for this CRD — the gateway takes ownership of any conflicting fields
	// rather than surfacing a conflict error. Equivalent to helm --force-conflict.
	// Can be overridden per-request with ?overwrite=true regardless of this setting.
	// Default: false.
	ForceConflict bool `yaml:"forceConflict,omitempty" json:"forceConflict,omitempty"`

	// Name is a template expression the Gateway API resolves server-side to
	// decide the CR's metadata.name — e.g. '{{ repoSlug .spec.repository }}'.
	// Once set, it always wins over whatever (if anything) the client sent.
	// Applies regardless of CRD scope (metadata.name exists either way),
	// but is optional, unlike Namespace below: most CRDs still want the
	// caller to choose a name (multiple concurrent instances of one repo —
	// PR previews, ephemeral environments). Set this only when instances are
	// 1:1 with some other identity the caller already supplies (repository,
	// team) and redeploys should update the same CR in place rather than
	// create a new one — a stable environment where only the image tag
	// or few configuration changes between deploys.
	//
	// When unset, the Gateway API requires the caller to supply metadata.name
	// and rejects the request with a structured violation if it's empty,
	// rather than leaving it to the Kubernetes API server's own generic
	// rejection.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Namespace is a template expression the Gateway API resolves server-side
	// to decide which namespace a new CR is created in — e.g. '{{ teamName }}'.
	// Same resolution mechanics as Name above, but only applies to namespaced
	// CRDs, and — unlike Name — is required rather than optional: the
	// Control Center form and any Gateway API caller never need to render or
	// submit namespace themselves. A plain literal (no template) is valid
	// too, for a CRD whose instances always land in one fixed namespace.
	//
	// Required when the CRD is namespaced (the default) and serve.enabled is
	// true — see Katalog.validateServeNamespace. Meaningless, and rejected at
	// load time, on a cluster-scoped CRD (namespaced: false) — there's no
	// namespace to resolve into.
	//
	// This only affects the Gateway API (POST /api/v1/apply). Raw kubectl
	// callers are unaffected: kubectl always resolves some namespace
	// client-side before a request reaches the API server (typically
	// "default"), so there is never a genuinely empty namespace for a
	// webhook to fill in the way an omitted JSON field lets the Gateway API
	// detect intent — deliberately not implemented as a mutating webhook
	// default for that reason.
	//
	// Resolves into an existing namespace; it does not create one. The
	// platform team provisions whatever namespace(s) this can resolve to
	// ahead of time (setup.apply in e2e, real onboarding in production) —
	// same as a cluster-scoped CRD's onCreate provisioning a namespace as a
	// child resource is a different, complementary answer to the same
	// "developer shouldn't have to pick a namespace" problem.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// Config declares optional response shaping applied by the gateway when
	// returning CR data to callers. Evaluated at request time against the
	// fetched CR — no additional Kubernetes API calls are made.
	// Nil config returns the CR unchanged.
	Config *ServeConfigSettings `yaml:"config,omitempty" json:"config,omitempty"`

	// Tokens maps gateway token names to the operations they may perform on this
	// CRD and, optionally, the namespaces they may access.
	// When empty, any valid gateway token may perform any operation on this CRD.
	// ork validate confirms every token name here matches an entry in gateway.api.auth.tokens.
	Tokens map[string]ServeTokenPermissions `yaml:"tokens,omitempty" json:"tokens,omitempty"`
}

// ServeFieldConfig holds display hints for one spec field in the serve form.
type ServeFieldConfig struct {
	// Label overrides the field name in the rendered form.
	Label string `yaml:"label,omitempty" json:"label,omitempty"`

	// Placeholder is the input placeholder text.
	Placeholder string `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`

	// Hint is descriptive text rendered below the field.
	Hint string `yaml:"hint,omitempty" json:"hint,omitempty"`

	// Order controls position in the rendered form. Lower values appear first.
	// Fields with no order (0) appear after all explicitly ordered fields —
	// any number of fields may leave it unset, since 0 means "no preference,"
	// not a real position.
	//
	// Not just form layout: allServeFieldRefs sorts by Order to decide
	// synthesized validation-rule priority too (see RuleViolation.Field /
	// ValidationResult.DenialMessage — only the first violation is reported
	// as the headline denial reason), so the field a developer sees first is
	// also the one whose error they see first when several fail at once.
	// Two fields on the same CRD sharing a non-zero Order is a load-time
	// error (see Katalog.validateServeFieldOrder) for exactly this reason.
	Order int `yaml:"order,omitempty" json:"order,omitempty"`

	// Category is a section heading for visual grouping. Fields sharing a category
	// are rendered under the same heading. Works with When — if all fields in
	// a category are hidden, the heading is also hidden.
	Category string `yaml:"category,omitempty" json:"category,omitempty"`

	// When is a list of conditions that must ALL be true for this field to be
	// visible. Evaluated client-side as the user fills the form. An empty When
	// means the field is always visible.
	// Supports: equals, notEquals, time, dayOfWeek, cron, negate — same as
	// template source when: blocks.
	When []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// AnyOf is a list of conditions where at least ONE must be true for the
	// field to be visible. OR counterpart to When (AND).
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`

	// Required, when true, marks the field as mandatory in the serve form —
	// the browser enforces this natively (asterisk on the label, form cannot
	// be submitted while empty) — and is also enforced server-side: an
	// implicit exists validation rule is synthesized automatically at
	// katalog load time (see CRDEntry.RequiredServeFieldRules), covering every
	// client of the Gateway API, not just the Control Center form. No matching
	// validation.rules entry needs to be hand-written.
	Required bool `yaml:"required,omitempty" json:"required,omitempty"`

	// Disabled, when non-empty, renders the field greyed out with this string
	// as the reason. The field is excluded from form submission.
	// Use for maintenance windows or temporarily locked fields.
	Disabled string `yaml:"disabled,omitempty" json:"disabled,omitempty"`

	// Type is required for AdditionalFields entries (labels/annotations have
	// no CRD schema to infer type from). Ignored for Fields entries, which
	// always infer type from the CRD's OpenAPI schema.
	// Supported: string (default), integer, number, boolean, enum.
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Enum lists valid values when Type == "enum". Required in that case.
	Enum []string `yaml:"enum,omitempty" json:"enum,omitempty"`

	// Path is the dot-notation path in the CRD spec where this field belongs.
	// Example: "app.repository", "scaling.minReplicas"
	// When set, the field is mapped to this nested path.
	// When empty, the field name is used as the path (flat).
	Path string `yaml:"path,omitempty" json:"path,omitempty"`
}

// ServeConfigSettings is the container for gateway-level CRD configuration.
// Named with the parent prefix to avoid colliding with the package-level
// Config types while keeping the YAML key simply "config:".
type ServeConfigSettings struct {
	// Response controls what callers see when the gateway returns CR data.
	// See ServeResponseConfig for full documentation.
	Response *ServeResponseConfig `yaml:"response,omitempty" json:"response,omitempty"`
}

// IsValidServeFieldType reports whether t is a valid ServeFieldConfig.Type value.
// "" (omitted) is valid — it means the default, string.
func IsValidServeFieldType(t string) bool {
	switch t {
	case "", "string", "integer", "number", "boolean", "enum":
		return true
	default:
		return false
	}
}

// FieldType returns the configured type for this field or default to "string".
func (f ServeFieldConfig) FieldType() string {
	if f.Type != "" {
		return f.Type
	}
	return "string"
}

// SpecPath returns the dot-notation path to use in the CRD spec.
// If Path is set, use Path. Otherwise, use the field name.
func (f ServeFieldConfig) SpecPath(name string) string {
	if f.Path != "" {
		return f.Path
	}
	return name
}

// IsNested returns true if the spec path contains a dot.
func (f ServeFieldConfig) IsNested(name string) bool {
	return strings.Contains(f.SpecPath(name), ".")
}

// HasSpecPath returns true if the spec path is set.
func (f ServeFieldConfig) HasSpecPath() bool {
	return f.Path != ""
}

// HasTokenRestrictions reports whether any per-token access rules are declared.
// When false, any valid gateway token may access this CRD — backward-compatible
// with the previous model where tokens were only checked for existence.
func (i *ServeConfig) HasTokenRestrictions() bool {
	if i == nil {
		return false
	}
	return len(i.Tokens) > 0
}

// AllowedServeTokens returns a list of token names allowed for this serve configuration
func (i *ServeConfig) AllowedServeTokens() []string {
	if i == nil {
		return nil
	}
	var tokens []string
	for token := range i.Tokens {
		tokens = append(tokens, token)
	}
	return tokens
}

// TokenAllowed reports whether tokenName may perform op in namespace on this
// CRD for the given endpoint class.
//
// Returns (true, ServeDenyReasonNone) when allowed.
// Returns (false, reason) when denied; reason carries the specific cause so
// callers can compose precise error messages without re-implementing the logic.
// The check is intentionally three-stage for clarity in error messages:
//  1. Is the token listed at all?   → 403 unknown token
//  2. Is the namespace permitted?   → 403 namespace not allowed
//  3. Is the operation permitted?   → 403 operation not allowed
func (c *ServeConfig) TokenAllowed(
	tokenName, op, namespace string,
	class ServeEndpointClass,
) (bool, ServeDenyReason) {
	if !c.HasTokenRestrictions() {
		return true, ServeDenyReasonNone
	}

	perms, ok := c.Tokens[tokenName]
	if !ok {
		return false, ServeDenyReasonUnknownToken
	}

	// For schema endpoints, skip namespace check entirely.
	// Schema endpoints are cluster-scoped and don't have a namespace.
	if class != ServeClassSchema {
		if len(perms.Namespaces) > 0 && !slices.Contains(perms.Namespaces, namespace) {
			return false, ServeDenyReasonNamespace
		}
	}

	active := perms.activePerms(class)
	if len(active) == 0 {
		return false, ServeDenyReasonOperation
	}

	for _, p := range active {
		if p == ServeOpAll || p == op {
			return true, ServeDenyReasonNone
		}
	}
	return false, ServeDenyReasonOperation
}

// Message returns a human-readable denial message for use in HTTP responses
// and ork validate output.
func (r ServeDenyReason) Message(tokenName, op, kind, namespace string) string {
	switch r {
	case ServeDenyReasonUnknownToken:
		return fmt.Sprintf(
			"token %q is not allowed to access %q — not listed in serve.tokens",
			tokenName, kind,
		)
	case ServeDenyReasonNamespace:
		return fmt.Sprintf(
			"token %q is not allowed to access %q in namespace %q",
			tokenName, kind, namespace,
		)
	case ServeDenyReasonOperation:
		return fmt.Sprintf(
			"token %q lacks %q permission on %q",
			tokenName, op, kind,
		)
	default:
		return ""
	}
}
