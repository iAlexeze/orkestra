// pkg/types/conditions.go
package orktypes

// ── Conditional provisioning ───────────────────────────────────────────────────
//
// When conditions are declared on individual template sources under onCreate,
// onReconcile, or onDelete. The reconciler evaluates them against the live CR
// before calling the registry. If any condition fails, the resource is skipped
// for this reconcile cycle — not an error, just a no-op.
//
// All conditions in a When block are AND'd:
//   when:
//     - field: spec.environment
//       equals: production
//     - field: spec.enabled
//       equals: "true"
//
// Both must pass for the resource to be created.

// Condition declares a single condition to evaluate against the CR.
// Fields reference CR paths using dot notation: spec.environment, metadata.name.
type Condition struct {
	// Field — dot-notation path to a field in the CR object.
	// e.g. "spec.environment", "spec.replicas", "metadata.labels.tier"
	Field string `yaml:"field" validate:"required"`

	// Operator — how to compare the field value.
	// See ConditionOperator constants below.
	Operator ConditionOperator `yaml:"operator,omitempty"`

	// Value — the value to compare against.
	// Not used for exists/notExists operators.
	// Supports template expressions: "{{ .metadata.name }}-prod"
	Value string `yaml:"value,omitempty"`

	// Equals is a shorthand for operator: equals.
	// If set, Operator is ignored. Exists so the common case reads cleanly:
	//   when:
	//     - field: spec.environment
	//       equals: production
	Equals string `yaml:"equals,omitempty"`
}

// ConditionOperator defines how a condition's field is compared to its value.
type ConditionOperator string

const (
	// ConditionEquals — field value exactly equals the condition value (string comparison)
	ConditionEquals ConditionOperator = "equals"

	// ConditionNotEquals — field value does not equal the condition value
	ConditionNotEquals ConditionOperator = "notEquals"

	// ConditionContains — field value contains the condition value as a substring
	ConditionContains ConditionOperator = "contains"

	// ConditionPrefix — field value starts with the condition value
	ConditionPrefix ConditionOperator = "prefix"

	// ConditionSuffix — field value ends with the condition value
	ConditionSuffix ConditionOperator = "suffix"

	// ConditionExists — the field is present and non-empty
	// Value is ignored for this operator.
	ConditionExists ConditionOperator = "exists"

	// ConditionNotExists — the field is absent or empty
	// Value is ignored for this operator.
	ConditionNotExists ConditionOperator = "notExists"

	// ConditionGt — field value is numerically greater than condition value
	ConditionGt ConditionOperator = "gt"

	// ConditionLt — field value is numerically less than condition value
	ConditionLt ConditionOperator = "lt"
)

// ── Validation rules ───────────────────────────────────────────────────────────
//
// Validation rules run before runTemplateReconcile. A failing rule halts
// the reconcile loop, records a Kubernetes event, increments the metric,
// and returns an error — the workqueue retries with backoff.
//
// Rules are additive across Komposer → Katalog → CRD levels.
// A CRD-level rule for field X overrides the Komposer-level rule for field X.
// Rules for different fields accumulate.
//
// Example:
//   reconciler:
//     validation:
//       - field: spec.image
//         prefix: "myorg/"
//         message: "image must be from myorg registry"
//       - field: spec.replicas
//         max: "10"
//         message: "replicas cannot exceed 10"

// ValidationRule declares one validation constraint on a CR field.
type ValidationRule struct {
	// Field — dot-notation path to the CR field being validated.
	Field string `yaml:"field" validate:"required"`

	// Operator — how to validate the field value.
	// Supports all ConditionOperator values plus numeric range operators.
	Operator ConditionOperator `yaml:"operator,omitempty"`

	// Value — the value to validate against.
	// Supports template expressions.
	Value string `yaml:"value,omitempty"`

	Action ValidationAction `yaml:"action,omitempty"` // deny (default), warn, or cleanup

	// Shorthands — these map to the corresponding operator.
	// Use these for readability in the common cases.
	Equals    string `yaml:"equals,omitempty"`
	NotEquals string `yaml:"notEquals,omitempty"`
	Prefix    string `yaml:"prefix,omitempty"`
	Suffix    string `yaml:"suffix,omitempty"`
	Contains  string `yaml:"contains,omitempty"`
	Min       string `yaml:"min,omitempty"` // numeric minimum (inclusive)
	Max       string `yaml:"max,omitempty"` // numeric maximum (inclusive)

	// Message — the error shown in the Kubernetes event and operator log
	// when this rule is violated.
	Message string `yaml:"message" validate:"required"`

	// Cleanup-specific options.
	// Only meaningful when action: cleanup. Ignored for deny and warn.

	// GracePeriodSeconds — how long to wait before force-deleting.
	// Default 0: immediate deletion.
	// Set to a positive value for resources that need graceful shutdown.
	GracePeriodSeconds *int64 `yaml:"gracePeriodSeconds,omitempty"`

	// DryRun — when true, the cleanup rule logs and emits an event but
	// does not delete the resource. Use during policy rollout to observe
	// what would be removed before enabling live deletion.
	// Default false.
	DryRun bool `yaml:"dryRun,omitempty"`
}

// ValidationConfig holds the full validation configuration for a CRD.
type ValidationConfig struct {
	// Rules — the ordered list of validation rules.
	// All rules are evaluated. All failures are reported, not just the first.
	Rules []ValidationRule `yaml:"rules,omitempty"`
}

// ── Mutation rules ────────────────────────────────────────────────────────────
//
// Mutation rules run to apply defaults and normalise CR fields before
// reconciliation proceeds. Mutation patches the CR via a merge patch —
// the updated values are visible in kubectl describe after mutation runs.
//
// Example:
//   reconciler:
//     mutation:
//       - field: spec.replicas
//         default: "1"        # set only if field is absent or empty
//       - field: spec.logLevel
//         default: "info"
//       - field: spec.image
//         override: "myorg/{{ .metadata.name }}:latest"  # always set

// MutationRule declares one mutation to apply to a CR field.
type MutationRule struct {
	// Field — dot-notation path to the CR field to mutate.
	Field string `yaml:"field" validate:"required"`

	// Default — value to set if the field is absent or empty.
	// Supports template expressions: "{{ .metadata.name }}"
	// If the field already has a value, this is a no-op.
	Default string `yaml:"default,omitempty"`

	// Override — value to always set, regardless of current field value.
	// Use with caution — this overwrites user-provided values.
	// Supports template expressions.
	Override string `yaml:"override,omitempty"`
}

// MutationConfig holds the full mutation configuration for a CRD.
type MutationConfig struct {
	// Rules — the ordered list of mutation rules.
	// Applied in declaration order.
	Rules []MutationRule `yaml:"rules,omitempty"`

	// MutateFirst — when true, mutation runs before validation.
	// Default: false (validate first, then mutate valid objects).
	// Set to true when defaults are needed to make an object pass validation.
	MutateFirst bool `yaml:"mutateFirst,omitempty"`
}
