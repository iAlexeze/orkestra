package template

import (

	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// evaluateConditions evaluates a when: block against the resolver's data map.
// All conditions AND together. Returns false on first failure.
// Called by ResolveStatusFields (same package) and exported for the reconciler.
func evaluateConditions(data map[string]interface{}, conditions []orktypes.Condition) bool {
	for _, cond := range conditions {
		if !orktypes.EvaluateOneCond(data, cond) {
			return false
		}
	}
	return true
}

