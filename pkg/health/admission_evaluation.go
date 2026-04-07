// health/admission_evaluation.go
package health

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ialexeze/orkestra/pkg/logger"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// ── Validation evaluation ─────────────────────────────────────────────────
//
// evaluateValidationRules runs all validation rules against an unstructured
// object map. Returns denials (action: deny) and warnings (action: warn)
// as separate slices so the handler can build the correct response.
//
// All rules are evaluated — not fail-fast — so the user sees all violations
// in one kubectl apply output rather than fixing one at a time.

// validationViolation describes one violated rule.
type validationViolation struct {
	Field    string
	Message  string
	Got      string // the actual field value that failed
	RuleType string // operator name (e.g. "prefix", "exists") for metrics
	Action   orktypes.ValidationAction
}

// evaluateValidationRules evaluates all rules and returns denials and warnings.
func (h *HealthServer) evaluateValidationRules(
	obj map[string]interface{},
	cfg *orktypes.ValidationConfig,
	kindName string,
) (denials []validationViolation, warnings []validationViolation) {
	for _, rule := range cfg.Rules {
		v := evaluateOneRule(obj, rule)
		if v == nil {
			continue // rule passed
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

// evaluateOneRule evaluates a single validation rule against the object.
// Returns nil when the rule passes, a violation when it fails.
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
	case orktypes.ConditionGt: // used for Min
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
	case orktypes.ConditionLt: // used for Max
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

	return nil // rule passed
}

// resolveValidationOperator extracts the effective operator and comparison
// value from a ValidationRule, preferring shorthands over the explicit form.
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

// ── Mutation evaluation ───────────────────────────────────────────────────
//
// applyMutationRules applies mutation rules to an already-copied object map.
// Returns the list of changes made. The caller diffs original vs mutated to
// build the JSON patch.
//
// Template expressions are resolved against the object being admitted —
// {{ .metadata.name }} resolves to the CR's name, {{ .spec.image }} resolves
// to the declared image, etc.

// applyMutationRules mutates obj in place and returns the list of changes.
func (h *HealthServer) applyMutationRules(
	ctx context.Context,
	obj map[string]interface{},
	cfg *orktypes.MutationConfig,
	kindName string,
) ([]fieldChange, error) {
	if cfg == nil || len(cfg.Rules) == 0 {
		return nil, nil
	}

	// Build a resolver from the object map
	// The same resolver used by conversion — resolves {{ .spec.X }} etc.
	resolver := orktmpl.NewResolverFromMap(obj)

	var changes []fieldChange

	for _, rule := range cfg.Rules {
		currentVal, found := resolveFieldPath(obj, rule.Field)

		var newVal string
		var changeType string

		switch {
		case rule.Override != nil && anyToString(rule.Override) != "":
			// Override — always apply regardless of current value
			resolved, err := resolver.Resolve(anyToString(rule.Override))
			if err != nil {
				return nil, fmt.Errorf("mutation rule override for field %q: %w", rule.Field, err)
			}
			newVal = resolved
			changeType = "override"

		case rule.Default != nil:
			// Default — apply only when field is absent or empty
			if found && currentVal != "" {
				continue // field already has a value — skip
			}
			resolved, err := resolver.Resolve(anyToString(rule.Default))
			if err != nil {
				return nil, fmt.Errorf("mutation rule default for field %q: %w", rule.Field, err)
			}
			newVal = resolved
			changeType = "default"

		default:
			continue // rule has neither Default nor Override — skip
		}

		// Skip if the value is already what we want to set
		if newVal == currentVal {
			continue
		}

		// Apply the change to the object map in-place
		setFieldPath(obj, rule.Field, newVal)

		changes = append(changes, fieldChange{
			Field:      rule.Field,
			OldValue:   currentVal,
			NewValue:   newVal,
			ChangeType: changeType,
		})

		logger.Debug().
			Str("kind", kindName).
			Str("field", rule.Field).
			Str("was", currentVal).
			Str("now", newVal).
			Str("type", changeType).
			Msg("admission/mutate: rule applied")
	}

	return changes, nil
}

// ── Field path helpers ────────────────────────────────────────────────────
//
// These are identical to the conversion logic field helpers.
// They live here too so this package is self-contained.
// In the final merge, share them via an internal package.

// resolveFieldPath resolves a dot-notation path against an object map.
// Returns the string value and whether the field was found.
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

// setFieldPath sets a value at a dot-notation path, creating intermediate
// maps as needed. Used by mutation to apply defaults in place.
func setFieldPath(obj map[string]interface{}, path string, value string) {
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

// anyToString converts an unstructured field value to its string form.
// Handles all types that Kubernetes JSON objects contain after decode.
func anyToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		// JSON numbers are float64 — integers should print without decimals
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
