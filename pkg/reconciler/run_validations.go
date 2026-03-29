// pkg/reconciler/run_validation.go
package reconciler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ── Prometheus metrics ────────────────────────────────────────────────────────
// Four metrics — all meaningful, all unique to Orkestra.
// Nothing kubectl or Prometheus itself shows about per-CRD validation.

var (
	// validationTotal counts every validation pass or rejection across all CRDs.
	// Labeled by CRD and result (passed|rejected).
	// Use: alert when rejection rate exceeds a threshold.
	validationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "controller_validation_total",
			Help: "Total number of validation checks performed, labeled by result.",
		},
		[]string{"crd", "result"},
	)

	// validationRejectedDetail counts rejections with field and rule context.
	// More expensive than validationTotal but gives actionable detail:
	// which field is failing and which rule is triggering it.
	// Use: alert + investigate which rule is causing the most friction.
	validationRejectedDetail = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "controller_validation_rejected_total",
			Help: "Validation rejections labeled by CRD, field, and rule type.",
		},
		[]string{"crd", "field", "rule"},
	)
)

// ValidationResult holds the outcome of running all validation rules for one reconcile.
type ValidationResult struct {
	// Passed — true when all rules passed
	Passed bool

	// Warning
	Warnings []string

	// Deny
	Deny bool

	// Violations — one entry per failed rule
	Violations []ValidationViolation
}

// ValidationViolation describes one failed validation rule.
type ValidationViolation struct {
	Field   string
	Rule    string
	Value   string // the actual CR field value that failed
	Message string // the user-defined message from the Katalog
}

// Error returns a combined error message for all violations.
func (r *ValidationResult) Error() error {
	if r.Passed {
		return nil
	}
	msgs := make([]string, 0, len(r.Violations))
	for _, v := range r.Violations {
		msgs = append(msgs, fmt.Sprintf("field %q: %s (got %q)", v.Field, v.Message, v.Value))
	}
	return fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
}

// runValidation evaluates all validation rules against the CR object.
// All rules are evaluated — multiple violations are collected and reported together.
// Returns a ValidationResult. The caller decides whether to halt reconciliation.
//
// Called from generic.go before runTemplateReconcile (or after runMutation
// when mutateFirst: true).
func runValidation(
	obj domain.Object,
	cfg *orktypes.ValidationConfig,
	crdName string,
) *ValidationResult {
	result := &ValidationResult{Passed: true}

	if cfg == nil || len(cfg.Rules) == 0 {
		return result
	}

	u, ok := toUnstructured(obj)
	if !ok {
		// Typed objects — cannot evaluate dot-notation paths.
		// Validation is skipped for typed CRDs. Use Go hooks for typed validation.
		// TODO — Use templating to access typed fields
		logger.Debug().
			Str("crd", crdName).
			Msg("validation: typed object — skipping declarative validation (use Go hooks)")
		return result
	}

	for _, rule := range cfg.Rules {
		violation := evaluateValidationRule(u, rule)
		if violation != nil {
			result.Passed = false
			result.Violations = append(result.Violations, *violation)

			// Per-rejection detail metric — field and rule context
			validationRejectedDetail.WithLabelValues(crdName, rule.Field, ruleType(rule)).Inc()
		}
	}

	// Aggregate metric — one counter per reconcile, labeled pass|reject
	resultLabel := "passed"
	if !result.Passed {
		resultLabel = "rejected"
	}
	validationTotal.WithLabelValues(crdName, resultLabel).Inc()

	if !result.Passed {
		logger.Info().
			Str("crd", crdName).
			Str("name", obj.GetName()).
			Str("namespace", obj.GetNamespace()).
			Int("violations", len(result.Violations)).
			Msg("validation: rules failed — reconciliation halted")
	}

	return result
}

// evaluateValidationRule evaluates one rule against the CR.
// Returns a ValidationViolation if the rule fails, nil if it passes.
func evaluateValidationRule(obj *unstructured.Unstructured, rule orktypes.ValidationRule) *ValidationViolation {
	op, expected := resolveValidationOp(rule)

	fieldVal, found := resolveField(obj.Object, rule.Field)

	// Build a violation helper
	fail := func() *ValidationViolation {
		return &ValidationViolation{
			Field:   rule.Field,
			Rule:    string(op),
			Value:   fieldVal,
			Message: rule.Message,
		}
	}

	switch op {
	case orktypes.ConditionExists:
		if !found || fieldVal == "" {
			return fail()
		}

	case orktypes.ConditionNotExists:
		if found && fieldVal != "" {
			return fail()
		}

	case orktypes.ConditionEquals:
		if !found || fieldVal != expected {
			return fail()
		}

	case orktypes.ConditionNotEquals:
		if found && fieldVal == expected {
			return fail()
		}

	case orktypes.ConditionContains:
		if !found || !strings.Contains(fieldVal, expected) {
			return fail()
		}

	case orktypes.ConditionPrefix:
		if !found || !strings.HasPrefix(fieldVal, expected) {
			return fail()
		}

	case orktypes.ConditionSuffix:
		if !found || !strings.HasSuffix(fieldVal, expected) {
			return fail()
		}

	case orktypes.ConditionGt: // used as Min when coming from rule.Min
		cv, err := strconv.ParseFloat(expected, 64)
		if err != nil {
			logger.Warn().Str("field", rule.Field).Str("val", expected).
				Msg("validation: min/gt requires numeric value — rule skipped")
			return nil
		}
		fv, err := strconv.ParseFloat(fieldVal, 64)
		if err != nil || fv < cv {
			return fail()
		}

	case orktypes.ConditionLt: // used as Max when coming from rule.Max
		cv, err := strconv.ParseFloat(expected, 64)
		if err != nil {
			logger.Warn().Str("field", rule.Field).Str("val", expected).
				Msg("validation: max/lt requires numeric value — rule skipped")
			return nil
		}
		fv, err := strconv.ParseFloat(fieldVal, 64)
		if err != nil || fv > cv {
			return fail()
		}
	}

	return nil // rule passed
}

// resolveValidationOp resolves the effective operator and comparison value
// from a ValidationRule, handling all shorthand fields.
func resolveValidationOp(r orktypes.ValidationRule) (orktypes.ConditionOperator, string) {
	// Shorthands take precedence — listed in order of typical usage
	if r.Equals != "" {
		return orktypes.ConditionEquals, r.Equals
	}
	if r.NotEquals != "" {
		return orktypes.ConditionNotEquals, r.NotEquals
	}
	if r.Prefix != "" {
		return orktypes.ConditionPrefix, r.Prefix
	}
	if r.Suffix != "" {
		return orktypes.ConditionSuffix, r.Suffix
	}
	if r.Contains != "" {
		return orktypes.ConditionContains, r.Contains
	}
	if r.Min != "" {
		// Min maps to Gt — field must be >= min (we check field >= min)
		// We store min as Gt target but evaluate as >=
		return orktypes.ConditionGt, r.Min
	}
	if r.Max != "" {
		// Max maps to Lt — field must be <= max
		return orktypes.ConditionLt, r.Max
	}
	if r.Operator != "" {
		return r.Operator, r.Value
	}
	// No operator and no value — default to exists check
	return orktypes.ConditionExists, ""
}

// ruleType returns a short string identifying the rule type for the metric label.
func ruleType(r orktypes.ValidationRule) string {
	if r.Equals != "" {
		return "equals"
	}
	if r.NotEquals != "" {
		return "notEquals"
	}
	if r.Prefix != "" {
		return "prefix"
	}
	if r.Suffix != "" {
		return "suffix"
	}
	if r.Contains != "" {
		return "contains"
	}
	if r.Min != "" {
		return "min"
	}
	if r.Max != "" {
		return "max"
	}
	if r.Operator != "" {
		return string(r.Operator)
	}
	return "exists"
}

// ─────────────────────────────────────────────────────────────
// High‑level state checks
// ─────────────────────────────────────────────────────────────

// HasWarnings returns true if any advisory warnings were produced.
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// Denied returns true if validation rules explicitly blocked reconciliation.
func (r *ValidationResult) Denied() bool {
	return r.Deny
}

// Blocked returns true if reconciliation must stop.
// Deny takes precedence; warnings alone do NOT block.
func (r *ValidationResult) Blocked() bool {
	return r.Deny
}

// ─────────────────────────────────────────────────────────────
// Error + message helpers
// ─────────────────────────────────────────────────────────────

// DenialError returns a proper error describing the denial reason.
func (r *ValidationResult) DenialError() error {
	if !r.Deny {
		return nil
	}
	return fmt.Errorf("validation denied: %s", r.DenialMessage())
}

// DenialMessage returns a human‑readable summary of the denial cause.
func (r *ValidationResult) DenialMessage() string {
	if !r.Deny {
		return ""
	}
	if len(r.Violations) == 0 {
		return "validation denied"
	}
	// First violation is usually the most relevant
	v := r.Violations[0]
	return fmt.Sprintf("field %q: %s (got %q)", v.Field, v.Message, v.Value)
}

// WarningSummary returns a semicolon‑joined list of warnings.
func (r *ValidationResult) WarningSummary() string {
	if len(r.Warnings) == 0 {
		return ""
	}
	return strings.Join(r.Warnings, "; ")
}

// HasViolations returns true if any rules failed.
func (r *ValidationResult) HasViolations() bool {
	return len(r.Violations) > 0
}

// ViolationSummary returns a semicolon‑joined list of violations.
func (r *ValidationResult) ViolationSummary() string {
	if len(r.Violations) == 0 {
		return ""
	}
	out := make([]string, 0, len(r.Violations))
	for _, v := range r.Violations {
		out = append(out, fmt.Sprintf("%s: %s", v.Field, v.Message))
	}
	return strings.Join(out, "; ")
}

// Log helper for debugging
func (r *ValidationResult) Log(crd, name, ns string) {
	if r.Passed {
		return
	}
	logger.Info().
		Str("crd", crd).
		Str("name", name).
		Str("namespace", ns).
		Int("violations", len(r.Violations)).
		Msg("validation failed")
}
