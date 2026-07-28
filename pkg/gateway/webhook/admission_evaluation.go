// webhook/admission_evaluation.go — validation and mutation rule evaluation.
package webhook

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	orkexternal "github.com/orkspace/orkestra/pkg/external"
	"github.com/orkspace/orkestra/pkg/logger"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
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
	ctx context.Context,
	obj map[string]interface{},
	cfg *orktypes.ValidationConfig,
	kindName string,
) (denials []validationViolation, warnings []validationViolation) {
	resolver := orktmpl.NewResolverFromMap(obj)
	if ws.katalog != nil {
		resolver = resolver.WithUserNotes(ws.katalog.Notes)
	}
	if calls := cfg.AdmissionExternal(); len(calls) > 0 {
		var err error
		resolver, err = orkexternal.Run(ctx, kindName, resolver, calls, ws.kubeClient)
		if err != nil {
			logger.FromContext(ctx).Warn().Err(err).Str("kind", kindName).Msg("admission/validate: external call failed")
		}
	}
	data := resolver.Data()
	for _, rule := range cfg.Rules {
		if !orktypes.EvaluateWhen(data, rule.When, rule.AnyOf, resolver.TemplateEvaluator()) {
			continue
		}
		rv := orktypes.EvaluateValidationRule(data, resolver, rule)
		if rv == nil {
			continue
		}
		v := &validationViolation{
			Field:    rv.Field,
			Message:  rv.Message,
			Got:      rv.Value,
			RuleType: rv.Rule,
			Action:   rule.Action,
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
	if ws.katalog != nil {
		resolver = resolver.WithUserNotes(ws.katalog.Notes)
	}
	if calls := cfg.AdmissionExternal(); len(calls) > 0 {
		var err error
		resolver, err = orkexternal.Run(ctx, kindName, resolver, calls, ws.kubeClient)
		if err != nil {
			logger.FromContext(ctx).Warn().Err(err).Str("kind", kindName).Msg("admission/mutate: external call failed")
		}
	}
	var changes []fieldChange

	mdata := resolver.Data()
	for _, rule := range cfg.Rules {
		if !orktypes.EvaluateWhen(mdata, rule.When, rule.AnyOf, resolver.TemplateEvaluator()) {
			continue
		}

		// Resolve template expression in the field path.
		targetField := rule.Field
		if orktypes.IsTemplate(targetField) {
			if resolved, err := resolver.Resolve(targetField); err == nil {
				targetField = resolved
			}
		}

		// Resolve raw value (string from template or static value) first
		var rawResolved string
		var changeType string
		var err error

		switch {
		case rule.Override != nil && orktypes.ScalarToString(rule.Override) != "":
			raw, err := resolver.Resolve(orktypes.ScalarToString(rule.Override))
			if err != nil {
				return nil, fmt.Errorf("mutation rule override for field %q: %w", targetField, err)
			}
			rawResolved = raw
			changeType = "override"

		case rule.Default != nil:
			currentVal, found := orktypes.ResolveScalarField(obj, targetField)
			if found && currentVal != "" {
				continue // already set, skip default
			}
			raw, err := resolver.Resolve(orktypes.ScalarToString(rule.Default))
			if err != nil {
				return nil, fmt.Errorf("mutation rule default for field %q: %w", targetField, err)
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
		currentVal, _ := orktypes.ResolveScalarField(obj, targetField)
		if fmt.Sprintf("%v", typedVal) == currentVal {
			continue // unchanged
		}

		// Apply the typed value to the object
		setFieldPath(obj, targetField, typedVal)

		changes = append(changes, fieldChange{
			Field:      targetField,
			OldValue:   currentVal,
			NewValue:   fmt.Sprintf("%v", typedVal),
			TypedValue: typedVal,
			ChangeType: changeType,
		})

		logger.Debug().
			Str("kind", kindName).
			Str("field", targetField).
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
