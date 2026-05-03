// Package katalog provides validation, enrichment, and defaulting for Katalog
// entries before they are merged into the final Orkestra runtime configuration.
//
// This file implements Resource Profiles — pre-built CPU/memory presets that
// expand into a complete ResourceRequirements struct.
//
// Profiles are computed presets. They:
//   - validate the profile name
//   - expand into full requests/limits
//   - apply safe, production-friendly defaults
//   - fail fast on unknown profiles
//
// Profiles DO NOT:
//   - mix with manual resources fields
//   - run at runtime (only during katalog load)
//   - introduce new runtime behavior
//
// The runtime sees a fully-expanded ResourceRequirements exactly as if the
// user had written it manually.

package katalog

import (
	"fmt"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile enum
// ────────────────────────────────────────────────────────────────────────────────
//

type ResourceProfile string

const (
	ResourceTiny         ResourceProfile = "tiny"
	ResourceSmall        ResourceProfile = "small"
	ResourceMedium       ResourceProfile = "medium"
	ResourceLarge        ResourceProfile = "large"
	ResourceBurst        ResourceProfile = "burst"
	ResourceSteady       ResourceProfile = "steady"
	ResourceComputeHeavy ResourceProfile = "compute-heavy"
	ResourceMemoryHeavy  ResourceProfile = "memory-heavy"
)

//
// ────────────────────────────────────────────────────────────────────────────────
// Entrypoint: ApplyResourceProfile
// ────────────────────────────────────────────────────────────────────────────────
//
// Expands a named resource profile into a complete ResourceRequirements struct.
// This runs BEFORE merge, so the runtime only ever sees a fully-formed spec.
//

func ApplyResourceProfile(profile string) (*orktypes.ResourceRequirements, error) {
	switch ResourceProfile(strings.ToLower(profile)) {

	case ResourceTiny:
		return expandTiny(), nil

	case ResourceSmall:
		return expandSmall(), nil

	case ResourceMedium:
		return expandMedium(), nil

	case ResourceLarge:
		return expandLarge(), nil

	case ResourceBurst:
		return expandBurstProfile(), nil

	case ResourceSteady:
		return expandSteadyProfile(), nil

	case ResourceComputeHeavy:
		return expandComputeHeavy(), nil

	case ResourceMemoryHeavy:
		return expandMemoryHeavy(), nil

	default:
		return nil, fmt.Errorf("unknown resource profile: %q", profile)
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile: tiny
// ────────────────────────────────────────────────────────────────────────────────
//
// Goal: Minimal footprint for sidecars, health endpoints, tiny services.
//

func expandTiny() *orktypes.ResourceRequirements {
	return &orktypes.ResourceRequirements{
		Requests: map[string]string{
			"cpu":    "25m",
			"memory": "64Mi",
		},
		Limits: map[string]string{
			"cpu":    "100m",
			"memory": "128Mi",
		},
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile: small
// ────────────────────────────────────────────────────────────────────────────────
//
// Goal: Default for most microservices.
//

func expandSmall() *orktypes.ResourceRequirements {
	return &orktypes.ResourceRequirements{
		Requests: map[string]string{
			"cpu":    "100m",
			"memory": "128Mi",
		},
		Limits: map[string]string{
			"cpu":    "500m",
			"memory": "512Mi",
		},
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile: medium
// ────────────────────────────────────────────────────────────────────────────────
//
// Goal: Standard production web apps.
//

func expandMedium() *orktypes.ResourceRequirements {
	return &orktypes.ResourceRequirements{
		Requests: map[string]string{
			"cpu":    "250m",
			"memory": "256Mi",
		},
		Limits: map[string]string{
			"cpu":    "1",
			"memory": "1Gi",
		},
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile: large
// ────────────────────────────────────────────────────────────────────────────────
//
// Goal: High-traffic or heavier workloads.
//

func expandLarge() *orktypes.ResourceRequirements {
	return &orktypes.ResourceRequirements{
		Requests: map[string]string{
			"cpu":    "500m",
			"memory": "512Mi",
		},
		Limits: map[string]string{
			"cpu":    "2",
			"memory": "2Gi",
		},
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile: burst
// ────────────────────────────────────────────────────────────────────────────────
//
// Goal: Handle sudden spikes. Higher limits, moderate requests.
//

func expandBurstProfile() *orktypes.ResourceRequirements {
	return &orktypes.ResourceRequirements{
		Requests: map[string]string{
			"cpu":    "200m",
			"memory": "256Mi",
		},
		Limits: map[string]string{
			"cpu":    "2",
			"memory": "2Gi",
		},
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile: steady
// ────────────────────────────────────────────────────────────────────────────────
//
// Goal: Predictable, stable workloads with balanced limits.
//

func expandSteadyProfile() *orktypes.ResourceRequirements {
	return &orktypes.ResourceRequirements{
		Requests: map[string]string{
			"cpu":    "300m",
			"memory": "256Mi",
		},
		Limits: map[string]string{
			"cpu":    "600m",
			"memory": "512Mi",
		},
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile: compute-heavy
// ────────────────────────────────────────────────────────────────────────────────
//
// Goal: CPU-bound workloads (builds, pipelines, ML inference).
//

func expandComputeHeavy() *orktypes.ResourceRequirements {
	return &orktypes.ResourceRequirements{
		Requests: map[string]string{
			"cpu":    "1",
			"memory": "512Mi",
		},
		Limits: map[string]string{
			"cpu":    "2",
			"memory": "1Gi",
		},
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile: memory-heavy
// ────────────────────────────────────────────────────────────────────────────────
//
// Goal: Memory-bound workloads (Java, caching, analytics).
//

func expandMemoryHeavy() *orktypes.ResourceRequirements {
	return &orktypes.ResourceRequirements{
		Requests: map[string]string{
			"cpu":    "250m",
			"memory": "1Gi",
		},
		Limits: map[string]string{
			"cpu":    "500m",
			"memory": "2Gi",
		},
	}
}
