package common

import (
	"fmt"
	"strings"
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

// ExpandResourceProfile converts a named resource profile into a
// ResourceRequirements struct. Returns an error for unknown profile names.
// Profiles: tiny, small, medium, large, burst, steady, compute-heavy, memory-heavy.
func ExpandResourceProfile(profile string) (*orktypes.ResourceRequirements, error) {
	switch strings.ToLower(profile) {
	case "tiny":
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "25m", "memory": "64Mi"},
			Limits:   map[string]string{"cpu": "100m", "memory": "128Mi"},
		}, nil
	case "small":
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "100m", "memory": "128Mi"},
			Limits:   map[string]string{"cpu": "500m", "memory": "512Mi"},
		}, nil
	case "medium":
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "250m", "memory": "256Mi"},
			Limits:   map[string]string{"cpu": "1", "memory": "1Gi"},
		}, nil
	case "large":
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "500m", "memory": "512Mi"},
			Limits:   map[string]string{"cpu": "2", "memory": "2Gi"},
		}, nil
	case "burst":
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "200m", "memory": "256Mi"},
			Limits:   map[string]string{"cpu": "2", "memory": "2Gi"},
		}, nil
	case "steady":
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "300m", "memory": "256Mi"},
			Limits:   map[string]string{"cpu": "600m", "memory": "512Mi"},
		}, nil
	case "compute-heavy":
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "1", "memory": "512Mi"},
			Limits:   map[string]string{"cpu": "2", "memory": "1Gi"},
		}, nil
	case "memory-heavy":
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "250m", "memory": "1Gi"},
			Limits:   map[string]string{"cpu": "500m", "memory": "2Gi"},
		}, nil
	default:
		return nil, fmt.Errorf("unknown resource profile: %q", profile)
	}
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
