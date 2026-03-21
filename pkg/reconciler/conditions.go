// pkg/reconciler/conditions.go
package reconciler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// EvaluateConditions evaluates a slice of conditions against the CR object.
// All conditions are AND'd — returns true only if every condition passes.
// Returns true when the When slice is empty (no conditions = always create).
func EvaluateConditions(obj domain.Object, conditions []orktypes.Condition) bool {
	if len(conditions) == 0 {
		return true // no conditions → unconditional
	}

	u, ok := toUnstructured(obj)
	if !ok {
		// Typed objects cannot be evaluated via dot-notation field paths.
		// Fall back to true — don't silently skip resources for typed CRDs.
		logger.Warn().Msg("conditions: typed object cannot be evaluated — skipping conditions, resource will be created")
		return true
	}

	for _, cond := range conditions {
		if !evaluateOne(u, cond) {
			return false
		}
	}
	return true
}

// evaluateOne evaluates a single condition against an unstructured object.
func evaluateOne(obj *unstructured.Unstructured, cond orktypes.Condition) bool {
	// Resolve operator — shorthands take precedence over Operator field
	op, val := resolveConditionOp(cond)

	// Resolve the field value from the CR
	fieldVal, found := resolveField(obj.Object, cond.Field)

	switch op {
	case orktypes.ConditionExists:
		return found && fieldVal != ""

	case orktypes.ConditionNotExists:
		return !found || fieldVal == ""

	case orktypes.ConditionEquals:
		return found && fieldVal == val

	case orktypes.ConditionNotEquals:
		return !found || fieldVal != val

	case orktypes.ConditionContains:
		return found && strings.Contains(fieldVal, val)

	case orktypes.ConditionPrefix:
		return found && strings.HasPrefix(fieldVal, val)

	case orktypes.ConditionSuffix:
		return found && strings.HasSuffix(fieldVal, val)

	case orktypes.ConditionGt:
		fv, err1 := strconv.ParseFloat(fieldVal, 64)
		cv, err2 := strconv.ParseFloat(val, 64)
		if err1 != nil || err2 != nil {
			logger.Warn().
				Str("field", cond.Field).
				Str("fieldVal", fieldVal).
				Str("condVal", val).
				Msg("conditions: gt requires numeric values — condition skipped (false)")
			return false
		}
		return fv > cv

	case orktypes.ConditionLt:
		fv, err1 := strconv.ParseFloat(fieldVal, 64)
		cv, err2 := strconv.ParseFloat(val, 64)
		if err1 != nil || err2 != nil {
			logger.Warn().
				Str("field", cond.Field).
				Str("fieldVal", fieldVal).
				Str("condVal", val).
				Msg("conditions: lt requires numeric values — condition skipped (false)")
			return false
		}
		return fv < cv

	default:
		logger.Warn().
			Str("operator", string(op)).
			Str("field", cond.Field).
			Msg("conditions: unknown operator — condition skipped (false)")
		return false
	}
}

// resolveConditionOp returns the effective operator and value for a condition,
// resolving shorthands (Equals field) before falling back to Operator.
func resolveConditionOp(c orktypes.Condition) (orktypes.ConditionOperator, string) {
	// Shorthand takes precedence over explicit Operator field
	if c.Equals != "" {
		return orktypes.ConditionEquals, c.Equals
	}
	if c.Operator == "" {
		// Default operator is equals when Value is set
		if c.Value != "" {
			return orktypes.ConditionEquals, c.Value
		}
		return orktypes.ConditionExists, ""
	}
	return c.Operator, c.Value
}

// resolveField resolves a dot-notation field path against an object map.
// Returns the string representation of the value and whether it was found.
//
// Examples:
//
//	"spec.environment"         → obj["spec"]["environment"]
//	"metadata.labels.tier"    → obj["metadata"]["labels"]["tier"]
//	"spec.replicas"            → converts int64/float64 to string
func resolveField(obj map[string]interface{}, path string) (string, bool) {
	parts := strings.Split(path, ".")
	current := obj

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return "", false
		}

		// If this is the last part, convert to string and return
		if i == len(parts)-1 {
			return toStringVal(val), true
		}

		// Otherwise, recurse into the next level
		next, ok := val.(map[string]interface{})
		if !ok {
			return "", false // path continues but value is not a map
		}
		current = next
	}

	return "", false
}

// toStringVal converts an interface{} field value to a string.
// Handles the types that Kubernetes unstructured objects contain.
func toStringVal(v interface{}) string {
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
		// Kubernetes stores integers as float64 in JSON
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// toUnstructured attempts to return the object as *unstructured.Unstructured.
// Typed objects cannot be evaluated via dot-notation paths.
func toUnstructured(obj domain.Object) (*unstructured.Unstructured, bool) {
	u, ok := obj.(*unstructured.Unstructured)
	return u, ok
}
