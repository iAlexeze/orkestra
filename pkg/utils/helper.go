package utils

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
