// pkg/types/conditions.go
package types

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

	// NotEquals is a shorthand for operator: notEquals.
	// If set, Operator is ignored.
	// 	when:
	//     - field: spec.environment
	//       notEquals: production
	NotEquals string `yaml:"notEquals,omitempty"`

	// Contains is a shorthand for operator: contains.
	// If set, Operator is ignored.
	// 	when:
	//     - field: spec.environment
	//       contains: prod
	Contains string `yaml:"contains,omitempty"`

	// Prefix is a shorthand for operator: prefix.
	// If set, Operator is ignored.
	// 	when:
	//     - field: spec.environment
	//       prefix: prod
	Prefix string `yaml:"prefix,omitempty"`

	// Suffix is a shorthand for operator: suffix.
	// If set, Operator is ignored.
	// 	when:
	//     - field: spec.environment
	//       suffix: prod
	Suffix string `yaml:"suffix,omitempty"`

	// Numeric shorthands
	GreaterThan string `yaml:"greaterThan,omitempty"`
	LessThan    string `yaml:"lessThan,omitempty"`

	// Notify — optional notification targets to trigger when this condition
	// transitions from false → true.
	//
	// Targets reference names declared under:
	//   notification:
	//     teams:
	//       platform:
	//         email: [...]
	//       application:
	//         email: [...]
	//         slack: [...]
	//
	// Example:
	//   when:
	//     - field: status.phase
	//       equals: Degraded
	//       notify:
	//         - platform
	//         - application
	//
	// Notifications fire only on transition, not on every reconcile.
	Notify []string `yaml:"notify,omitempty"`
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
	// Use for: first-reconcile detection (phase not yet written).
	//   when:
	//     - field: status.phase
	//       operator: notExists
	ConditionNotExists ConditionOperator = "notExists"

	// ConditionGt — field value is numerically greater than condition value
	ConditionGt ConditionOperator = "gt"

	// ConditionLt — field value is numerically less than condition value
	ConditionLt ConditionOperator = "lt"

	// ConditionIn — field value is one of a comma-separated list.
	// Empty string matches an empty field (for first-reconcile detection).
	//   when:
	//     - field: status.phase
	//       operator: in
	//       value: ",Pending"   # empty or "Pending"
	ConditionIn ConditionOperator = "in"

	// ConditionUnique — field value is unique across all existing CR instances.
	//
	// Only valid in validation rules (deny action). Checks the informer cache
	// for any existing CR with the same field value.
	//
	//	validation:
	//	  rules:
	//	    - field: spec.domain
	//	      operator: unique
	//	      message: "spec.domain must be unique across all instances"
	//	      action: deny
	//
	// Not valid in when: blocks on template sources — uniqueness requires
	// informer access which is not available during template evaluation.
	// In when: context it is treated as always-true (see pkg/types/when/EvaluateOneCond).
	ConditionUnique ConditionOperator = "unique"

	// ConditionTypeOf — check field type
	//
	ConditionTypeOf ConditionOperator = "typeOf"

	// ConditionTypeMap — field value is a map (YAML object)
	ConditionTypeMap ConditionOperator = "typeMap"

	// ConditionTypeList — field value is a slice (YAML array)
	ConditionTypeList ConditionOperator = "typeList"

	// ConditionTypeString — field value is a string
	ConditionTypeString ConditionOperator = "typeString"

	// ConditionTypeNumber — field value is a number (int/float)
	ConditionTypeNumber ConditionOperator = "typeNumber"

	// ConditionTypeBool — field value is a boolean
	ConditionTypeBool ConditionOperator = "typeBool"

	// ConditionTypeNull — field value is null or missing
	ConditionTypeNull ConditionOperator = "typeNull"
)
