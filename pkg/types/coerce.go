package types

import (
	"encoding/json"
	"strconv"
	"strings"
)

// TryCoerceString attempts to parse a resolved template string as a native
// Go type, so integer/boolean/JSON CRD fields pass Kubernetes API server
// validation instead of being submitted as literal strings. Returns
// float64 for integers and floats (JSON-safe), bool for booleans,
// map[string]any/[]any for a JSON object/array, and the original string
// for everything else — including a malformed near-JSON value, so callers
// can fail closed the same way the numeric/boolean attempts already do.
//
// Shared by every place that resolves a Go-template value destined for an
// arbitrary (untyped) Kubernetes field — onCreate custom resources, their
// forEach-expanded form, and conversion webhook rules — rather than each
// maintaining its own copy.
func TryCoerceString(s string) any {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return float64(i)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	if trimmed := strings.TrimSpace(s); len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		var v any
		if json.Unmarshal([]byte(trimmed), &v) == nil {
			return v
		}
	}
	return s
}
