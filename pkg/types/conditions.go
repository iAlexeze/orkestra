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

// Condition declares a single condition to evaluate against the CR or runtime state.
// Fields reference CR paths using dot notation: spec.environment, metadata.name.
//
// The same type is used in:
//   - when: / anyOf: on template sources (resource conditions)
//   - operatorBox.autoscale.conditions.anyOf and when: (autoscale conditions)
//   - operatorBox.rollback.trigger (rollback conditions)
//   - notification condition blocks
type Condition struct {
	// Field — dot-notation path to a field in the CR object or a runtime metric.
	// e.g. "spec.environment", "metrics.queueDepth", "cross.managed-database.metrics.queueDepth"
	Field string `yaml:"field" validate:"required" json:"field"`

	// Operator — how to compare the field value.
	// See ConditionOperator constants below.
	Operator ConditionOperator `yaml:"operator,omitempty" json:"operator,omitempty"`

	// Value — the value to compare against.
	// Not used for exists/notExists operators.
	// Supports template expressions: "{{ .metadata.name }}-prod"
	Value string `yaml:"value,omitempty" json:"value,omitempty"`

	// Equals is a shorthand for operator: equals.
	Equals string `yaml:"equals,omitempty" json:"equals,omitempty"`

	// NotEquals is a shorthand for operator: notEquals.
	NotEquals string `yaml:"notEquals,omitempty" json:"notEquals,omitempty"`

	// Contains is a shorthand for operator: contains.
	Contains string `yaml:"contains,omitempty" json:"contains,omitempty"`

	// Prefix is a shorthand for operator: prefix.
	Prefix string `yaml:"prefix,omitempty" json:"prefix,omitempty"`

	// Suffix is a shorthand for operator: suffix.
	Suffix string `yaml:"suffix,omitempty" json:"suffix,omitempty"`

	// Exists is a shorthand for operator: exists.
	// When true, the condition passes when the field is present and non-empty.
	Exists *bool `yaml:"exists,omitempty" json:"exists,omitempty"`

	// NotExists is a shorthand for operator: notExists.
	// When true, the condition passes when the field is absent or empty.
	NotExists *bool `yaml:"notExists,omitempty" json:"notExists,omitempty"`

	// Numeric shorthands
	GreaterThan string `yaml:"greaterThan,omitempty" json:"greaterThan,omitempty"`
	LessThan    string `yaml:"lessThan,omitempty" json:"lessThan,omitempty"`

	// ── Time-based fields (anyOf in autoscale conditions) ────────────────────

	// Time — active when the current time is within the declared window.
	// After and Before are both optional; omit one for a half-open range.
	//   anyOf:
	//     - time:
	//         after: "08:00"
	//         before: "20:00"
	Time *TimeWindow `yaml:"time,omitempty" json:"time,omitempty"`

	// DayOfWeek — active on the specified days of the week.
	//   anyOf:
	//     - dayOfWeek:
	//         in: [Monday, Tuesday, Wednesday, Thursday, Friday]
	DayOfWeek *DayOfWeekCondition `yaml:"dayOfWeek,omitempty" json:"dayOfWeek,omitempty"`

	// Cron — a standard cron expression (5-field) that defines when the
	// window opens. Duration defines how long the window stays open.
	// Without Duration, the window closes after one evaluation interval.
	Cron string `yaml:"cron,omitempty" json:"cron,omitempty"`

	// Duration — how long a cron-opened window remains active.
	Duration Duration `yaml:"duration,omitempty" json:"duration,omitempty"`

	// ── Notification ─────────────────────────────────────────────────────────

	// Notify declares teams to alert when this condition is true.
	Notify *NotifyBlock `yaml:"notify,omitempty" json:"notify,omitempty"`

	// ── Cross-binary metric fallback ─────────────────────────────────────────

	// Source is the HTTP fallback for cross-binary metric observation.
	// Only used when field is a cross.<crd>.metrics.* field and the CRD
	// is not registered in GlobalCrossMetricsRegistry (different binary).
	// The endpoint must be the remote operator's /katalog/{crd} URL.
	//
	//   when:
	//     - field: cross.managed-database.metrics.queueDepth
	//       greaterThan: "500"
	//       source:
	//         host: "http://orkestra-database-operator:8080"
	// 		   crd: managed-database
	//   when:
	//     - field: cross.managed-database.metrics.queueDepth
	//       greaterThan: "500"
	//       source:
	//         endpoint: "http://non-orkestra-database-operator:8080/api/managed-database/metrics"
	Source *CrossSource `yaml:"source,omitempty" json:"source,omitempty"`
}

// NotifyBlock declares notification targets and an optional message override
// for a specific condition.
type NotifyBlock struct {
	// Teams is the list of team names (from notification.teams) to alert.
	Teams []string `yaml:"teams" json:"teams"`
	// Message is a Go template expression for the notification body.
	// Overrides the team's own message template.
	// When empty, uses the team's configured message or the system default.
	Message string `yaml:"message,omitempty" json:"message,omitempty"`
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
