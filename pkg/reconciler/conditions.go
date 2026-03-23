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

//
// ──────────────────────────────────────────────────────────────────────────────
// Conditional Provisioning
// ──────────────────────────────────────────────────────────────────────────────
//
// Conditions allow resource templates (Deployments, Services, Secrets, etc.)
// to be applied only when the live Custom Resource matches a set of predicates.
//
// Example:
//
//   when:
//     - field: spec.environment
//       equals: production
//     - field: spec.enabled
//       equals: "true"
//
// All conditions are AND‑ed. If any condition fails, the resource is skipped for
// this reconcile cycle. This is not an error — it simply means “do not create or
// update this resource right now”. This enables dynamic topologies, feature
// flags, environment‑specific behavior, and conditional provisioning without
// writing Go code.
//
// Only unstructured CRDs support dot‑notation field evaluation. Typed CRDs
// always return true here — typed controllers should use Go hooks for conditional
// logic.
//

// EvaluateConditions reports whether *all* conditions pass for the given CR.
//
// - nil or empty slice → unconditional (true)
// - typed CRDs → always true (cannot evaluate dot‑notation paths)
// - unstructured CRDs → evaluate each condition using dot‑notation
//
// Any condition that fails causes the entire block to fail (AND semantics).
func EvaluateConditions(obj domain.Object, conditions []orktypes.Condition) bool {
	if len(conditions) == 0 {
		return true // no conditions → unconditional
	}

	// Only unstructured CRDs support dot‑notation field access.
	u, ok := toUnstructured(obj)
	if !ok {
		// Typed CRDs cannot evaluate conditions — do not silently skip resources.
		logger.Warn().
			Str("kind", obj.GetObjectKind().GroupVersionKind().Kind).
			Str("name", obj.GetName()).
			Msg("conditions: typed object cannot be evaluated — skipping conditions, resource will be created")
		return true
	}

	// Evaluate each condition.
	for _, cond := range conditions {
		if !evaluateOne(u, cond) {
			return false
		}
	}
	return true
}

// evaluateOne evaluates a single condition against an unstructured CR object.
// It resolves shorthands (equals:) before falling back to Operator/Value.
func evaluateOne(obj *unstructured.Unstructured, cond orktypes.Condition) bool {
	// Determine the effective operator and comparison value.
	op, val := resolveConditionOp(cond)

	// Resolve the field value from the CR using dot‑notation.
	fieldVal, found := resolveField(obj.Object, cond.Field)

	switch op {

	// ── Existence checks ─────────────────────────────────────────────────────
	case orktypes.ConditionExists:
		return found && fieldVal != ""

	case orktypes.ConditionNotExists:
		return !found || fieldVal == ""

	// ── String comparisons ───────────────────────────────────────────────────
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

	// ── Numeric comparisons ───────────────────────────────────────────────────
	case orktypes.ConditionGt, orktypes.ConditionLt:
		fv, err1 := strconv.ParseFloat(fieldVal, 64)
		cv, err2 := strconv.ParseFloat(val, 64)
		if err1 != nil || err2 != nil {
			logger.Warn().
				Str("field", cond.Field).
				Str("fieldVal", fieldVal).
				Str("condVal", val).
				Msg("conditions: numeric comparison requires numeric values — condition evaluated as false")
			return false
		}
		if op == orktypes.ConditionGt {
			return fv > cv
		}
		return fv < cv

	// ── Unknown operator ─────────────────────────────────────────────────────
	default:
		logger.Warn().
			Str("operator", string(op)).
			Str("field", cond.Field).
			Msg("conditions: unknown operator — condition evaluated as false")
		return false
	}
}

// resolveConditionOp determines the effective operator/value pair for a condition.
//
// Precedence:
//  1. equals: "x" shorthand
//  2. explicit operator + value
//  3. default operator (equals) when Value is set
//  4. default operator (exists) when nothing else is set
func resolveConditionOp(c orktypes.Condition) (orktypes.ConditionOperator, string) {
	// Shorthand takes precedence.
	if c.Equals != "" {
		return orktypes.ConditionEquals, c.Equals
	}

	// Explicit operator.
	if c.Operator != "" {
		return c.Operator, c.Value
	}

	// Default operator when a value is provided.
	if c.Value != "" {
		return orktypes.ConditionEquals, c.Value
	}

	// No operator and no value → treat as exists.
	return orktypes.ConditionExists, ""
}

// resolveField resolves a dot‑notation field path against an unstructured object.
// Returns the string representation of the value and whether it was found.
//
// Examples:
//
//	"spec.replicas" → obj["spec"]["replicas"]
//	"metadata.labels.tier" → obj["metadata"]["labels"]["tier"]
func resolveField(obj map[string]interface{}, path string) (string, bool) {
	parts := strings.Split(path, ".")
	current := obj

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return "", false
		}

		// Last segment → convert to string and return.
		if i == len(parts)-1 {
			return toStringVal(val), true
		}

		// Descend into nested map.
		next, ok := val.(map[string]interface{})
		if !ok {
			return "", false
		}
		current = next
	}

	return "", false
}

// toStringVal converts Kubernetes JSON values into strings.
// Handles string, bool, int64, float64, and falls back to fmt.Sprintf.
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
		// Kubernetes stores integers as float64 in JSON.
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// toUnstructured returns the object as *unstructured.Unstructured if possible.
// Typed CRDs cannot be evaluated via dot‑notation and return (nil, false).
func toUnstructured(obj domain.Object) (*unstructured.Unstructured, bool) {
	u, ok := obj.(*unstructured.Unstructured)
	return u, ok
}
