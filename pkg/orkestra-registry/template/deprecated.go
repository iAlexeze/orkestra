package template

import (
	"fmt"
	"strings"

	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// 
// Deprecated. Now handled from pkg/types/when to avoid repetition
func evaluateOneCondition(data map[string]interface{}, cond orktypes.Condition) bool {
	fieldVal := resolveNestedField(data, cond.Field)
	op, expected := resolveConditionOp(cond)

	switch op {
	case orktypes.ConditionExists:
		return fieldVal != "" && fieldVal != "<no value>"
	case orktypes.ConditionNotExists:
		return fieldVal == "" || fieldVal == "<no value>"
	case orktypes.ConditionEquals:
		return fieldVal == expected
	case orktypes.ConditionNotEquals:
		return fieldVal != expected
	case orktypes.ConditionContains:
		return strings.Contains(fieldVal, expected)
	case orktypes.ConditionPrefix:
		return strings.HasPrefix(fieldVal, expected)
	case orktypes.ConditionSuffix:
		return strings.HasSuffix(fieldVal, expected)
	case orktypes.ConditionGt:
		fv, fe := parseNumeric(fieldVal)
		ev, ee := parseNumeric(expected)
		if fe != nil || ee != nil {
			return false
		}
		return fv > ev
	case orktypes.ConditionLt:
		fv, fe := parseNumeric(fieldVal)
		ev, ee := parseNumeric(expected)
		if fe != nil || ee != nil {
			return false
		}
		return fv < ev
	case orktypes.ConditionIn:
		for _, v := range strings.Split(expected, ",") {
			if strings.TrimSpace(v) == fieldVal {
				return true
			}
		}
		return false
	}
	return false
}

// resolveNestedField navigates a dot-notation path through a map.
//
//	"status.phase"                   → m["status"]["phase"]
//	"children.job.status.succeeded"  → m["children"]["job"]["status"]["succeeded"]
//
// Returns "" when any segment is missing — this is the notExists case.
// Never panics: type assertion failures return "" rather than crashing.
func resolveNestedField(m map[string]interface{}, path string) string {
	var current interface{} = m
	for _, part := range strings.Split(path, ".") {
		typed, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current, ok = typed[part]
		if !ok {
			return ""
		}
	}
	if current == nil {
		return ""
	}
	return fmt.Sprintf("%v", current)
}

func resolveConditionOp(c orktypes.Condition) (orktypes.ConditionOperator, string) {
	if c.Equals != "" {
		return orktypes.ConditionEquals, c.Equals
	}
	if c.NotEquals != "" {
		return orktypes.ConditionNotEquals, c.NotEquals
	}
	if c.Prefix != "" {
		return orktypes.ConditionPrefix, c.Prefix
	}
	if c.Suffix != "" {
		return orktypes.ConditionSuffix, c.Suffix
	}
	if c.Contains != "" {
		return orktypes.ConditionContains, c.Contains
	}
	if c.GreaterThan != "" {
		return orktypes.ConditionGt, c.GreaterThan
	}
	if c.LessThan != "" {
		return orktypes.ConditionLt, c.LessThan
	}
	if c.Operator != "" {
		return c.Operator, c.Value
	}
	if c.Value != "" {
		return orktypes.ConditionEquals, c.Value
	}
	return orktypes.ConditionExists, ""
}


func parseNumeric(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// evaluateStatusConditions evaluates a list of Condition against the object map.
// All conditions must pass (AND semantics).
//
// The objMap comes from resolver.Data() — it includes .spec.*, .status.*,
// .metadata.*, and .children.* (when WithChildren has been called).
func evaluateStatusConditions(objMap map[string]interface{}, conditions []orktypes.Condition) bool {
	for _, cond := range conditions {
		if !evaluateOneStatusCondition(objMap, cond) {
			return false
		}
	}
	return true
}

// evaluateOneStatusCondition evaluates a single Condition against the object map.
//
// Operators:
//   - exists:    field is non-empty and not "<no value>"
//   - notExists: field is absent or empty (first-reconcile detection)
//   - equals:    exact string match
//   - notEquals: string mismatch
//   - contains:  substring match
//   - hasPrefix: prefix match
//   - hasSuffix: suffix match
//   - gt:        numeric greater-than (for child succeeded/failed counts)
//   - lt:        numeric less-than
//   - in:        membership in comma-separated list; empty string matches ""
func evaluateOneStatusCondition(objMap map[string]interface{}, cond orktypes.Condition) bool {
	fieldVal := orktypes.NavigateDotPath(objMap, cond.Field)
	op, expected := orktypes.ResolveConditionOp(cond)

	switch op {
	case orktypes.ConditionExists:
		return fieldVal != "" && fieldVal != "<no value>"

	case orktypes.ConditionNotExists:
		return fieldVal == "" || fieldVal == "<no value>"

	case orktypes.ConditionEquals:
		return fieldVal == expected

	case orktypes.ConditionNotEquals:
		return fieldVal != expected

	case orktypes.ConditionContains:
		return strings.Contains(fieldVal, expected)

	case orktypes.ConditionPrefix:
		return strings.HasPrefix(fieldVal, expected)

	case orktypes.ConditionSuffix:
		return strings.HasSuffix(fieldVal, expected)

	case orktypes.ConditionGt:
		fv, fe := parseNumeric(fieldVal)
		ev, ee := parseNumeric(expected)
		if fe != nil || ee != nil {
			return false
		}
		return fv > ev

	case orktypes.ConditionLt:
		fv, fe := parseNumeric(fieldVal)
		ev, ee := parseNumeric(expected)
		if fe != nil || ee != nil {
			return false
		}
		return fv < ev

	case orktypes.ConditionIn:
		// Comma-separated list — empty string in list matches absent field.
		// "Pending," matches when phase is "Pending" OR not yet written.
		for _, v := range strings.Split(expected, ",") {
			if strings.TrimSpace(v) == fieldVal {
				return true
			}
		}
		return false
	}

	return false
}

// setNestedStatus writes a value at a dot-notation path in the result map,
// creating intermediate maps as needed.
//
//	"phase"               → result["phase"] = value
//	"database.host"       → result["database"]["host"] = value
//	"cloud.provider.name" → result["cloud"]["provider"]["name"] = value
func setNestedStatus(result map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := result

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
