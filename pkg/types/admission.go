// pkg/types/admission.go
package orktypes

// ── Validation and Mutation ────────────────────────────────────────────────
//
// These declarations control policy enforcement at two points:
//
//   1. Admission time  — when ENABLE_WEBHOOKS=true, the API server calls
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
//   reconciler:
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
	Equals    string `yaml:"equals,omitempty" json:"equals,omitempty"`
	NotEquals string `yaml:"notEquals,omitempty" json:"notEquals,omitempty"`
	Prefix    string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	Suffix    string `yaml:"suffix,omitempty" json:"suffix,omitempty"`
	Contains  string `yaml:"contains,omitempty" json:"contains,omitempty"`
	Min       string `yaml:"min,omitempty" json:"min,omitempty"` // numeric, inclusive
	Max       string `yaml:"max,omitempty" json:"max,omitempty"` // numeric, inclusive

	// Explicit operator form — use when no shorthand covers the comparison.
	Operator ConditionOperator `yaml:"operator,omitempty" json:"operator,omitempty"`
	Value    string            `yaml:"value,omitempty" json:"value,omitempty"`
}

// ValidationConfig holds all validation rules for a CRD.
type ValidationConfig struct {
	// Rules — evaluated in declaration order.
	// All rules are evaluated before returning — all violations are reported
	// together so the user can fix everything in one cycle.
	Rules []ValidationRule `yaml:"rules,omitempty" json:"rules,omitempty"`
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
	Default string `yaml:"default,omitempty" json:"default,omitempty"`

	// Override — always set, regardless of current value.
	// Supports template expressions.
	Override string `yaml:"override,omitempty" json:"override,omitempty"`
}

// MutationConfig holds all mutation rules for a CRD.
type MutationConfig struct {
	// Rules — applied in declaration order.
	Rules []MutationRule `yaml:"rules,omitempty" json:"rules,omitempty"`

	// MutateFirst — when true, mutation runs before validation at reconcile
	// time. Default false (validate first, then mutate valid objects).
	//
	// At admission time, mutation always runs first — this mirrors the
	// Kubernetes webhook ordering (MutatingWebhookConfiguration fires before
	// ValidatingWebhookConfiguration). This field only affects reconcile ordering.
	MutateFirst bool `yaml:"mutateFirst,omitempty" json:"mutateFirst,omitempty"`
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
// Both default to true when ENABLE_WEBHOOKS=true and the corresponding
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
