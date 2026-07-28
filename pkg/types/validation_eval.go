// pkg/types/validation_eval.go — ValidationRule evaluation, shared by the
// reconciler (every reconcile) and the admission webhook (admission time).
//
// This used to be two independently hand-maintained copies of the same
// switch statement, one in pkg/runtime/reconciler, one in
// pkg/gateway/webhook. That's how operator: in — defined for when:/anyOf:
// conditions — went unimplemented in validation.rules in both places at
// once, silently, without either side noticing: a rule using it always
// passed. One implementation means one place to fix, and no way for the two
// enforcement paths to drift out of sync with each other again.
package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/orkspace/orkestra/pkg/logger"
)

// TemplateResolver resolves a Go-template expression string against CR data
// and notes. Satisfied structurally by *pkg/resources/template.Resolver,
// which this package can't import directly — that package already imports
// pkg/types, so a reverse import would cycle.
type TemplateResolver interface {
	Resolve(value string) (string, error)
}

// RuleViolation is the outcome of one failed ValidationRule.
type RuleViolation struct {
	Field   string // the rule's original field expression (template or dot-path)
	Rule    string // effective operator name, e.g. "exists", "in", "gt"
	Value   string // the actual field value that failed
	Message string
	Action  ValidationAction
}

// ResolveScalarField resolves a dot-notation field path against a CR data
// map and returns its scalar string form, and whether it was found. Shared
// by validation-rule and mutation-rule evaluation, at both reconcile time
// and admission time.
func ResolveScalarField(data map[string]interface{}, path string) (string, bool) {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return "", false
		}
		if i == len(parts)-1 {
			return ScalarToString(val), true
		}
		next, ok := val.(map[string]interface{})
		if !ok {
			return "", false
		}
		current = next
	}
	return "", false
}

// ScalarToString converts a decoded Kubernetes JSON scalar (string, bool,
// float64 for numbers, or a pre-converted int64) into its string form.
func ScalarToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// ResolveValidationOp resolves the effective operator and comparison value
// from a ValidationRule, handling all shorthand fields. Falls back to an
// explicit operator+value pair, or a bare exists check when neither is set.
func ResolveValidationOp(r ValidationRule) (ConditionOperator, string) {
	switch {
	case r.Equals != "":
		return ConditionEquals, r.Equals
	case r.NotEquals != "":
		return ConditionNotEquals, r.NotEquals
	case r.Prefix != "":
		return ConditionPrefix, r.Prefix
	case r.Suffix != "":
		return ConditionSuffix, r.Suffix
	case r.Contains != "":
		return ConditionContains, r.Contains
	case r.Min != "":
		return ConditionGt, r.Min
	case r.Max != "":
		return ConditionLt, r.Max
	case r.GreaterThan != "":
		return ConditionGt, r.GreaterThan
	case r.LessThan != "":
		return ConditionLt, r.LessThan
	case r.Operator != "":
		return r.Operator, r.Value
	default:
		return ConditionExists, ""
	}
}

// RuleTypeLabel returns a short string identifying a ValidationRule's
// effective comparison for use as a low-cardinality metric label. Prefers
// the shorthand name the katalog author actually wrote (e.g. "min", "max")
// over the operator it resolves to (both "min" and "greaterThan" resolve to
// ConditionGt), since that's the more actionable label for "which rule type
// is causing the most friction" alerting.
func RuleTypeLabel(r ValidationRule) string {
	switch {
	case r.Equals != "":
		return "equals"
	case r.NotEquals != "":
		return "notEquals"
	case r.Prefix != "":
		return "prefix"
	case r.Suffix != "":
		return "suffix"
	case r.Contains != "":
		return "contains"
	case r.Min != "":
		return "min"
	case r.Max != "":
		return "max"
	case r.GreaterThan != "":
		return "greaterThan"
	case r.LessThan != "":
		return "lessThan"
	case r.Operator != "":
		return string(r.Operator)
	default:
		return "exists"
	}
}

// inCommaList reports whether value equals one of expected's comma-separated
// entries (each trimmed of surrounding whitespace).
func inCommaList(value, expected string) bool {
	for _, v := range strings.Split(expected, ",") {
		if strings.TrimSpace(v) == value {
			return true
		}
	}
	return false
}

// EvaluateValidationRule evaluates one ValidationRule against CR data and
// returns the violation if it fails, nil if it passes.
//
// resolver may be nil — the reconciler runs without one when no resolver
// context is available; template expressions in field/value/message are
// then left unresolved and matched against the raw value instead. Callers
// passing a possibly-nil concrete resolver type must nil-check before
// converting to this interface, since a typed nil wrapped in an interface
// is not itself nil.
func EvaluateValidationRule(data map[string]interface{}, resolver TemplateResolver, rule ValidationRule) *RuleViolation {
	op, expected := ResolveValidationOp(rule)

	// Resolve template expressions in comparison values and messages.
	// Notes are called by name directly: {{ inBusinessHours }}, {{ allowedRegistry }}.
	if resolver != nil && IsTemplate(expected) {
		if resolved, err := resolver.Resolve(expected); err == nil {
			expected = resolved
		}
	}
	message := rule.Message
	if resolver != nil && IsTemplate(message) {
		if resolved, err := resolver.Resolve(message); err == nil {
			message = resolved
		}
	}

	// displayField is always the original expression — shown in violation
	// messages. When field is a template, the resolved value is the result
	// of the expression directly (not a CR path), so we skip
	// ResolveScalarField and use it as fieldVal.
	displayField := rule.Field
	isTemplate := IsTemplate(rule.Field)

	var fieldVal string
	var found bool
	if isTemplate && resolver != nil {
		fieldVal, _ = resolver.Resolve(rule.Field)
		found = fieldVal != ""
	} else {
		fieldVal, found = ResolveScalarField(data, rule.Field)
	}

	fail := func() *RuleViolation {
		return &RuleViolation{
			Field:   displayField,
			Rule:    string(op),
			Value:   fieldVal,
			Message: message,
			Action:  rule.Action,
		}
	}

	switch op {
	case ConditionExists:
		if !found || fieldVal == "" {
			return fail()
		}

	case ConditionNotExists:
		if found && fieldVal != "" {
			return fail()
		}

	case ConditionEquals:
		if !found || fieldVal != expected {
			return fail()
		}

	case ConditionNotEquals:
		if found && fieldVal == expected {
			return fail()
		}

	case ConditionContains:
		if !found || !strings.Contains(fieldVal, expected) {
			return fail()
		}

	case ConditionPrefix:
		if !found || !strings.HasPrefix(fieldVal, expected) {
			return fail()
		}

	case ConditionSuffix:
		if !found || !strings.HasSuffix(fieldVal, expected) {
			return fail()
		}

	case ConditionIn:
		if !found || !inCommaList(fieldVal, expected) {
			return fail()
		}

	case ConditionGt: // used as Min when coming from rule.Min
		cv, err := strconv.ParseFloat(expected, 64)
		if err != nil {
			logger.Warn().Str("field", rule.Field).Str("val", expected).
				Msg("validation: min/gt requires a numeric value — rule skipped")
			return nil
		}
		fv, err := strconv.ParseFloat(fieldVal, 64)
		if err != nil || fv < cv {
			return fail()
		}

	case ConditionLt: // used as Max when coming from rule.Max
		cv, err := strconv.ParseFloat(expected, 64)
		if err != nil {
			logger.Warn().Str("field", rule.Field).Str("val", expected).
				Msg("validation: max/lt requires a numeric value — rule skipped")
			return nil
		}
		fv, err := strconv.ParseFloat(fieldVal, 64)
		if err != nil || fv > cv {
			return fail()
		}
	}

	return nil // rule passed
}
