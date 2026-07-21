// pkg/reconciler/conditions.go
package reconciler

import (
	"fmt"
	"strconv"
	"strings"
)

// resolveField resolves a dot-notation field path against a CR data map.
// Returns the string representation of the value and whether it was found.
func resolveField(obj map[string]interface{}, path string) (string, bool) {
	parts := strings.Split(path, ".")
	current := obj

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return "", false
		}
		if i == len(parts)-1 {
			return toStringVal(val), true
		}
		next, ok := val.(map[string]interface{})
		if !ok {
			return "", false
		}
		current = next
	}
	return "", false
}

// toStringVal converts Kubernetes JSON values into strings.
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
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", val)
	}
}
