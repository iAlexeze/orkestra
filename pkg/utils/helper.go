package utils

import "strings"

// ToStringSet converts a slice of strings to a map[string]bool for O(1) lookups.
//
// Useful for:
//   - Permission checks (e.g., "is this operation in the allowed list?")
//   - Deduplication checks
//   - Set operations (intersection, subset, difference)
//
// Returns an empty map when ops is nil or empty.
func ToStringSet(ops []string) map[string]bool {
	s := make(map[string]bool, len(ops))
	for _, op := range ops {
		s[op] = true
	}
	return s
}

// NestedSlice navigates a map[string]interface{} using dot-notation keys
// and returns the final slice value.
// Returns nil, false if any key in the path is missing or not a map.
func NestedSlice(obj map[string]interface{}, keys ...string) ([]interface{}, bool) {
	cur := obj
	for i, k := range keys {
		if i == len(keys)-1 {
			v, ok := cur[k].([]interface{})
			return v, ok
		}
		next, ok := cur[k].(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur = next
	}
	return nil, false
}

// NestedMap navigates a map[string]interface{} using dot-notation keys
// and returns the final map value.
// Returns nil, false if any key in the path is missing or not a map.
func NestedMap(obj map[string]interface{}, keys ...string) (map[string]interface{}, bool) {
	cur := obj
	for _, k := range keys {
		next, ok := cur[k].(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

// DeleteNestedPath removes a dot-notation path from a nested map in place.
// Silently does nothing when the path does not exist — partial paths are not
// errors. Supports arbitrary depth: "metadata.managedFields",
// "status.observedGeneration", "metadata.annotations.internal-key".
func DeleteNestedPath(obj map[string]interface{}, path string) {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 0 || obj == nil {
		return
	}

	key := parts[0]
	if len(parts) == 1 {
		// Leaf — delete this key.
		delete(obj, key)
		return
	}

	// Intermediate — recurse into the nested map if it exists.
	if nested, ok := obj[key].(map[string]interface{}); ok {
		DeleteNestedPath(nested, parts[1])
	}
}

// DeepCopyMap returns a shallow-to-one-level deep copy of a map[string]interface{}.
// Nested maps are also copied; slices and scalar values share the same pointer.
// Sufficient for our use case: we only modify top-level keys and nested map keys
// via deleteNestedPath — we never mutate slice elements or scalar values.
func DeepCopyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		if nested, ok := v.(map[string]interface{}); ok {
			dst[k] = DeepCopyMap(nested)
		} else {
			dst[k] = v
		}
	}
	return dst
}

// SetContains is a nil-safe struct{} map membership check.
func SetContains(s map[string]struct{}, key string) bool {
	_, ok := s[key]
	return ok
}

// MapContains is a nil-safe map membership check for any value type.
func MapContains[V any](m map[string]V, key string) bool {
	if m == nil {
		return false
	}
	_, ok := m[key]
	return ok
}
