// pkg/types/admission.go
package types

// ── Validation and Mutation ────────────────────────────────────────────────
//
// These declarations control policy enforcement at two points:
//
//   1. Admission time  — when ENABLE_ADMISSION_WEBHOOK=true, the API server calls
//      Orkestra's /validate and /mutate endpoints synchronously during
//      kubectl apply. The CR is rejected or mutated before storage.
//
//   2. Reconcile time  — the same rules run inside the reconcile loop for
//      CRs that existed before the webhook was enabled, and as drift
//      correction for any rule violations that appear after admission.
//
// Both enforcement points read from the same Katalog declaration. There
// is no duplication — declare once, enforce everywhere.
//
// Example:
//
//   operatorBox:
//     validation:
//       - field: spec.image
//         prefix: "myorg/"
//         message: "image must be from myorg registry"
//         action: deny   # synchronous rejection at admission + halt at reconcile
//
//     mutation:
//       - field: spec.replicas
//         default: "2"   # applied at admission time, and on first reconcile

// ── ValidationAction ──────────────────────────────────────────────────────

// ValidationAction declares what happens when a validation rule fails.
//
//	deny  (default) — at admission: synchronous rejection, kubectl sees the error
//	                  at reconcile: reconciliation halted until CR is corrected
//
//	warn            — at admission: Kubernetes Warning header, CR is still stored
//	                  at reconcile: recorded as active warning on health API
//
// Default is deny when action is omitted. Warn must be declared explicitly.
// This is intentional — new rules block by default.
type ValidationAction string

const (
	ValidationActionDeny ValidationAction = "deny"
	ValidationActionWarn ValidationAction = "warn"
)

// EffectiveAction returns the effective action, defaulting to deny.
func EffectiveAction(a ValidationAction) ValidationAction {
	if a == "" {
		return ValidationActionDeny
	}
	return a
}

func (a ValidationAction) IsDeny() bool {
	return EffectiveAction(a) == ValidationActionDeny
}

func (a ValidationAction) IsWarn() bool {
	return EffectiveAction(a) == ValidationActionWarn
}

// Admission holds validation and mutation configuration for a CRD.
type Admission struct {
	Validation *ValidationConfig
	Mutation   *MutationConfig
}

// Returns true when either validation or mutation rules are declared.
func (a *Admission) HasValidationOrMutationRules() bool {
	return a.HasValidationRules() || a.HasMutationRules()
}

// Separate helpers for hasMutationRules and hasValidationRules
func (a *Admission) HasMutationRules() bool {
	if a.Mutation == nil {
		return false
	}
	return len(a.Mutation.Rules) > 0
}

func (a *Admission) HasValidationRules() bool {
	if a.Validation == nil {
		return false
	}
	return len(a.Validation.Rules) > 0
}

// ── ValidationRule ────────────────────────────────────────────────────────

// ValidationRule declares one constraint on a CR field.
//
// Shorthands (prefix, suffix, equals, etc.) exist so common rules read
// without boilerplate. Exactly one shorthand or an explicit operator+value
// pair should be set per rule.
//
// Example — deny:
//
//   - field: spec.image
//     prefix: "myorg/"
//     message: "image must be from myorg registry"
//     action: deny
//
// Example — warn, advisory only:
//
//   - field: metadata.labels.team
//     operator: exists
//     message: "all resources should declare a team owner"
//     action: warn
//
// Example — numeric bound:
//
//   - field: spec.replicas
//     max: "10"
//     message: "replicas cannot exceed 10"
type ValidationRule struct {
	// Field — dot-notation path to the CR field being validated.
	// e.g. "spec.image", "metadata.labels.tier", "spec.db.engine"
	Field string `yaml:"field" json:"field" validate:"required"`

	// Message — shown in the Kubernetes event, webhook response, and operator
	// log when this rule is violated. Make it actionable.
	Message string `yaml:"message" json:"message" validate:"required"`

	// Action — what happens when this rule fails.
	// deny (default): block at admission, halt at reconcile.
	// warn: warning at admission, advisory at reconcile.
	Action ValidationAction `yaml:"action,omitempty" json:"action,omitempty"`

	// Shorthands — use these for the common cases.
	Equals      string `yaml:"equals,omitempty" json:"equals,omitempty"`
	NotEquals   string `yaml:"notEquals,omitempty" json:"notEquals,omitempty"`
	Prefix      string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	NotPrefix   string `yaml:"notPrefix,omitempty" json:"notPrefix,omitempty"`
	Suffix      string `yaml:"suffix,omitempty" json:"suffix,omitempty"`
	NotSuffix   string `yaml:"notSuffix,omitempty" json:"notSuffix,omitempty"`
	Contains    string `yaml:"contains,omitempty" json:"contains,omitempty"`
	Min         string `yaml:"min,omitempty" json:"min,omitempty"` // numeric, inclusive
	Max         string `yaml:"max,omitempty" json:"max,omitempty"` // numeric, inclusive
	GreaterThan string `yaml:"greaterThan,omitempty" json:"greaterThan,omitempty"`
	LessThan    string `yaml:"lessThan,omitempty" json:"lessThan,omitempty"`

	// GreaterThanOrEqual is a shorthand for operator: gte.
	GreaterThanOrEqual string `yaml:"greaterThanOrEqual,omitempty" json:"greaterThanOrEqual,omitempty"`

	// LessThanOrEqual is a shorthand for operator: lte.
	LessThanOrEqual string `yaml:"lessThanOrEqual,omitempty" json:"lessThanOrEqual,omitempty"`

	// NotContains is a shorthand for operator: notContains.
	NotContains string `yaml:"notContains,omitempty" json:"notContains,omitempty"`

	// Regex is a shorthand for operator: regex.
	Regex string `yaml:"regex,omitempty" json:"regex,omitempty"`

	// Between is a shorthand for operator: between. Comma-separated "min,max",
	// both inclusive.
	Between string `yaml:"between,omitempty" json:"between,omitempty"`

	// NotBetween is a shorthand for operator: notBetween. Comma-separated
	// "min,max", same bounds as between.
	NotBetween string `yaml:"notBetween,omitempty" json:"notBetween,omitempty"`

	// In is a shorthand for operator: in. Comma-separated list.
	In string `yaml:"in,omitempty" json:"in,omitempty"`

	// NotIn is a shorthand for operator: notIn. Comma-separated list.
	NotIn string `yaml:"notIn,omitempty" json:"notIn,omitempty"`

	// Explicit operator form — use when no shorthand covers the comparison.
	Operator  ConditionOperator `yaml:"operator,omitempty" json:"operator,omitempty"`
	Value     string            `yaml:"value,omitempty" json:"value,omitempty"`
	ValueType string            `yaml:"valueType,omitempty"` // "string", "int" or "integer", "float" or "number", "bool" or "boolean"

	// When — all conditions must pass for this rule to be evaluated (AND).
	// Empty means unconditional.
	When []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// AnyOf — at least one condition must pass for this rule to be evaluated (OR).
	// When both When and AnyOf are declared, both blocks must pass.
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`

	// Link names the idp.fields or idp.additionalFields (labels/annotations)
	// key this rule concerns, for UI highlighting. Only needed when Field
	// isn't already a plain, self-describing path — additionalFields
	// entries always resolve through getLabel/getAnnotation template
	// expressions (or a notes: function built on one), and a hand-written
	// rule on a spec field can do the same (e.g. wrapping it in a format
	// check like isValidGitRepository) instead of comparing it directly.
	// Neither is itself a valid display name. When set, violations report
	// Link instead of Field as the offending field — the Control Center (or
	// any Apply API client) matches it directly against the form field it
	// rendered for that idp entry, no guessing at the expression required.
	//
	// Link is a plain literal string, not a template expression — it's the
	// exact idp.fields / idp.additionalFields key itself, never resolved
	// against the CR the way Field/Value/Message can be.
	//
	// Validated at katalog-load time: must match a key declared in
	// idp.fields, idp.additionalFields.labels, or .additionalFields.annotations
	// for this CRD. Linking a spec field whose Field is already exactly
	// "spec.<name>" is an error, though — at that point Field already is a
	// clean display name on its own, so the link is always redundant.
	//
	// Automatically set by the required/enum rules idp.additionalFields
	// entries synthesize. Hand-written rules that target the same field
	// (e.g. multiple focused checks split across separate rules instead of
	// one compound expression) should set it too, so every rule touching
	// that field highlights consistently:
	//
	//	validation:
	//	  rules:
	//	    - field: '{{ isDNS1123Subdomain team }}'
	//	      link: team
	//	      equals: "true"
	//	      message: "team must be a valid DNS subdomain"
	//	      action: deny
	Link string `yaml:"link,omitempty" json:"link,omitempty"`
}

// ValidationConfig holds all validation rules for a CRD.
type ValidationConfig struct {
	// Include is a path (relative to the katalog file) to a YAML file whose
	// top-level value is a list of ValidationRule entries. Expanded at load
	// time — included rules come first, inline rules append after.
	Include string `yaml:"include,omitempty" json:"include,omitempty"`

	// External declares HTTP calls made before validation rules are evaluated.
	// Results are available in rule field:, equals:, and message: expressions.
	// At admission time: all entries fire.
	// At reconcile time: only entries with fires.reconcile != false fire.
	External []ExternalCallSpec `yaml:"external,omitempty" json:"external,omitempty"`

	// Rules — evaluated in declaration order.
	// All rules are evaluated before returning — all violations are reported
	// together so the user can fix everything in one cycle.
	Rules []ValidationRule `yaml:"rules,omitempty" json:"rules,omitempty"`
}

// AdmissionExternal returns all external calls — every entry fires at admission time.
func (c *ValidationConfig) AdmissionExternal() []ExternalCallSpec {
	if c == nil {
		return nil
	}
	return c.External
}

// ReconcileExternal returns only calls that fire at reconcile time.
// Entries with fires.reconcile: false are excluded.
func (c *ValidationConfig) ReconcileExternal() []ExternalCallSpec {
	if c == nil {
		return nil
	}
	var out []ExternalCallSpec
	for _, e := range c.External {
		if e.Fires.firesAtReconcile() {
			out = append(out, e)
		}
	}
	return out
}

// HasDenyRules reports whether any rules use action: deny (or the default).
// Used to decide whether to register a ValidatingWebhookConfiguration.
func (c *ValidationConfig) HasDenyRules() bool {
	if c == nil {
		return false
	}
	for _, r := range c.Rules {
		if r.Action.IsDeny() || r.Action == "" {
			return true
		}
	}
	return false
}

// HasWarnRules reports whether any rules use action: warn.
func (c *ValidationConfig) HasWarnRules() bool {
	if c == nil {
		return false
	}
	for _, r := range c.Rules {
		if r.Action.IsWarn() {
			return true
		}
	}
	return false
}

// ── MutationRule ──────────────────────────────────────────────────────────

// MutationRule declares one field mutation.
//
// Two mutation types:
//
//	default  — set the field only if it is absent or empty.
//	           At admission time: applied before validation runs.
//	           At reconcile time: applied on first reconcile cycle.
//
//	override — always set the field, regardless of current value.
//	           Use with caution — overwrites user-provided values.
//
// Both types support Go template expressions resolved against the CR object:
//
//	default: "{{ .metadata.name }}-default"
//	override: "myorg/{{ .metadata.name }}:latest"
//
// Example:
//
//	mutation:
//	  - field: spec.replicas
//	    default: "2"
//	  - field: spec.logLevel
//	    default: "info"
//	  - field: spec.image
//	    override: "myorg/{{ .metadata.name }}:latest"
type MutationRule struct {
	// Field — dot-notation path to the CR field to mutate.
	Field string `yaml:"field" json:"field" validate:"required"`

	// Default — set only if the field is absent or empty.
	// Supports template expressions.
	Default interface{} `yaml:"default,omitempty"` // accepts int, bool, string from YAML

	// Override — always set, regardless of current value.
	// Supports template expressions.
	Override interface{} `yaml:"override,omitempty"` // accepts int, bool, string from YAML

	// Value type
	ValueType string `yaml:"valueType,omitempty"` // "string", "int" or "integer", "float" or "number", "bool" or "boolean"

	// When — all conditions must pass for this rule to be applied (AND).
	When []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// AnyOf — at least one condition must pass for this rule to be applied (OR).
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
}

// MutationConfig holds all mutation rules for a CRD.
type MutationConfig struct {
	// Include is a path (relative to the katalog file) to a YAML file whose
	// top-level value is a list of MutationRule entries. Expanded at load
	// time — included rules come first, inline rules append after.
	Include string `yaml:"include,omitempty" json:"include,omitempty"`

	// External declares HTTP calls made before mutation rules are applied.
	// Results are available in rule field:, default:, and override: expressions.
	// At admission time: all entries fire.
	// At reconcile time: only entries with fires.reconcile != false fire.
	External []ExternalCallSpec `yaml:"external,omitempty" json:"external,omitempty"`

	// Rules — applied in declaration order.
	Rules []MutationRule `yaml:"rules,omitempty" json:"rules,omitempty"`

	// MutateFirst — when true, mutation runs before validation at reconcile
	// time. Default true (mutate first, then validate valid objects).
	//
	// At admission time, mutation always runs first — this mirrors the
	// Kubernetes webhook ordering (MutatingWebhookConfiguration fires before
	// ValidatingWebhookConfiguration). This field only affects reconcile ordering.
	MutateFirst bool `yaml:"mutateFirst,omitempty" json:"mutateFirst,omitempty"`
}

// AdmissionExternal returns all external calls — every entry fires at admission time.
func (c *MutationConfig) AdmissionExternal() []ExternalCallSpec {
	if c == nil {
		return nil
	}
	return c.External
}

// ReconcileExternal returns only calls that fire at reconcile time.
// Entries with fires.reconcile: false are excluded.
func (c *MutationConfig) ReconcileExternal() []ExternalCallSpec {
	if c == nil {
		return nil
	}
	var out []ExternalCallSpec
	for _, e := range c.External {
		if e.Fires.firesAtReconcile() {
			out = append(out, e)
		}
	}
	return out
}

// ── AdmissionWebhookConfig ────────────────────────────────────────────────

// AdmissionWebhookConfig is the per-CRD admission webhook control block.
// Declared under spec.crds[].webhooks in the Katalog.
//
// Example:
//
//   - name: website
//     webhooks:
//     validation: true   # intercept at admission — ValidatingWebhookConfiguration
//     mutation: true     # intercept at admission — MutatingWebhookConfiguration
//
// Both default to true when ENABLE_ADMISSION_WEBHOOK=true and the corresponding
// validation/mutation block has rules declared. Set to false to opt a specific
// CRD out of admission interception while keeping its reconcile-time enforcement.
type AdmissionWebhookConfig struct {
	// Validation — include this CRD in the ValidatingWebhookConfiguration.
	// Default: true when validation rules are declared.
	Validation *bool `yaml:"validation,omitempty" json:"validation,omitempty"`

	// Mutation — include this CRD in the MutatingWebhookConfiguration.
	// Default: true when mutation rules are declared.
	Mutation *bool `yaml:"mutation,omitempty" json:"mutation,omitempty"`

	// Operations — which operations trigger the webhook.
	// Default: ["CREATE", "UPDATE"]
	// Valid values: CREATE, UPDATE, DELETE, CONNECT
	Operations []string `yaml:"operations,omitempty" json:"operations,omitempty"`
}

// WebhookValidationEnabled reports whether admission-time validation is
// enabled for this CRD. True when Validation is nil or true.
func (w *AdmissionWebhookConfig) WebhookValidationEnabled() bool {
	if w == nil || w.Validation == nil {
		return true // default on
	}
	return *w.Validation
}

// WebhookMutationEnabled reports whether admission-time mutation is enabled.
func (w *AdmissionWebhookConfig) WebhookMutationEnabled() bool {
	if w == nil || w.Mutation == nil {
		return true // default on
	}
	return *w.Mutation
}

// EffectiveOperations returns the operations this webhook intercepts.
// Defaults to CREATE and UPDATE when not specified.
func (w *AdmissionWebhookConfig) EffectiveOperations() []string {
	if w == nil || len(w.Operations) == 0 {
		return []string{"CREATE", "UPDATE"}
	}
	return w.Operations
}
