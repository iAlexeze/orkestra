// webhook/admission_evaluation.go — validation and mutation rule evaluation.
package webhook

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/orkspace/orkestra/pkg/logger"
	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ── Validation evaluation ─────────────────────────────────────────────────────

type validationViolation struct {
	Field    string
	Message  string
	Got      string
	RuleType string
	Action   orktypes.ValidationAction
}

func (ws *WebhookServer) evaluateValidationRules(
	obj map[string]interface{},
	cfg *orktypes.ValidationConfig,
	kindName string,
) (denials []validationViolation, warnings []validationViolation) {
	for _, rule := range cfg.Rules {
		v := evaluateOneRule(obj, rule)
		if v == nil {
			continue
		}
		switch orktypes.EffectiveAction(rule.Action) {
		case orktypes.ValidationActionDeny:
			denials = append(denials, *v)
		case orktypes.ValidationActionWarn:
			warnings = append(warnings, *v)
		}
	}
	return
}

func evaluateOneRule(obj map[string]interface{}, rule orktypes.ValidationRule) *validationViolation {
	op, expected := resolveValidationOperator(rule)
	fieldVal, found := resolveFieldPath(obj, rule.Field)

	fail := func() *validationViolation {
		return &validationViolation{
			Field:    rule.Field,
			Message:  rule.Message,
			Got:      fieldVal,
			RuleType: string(op),
			Action:   rule.Action,
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
	case orktypes.ConditionGt:
		cv, err := strconv.ParseFloat(expected, 64)
		if err != nil {
			logger.Warn().Str("field", rule.Field).Str("min", expected).
				Msg("admission/validate: min value is not numeric — rule skipped")
			return nil
		}
		fv, err := strconv.ParseFloat(fieldVal, 64)
		if err != nil || fv < cv {
			return fail()
		}
	case orktypes.ConditionLt:
		cv, err := strconv.ParseFloat(expected, 64)
		if err != nil {
			logger.Warn().Str("field", rule.Field).Str("max", expected).
				Msg("admission/validate: max value is not numeric — rule skipped")
			return nil
		}
		fv, err := strconv.ParseFloat(fieldVal, 64)
		if err != nil || fv > cv {
			return fail()
		}
	}

	return nil
}

func resolveValidationOperator(r orktypes.ValidationRule) (orktypes.ConditionOperator, string) {
	switch {
	case r.Equals != "":
		return orktypes.ConditionEquals, r.Equals
	case r.NotEquals != "":
		return orktypes.ConditionNotEquals, r.NotEquals
	case r.Prefix != "":
		return orktypes.ConditionPrefix, r.Prefix
	case r.Suffix != "":
		return orktypes.ConditionSuffix, r.Suffix
	case r.Contains != "":
		return orktypes.ConditionContains, r.Contains
	case r.GreaterThan != "":
		return orktypes.ConditionGt, r.GreaterThan
	case r.LessThan != "":
		return orktypes.ConditionLt, r.LessThan
	case r.Min != "":
		return orktypes.ConditionGt, r.Min
	case r.Max != "":
		return orktypes.ConditionLt, r.Max
	case r.Operator != "":
		return r.Operator, r.Value
	default:
		return orktypes.ConditionExists, ""
	}
}

// ── Mutation evaluation ───────────────────────────────────────────────────────

func (ws *WebhookServer) applyMutationRules(
	ctx context.Context,
	obj map[string]interface{},
	cfg *orktypes.MutationConfig,
	kindName string,
) ([]fieldChange, error) {
	if cfg == nil || len(cfg.Rules) == 0 {
		return nil, nil
	}

	resolver := orktmpl.NewResolverFromMap(obj)
	var changes []fieldChange

	for _, rule := range cfg.Rules {
		// Resolve raw value (string from template or static value) first
		var rawResolved string
		var changeType string
		var err error

		switch {
		case rule.Override != nil && anyToString(rule.Override) != "":
			raw, err := resolver.Resolve(anyToString(rule.Override))
			if err != nil {
				return nil, fmt.Errorf("mutation rule override for field %q: %w", rule.Field, err)
			}
			rawResolved = raw
			changeType = "override"

		case rule.Default != nil:
			currentVal, found := resolveFieldPath(obj, rule.Field)
			if found && currentVal != "" {
				continue // already set, skip default
			}
			raw, err := resolver.Resolve(anyToString(rule.Default))
			if err != nil {
				return nil, fmt.Errorf("mutation rule default for field %q: %w", rule.Field, err)
			}
			rawResolved = raw
			changeType = "default"

		default:
			continue
		}

		// Convert to the target type based on valueType
		typedVal, err := convertToType(rawResolved, rule.ValueType)
		if err != nil {
			logger.Error().Err(err).Str("field", rule.Field).Str("valueType", rule.ValueType).Msg("admission/mutate: type conversion failed")
			continue // skip this field instead of failing the whole admission
		}

		// Compare with current value (as string for simplicity)
		currentVal, _ := resolveFieldPath(obj, rule.Field)
		if fmt.Sprintf("%v", typedVal) == currentVal {
			continue // unchanged
		}

		// Apply the typed value to the object
		setFieldPath(obj, rule.Field, typedVal)

		changes = append(changes, fieldChange{
			Field:      rule.Field,
			OldValue:   currentVal,
			NewValue:   fmt.Sprintf("%v", typedVal),
			TypedValue: typedVal,
			ChangeType: changeType,
		})

		logger.Debug().
			Str("kind", kindName).
			Str("field", rule.Field).
			Str("was", currentVal).
			Str("now", fmt.Sprintf("%v", typedVal)).
			Str("type", changeType).
			Msg("admission/mutate: rule applied")
	}

	return changes, nil
}

// convertToType converts a string value to the requested type.
// Supported valueType: "int", "integer", "bool", "boolean", "float", "number", "string" (default).
// Returns the typed value (int64, bool, float64, or string) suitable for JSON patch.
func convertToType(val string, valueType string) (interface{}, error) {
	switch valueType {
	case "int", "integer":
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to int: %w", val, err)
		}
		return i, nil
	case "bool", "boolean":
		b, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to bool: %w", val, err)
		}
		return b, nil
	case "float", "number":
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to float: %w", val, err)
		}
		return f, nil
	default: // "string" or empty
		return val, nil
	}
}

// ── Field path helpers ────────────────────────────────────────────────────────

func resolveFieldPath(obj map[string]interface{}, path string) (string, bool) {
	parts := strings.Split(path, ".")
	current := obj

	for i, part := range parts {
		raw, ok := current[part]
		if !ok || raw == nil {
			return "", false
		}
		if i == len(parts)-1 {
			return anyToString(raw), true
		}
		next, ok := raw.(map[string]interface{})
		if !ok {
			return "", false
		}
		current = next
	}
	return "", false
}

func setFieldPath(obj map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := obj

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}
		if _, ok := current[part]; !ok {
			current[part] = map[string]interface{}{}
		}
		next, ok := current[part].(map[string]interface{})
		if !ok {
			next = map[string]interface{}{}
			current[part] = next
		}
		current = next
	}
}

func anyToString(v interface{}) string {
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
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", val)
	}
}
