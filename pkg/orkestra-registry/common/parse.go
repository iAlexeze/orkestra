package common

import (
	"fmt"
	"time"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ParseBool interprets common boolean representations from template expressions.
func ParseBool(s string) bool {
	switch s {
	case "true", "True", "TRUE", "1", "yes", "YES":
		return true
	default:
		return false
	}
}

// ParsePort interprets common port representations from template expressions.
func ParsePort(s string) int {
	var p int
	fmt.Sscanf(s, "%d", &p)
	return p
}

// ParseReplicas parses a replica count string, defaulting to 1 when empty or unparseable.
func ParseReplicas(s string) int32 {
	if s != "" {
		var r int
		if n, _ := fmt.Sscanf(s, "%d", &r); n == 1 && r > 0 {
			return int32(r)
		}
	}
	return 1
}

// SleepIfNeeded parses an extended duration string and sleeps if non-zero.
// Used by all operatorBox resources to inject artificial latency for
// autoscaling tests, chaos engineering, and latency simulation.
func SleepIfNeeded(s string) error {
	if s == "" {
		return nil
	}

	d, err := orktypes.ParseTimeDuration(s)
	if err != nil {
		return err
	}

	if d > 0 {
		time.Sleep(d)
	}

	return nil
}

// LabelsEqual reports whether two maps of labels are equivalent.
func LabelsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok || va != vb {
			return false
		}
	}
	return true
}
