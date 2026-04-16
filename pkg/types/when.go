// pkg/types/when.go
//
// EvaluateWhen — extended condition evaluation with OR logic.
//
// The when: field ([]Condition) uses AND semantics.
// anyOf: is a new parallel field on template sources with OR semantics.
//
//	# AND only
//	when:
//	  - field: status.phase
//	    equals: "Ready"
//
//	# OR
//	anyOf:
//	  - field: status.phase
//	    equals: "Failed"
//	  - field: status.phase
//	    equals: "Succeeded"
//
//	# Combined: (spec.enabled=true) AND (phase=Failed OR phase=Succeeded)
//	when:
//	  - field: spec.enabled
//	    equals: "true"
//	anyOf:
//	  - field: status.phase
//	    equals: "Failed"
//	  - field: status.phase
//	    equals: "Succeeded"
package types

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/note"
)

// EvaluateWhen evaluates when: (allOf, AND) and anyOf: (OR) conditions.
// data is resolver.Data() — full CR map including children, external, cross.
//
// Both blocks must pass when both are declared.
// Empty blocks always pass.
func EvaluateWhen(data map[string]interface{}, allOf []Condition, anyOf []Condition) bool {
	for _, cond := range allOf {
		if !EvaluateOneCond(data, cond) {
			return false
		}
	}
	if len(anyOf) > 0 {
		passed := false
		for _, cond := range anyOf {
			if EvaluateOneCond(data, cond) {
				passed = true
				break
			}
		}
		if !passed {
			return false
		}
	}
	return true
}

// EvaluateOneCond evaluates a single Condition against a data map.
// Exported so the template package and reconciler package can both call it.
// Defined here in pkg/types to avoid import cycles.
func EvaluateOneCond(data map[string]interface{}, cond Condition) bool {
	fieldVal := NavigateDotPath(data, cond.Field)

	// Numeric absent-field fix: Kubernetes omits zero-value integers.
	// An absent count field is semantically 0.
	op, expected := ResolveConditionOp(cond)

	switch op {
	case ConditionExists:
		return fieldVal != "" && fieldVal != "<no value>"
	case ConditionNotExists:
		return fieldVal == "" || fieldVal == "<no value>"
	case ConditionEquals:
		return fieldVal == expected
	case ConditionNotEquals:
		return fieldVal != expected
	case ConditionContains:
		return typeContains(fieldVal, expected)
	case ConditionPrefix:
		return typeHasPrefix(fieldVal, expected)
	case ConditionSuffix:
		return typeHasSuffix(fieldVal, expected)
	case ConditionGt:
		fv, _ := typeParseFloat(fieldVal) // absent = 0
		ev, ee := typeParseFloat(expected)
		if ee != nil {
			return false
		}
		return fv > ev
	case ConditionLt:
		fv, _ := typeParseFloat(fieldVal)
		ev, ee := typeParseFloat(expected)
		if ee != nil {
			return false
		}
		return fv < ev
	case ConditionIn:
		for _, v := range typesSplitComma(expected) {
			if typesTrimSpace(v) == fieldVal {
				return true
			}
		}
		return false
	case ConditionUnique:
		// Unique operator is only meaningful in validation context
		// (needs informer access). In when: blocks it always passes.
		return true
	case ConditionTypeOf:
		raw := NavigateRawPath(data, cond.Field) // returns interface{}, not string
		return note.TypeOf(raw) == expected
	case ConditionTypeMap:
		raw := NavigateRawPath(data, cond.Field)
		return note.TypeOf(raw) == "map"

	case ConditionTypeList:
		raw := NavigateRawPath(data, cond.Field)
		return note.TypeOf(raw) == "slice"

	case ConditionTypeString:
		raw := NavigateRawPath(data, cond.Field)
		return note.TypeOf(raw) == "string"

	case ConditionTypeNumber:
		raw := NavigateRawPath(data, cond.Field)
		return note.TypeOf(raw) == "number"

	case ConditionTypeBool:
		raw := NavigateRawPath(data, cond.Field)
		return note.TypeOf(raw) == "bool"

	case ConditionTypeNull:
		raw := NavigateRawPath(data, cond.Field)
		return note.TypeOf(raw) == "null"

	}
	return false
}

// NavigateRawPath walks a dot-notation path through a nested map.
// Returns interface{} when any segment is missing — the notExists case.
// Exported for use by the template package and status field resolver.
func NavigateRawPath(m map[string]interface{}, path string) interface{} {
	// Empty path check
	if path == "" {
		return nil
	}

	// Start at the root with 'current' as cursor
	var current interface{} = m

	// Split the path into parts(slice)
	for _, part := range typesSplitDot(path) {
		// Ensure current is a map
		typed, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		// Follow the path
		current, ok = typed[part]
		if !ok {
			return nil
		}
	}
	if current == nil {
		return nil
	}

	return current
}

// NavigateDotPath walks a dot-notation path through a nested map.
// Returns "" when any segment is missing — the notExists case.
// Exported for use by the template package and status field resolver.
func NavigateDotPath(m map[string]interface{}, path string) string {
	// Empty path check
	if path == "" {
		return ""
	}

	// Start at the root with 'current' as cursor
	var current interface{} = m

	// Split the path into parts(slice)
	for _, part := range typesSplitDot(path) {
		// Ensure current is a map
		typed, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		// Follow the path
		current, ok = typed[part]
		if !ok {
			return ""
		}
	}
	if current == nil {
		return ""
	}
	if s, ok := current.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", current)
}

// ResolveConditionOp resolves the effective operator and comparison value
// from a Condition, respecting shorthand fields.
// Exported so the template package can use the same resolution logic.
func ResolveConditionOp(c Condition) (ConditionOperator, string) {
	if c.Equals != "" {
		return ConditionEquals, c.Equals
	}
	if c.NotEquals != "" {
		return ConditionNotEquals, c.NotEquals
	}
	if c.Prefix != "" {
		return ConditionPrefix, c.Prefix
	}
	if c.Suffix != "" {
		return ConditionSuffix, c.Suffix
	}
	if c.Contains != "" {
		return ConditionContains, c.Contains
	}
	if c.GreaterThan != "" {
		return ConditionGt, c.GreaterThan
	}
	if c.LessThan != "" {
		return ConditionLt, c.LessThan
	}
	if c.Operator != "" {
		return c.Operator, c.Value
	}
	if c.Value != "" {
		return ConditionEquals, c.Value
	}
	return ConditionExists, ""
}

// ── Private string/numeric helpers ───────────────────────────────────────────

func typesSplitDot(s string) []string   { return typesSplitOn(s, '.') }
func typesSplitComma(s string) []string { return typesSplitOn(s, ',') }

func typesSplitOn(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return append(parts, s[start:])
}

func typesTrimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func typeContains(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func typeHasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }
func typeHasSuffix(s, p string) bool { return len(s) >= len(p) && s[len(s)-len(p):] == p }

// typeParseFloat treats empty string as 0 — Kubernetes omits zero-value integers.
func typeParseFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
