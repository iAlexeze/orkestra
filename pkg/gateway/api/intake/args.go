package intake

import (
	"fmt"
	"strings"
)

// ParseSlackArgs parses "<target> key=value key=value ..." into a flat
// field map — the first token is the target, everything after is a
// key=value pair. "app repository=myorg/payments-api environment=staging"
// becomes {"target": "app", "repository": "myorg/payments-api",
// "environment": "staging"}.
func ParseSlackArgs(text string) (map[string]interface{}, error) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return nil, fmt.Errorf("expected: <target> [key=value ...]")
	}

	fields := map[string]interface{}{"target": parts[0]}
	for _, arg := range parts[1:] {
		kv := strings.SplitN(arg, "=", 2)
		if len(kv) != 2 || kv[0] == "" {
			return nil, fmt.Errorf("invalid argument %q — expected key=value", arg)
		}
		fields[kv[0]] = kv[1]
	}
	return fields, nil
}
