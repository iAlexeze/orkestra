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
