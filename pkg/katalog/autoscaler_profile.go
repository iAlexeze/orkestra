// Package katalog provides validation, enrichment, and defaulting for Katalog
// entries before they are merged into the final Orkestra runtime configuration.
//
// This file implements Autoscaler Profiles — pre-built autoscale behaviors that
// expand into a complete AutoscaleSpec based on the CRD's declared baseline.
//
// Profiles are computed presets. They:
//   - validate the profile name
//   - compute thresholds and overrides from baseline values
//   - apply default interval/cooldown values
//   - fail fast on unknown profiles
//
// Profiles DO NOT:
//   - mix with manual autoscale fields
//   - run at runtime (only during katalog load)
//   - introduce new runtime behavior
//
// The autoscaler runtime sees a fully-expanded AutoscaleSpec exactly as if the
// user had written it manually.

package katalog

import (
	"fmt"
	"math"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

type AutoscaleProfile string

const (
	Burst            AutoscaleProfile = "burst"
	Steady           AutoscaleProfile = "steady"
	Batch            AutoscaleProfile = "batch"
	LatencySensitive AutoscaleProfile = "latency-sensitive"
	CostOptimized    AutoscaleProfile = "cost-optimized"
)

// ApplyAutoscalerProfile expands a named autoscale profile into a complete
// AutoscaleSpec using the CRD's declared baseline values.
//
// This runs BEFORE merge, so the runtime only ever sees a fully-formed spec.
func ApplyAutoscalerProfile(profile string, baseline orktypes.AutoscaleBaseline) (*orktypes.AutoscaleSpec, error) {
	switch AutoscaleProfile(strings.ToLower(profile)) {

	case Burst:
		return expandBurst(baseline), nil

	case Steady:
		return expandSteady(baseline), nil

	case Batch:
		return expandBatch(baseline), nil

	case LatencySensitive:
		return expandLatencySensitive(baseline), nil

	case CostOptimized:
		return expandCostOptimized(baseline), nil

	default:
		return nil, fmt.Errorf("unknown autoscale profile: %q", profile)
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile: burst
// ────────────────────────────────────────────────────────────────────────────────
//
// Goal: React instantly to spikes.
// Strategy:
//   - very short interval + cooldown
//   - threshold = queueDepth * 3
//   - workers = baseline * 4
//   - queueDepth = baseline * 10
//

func expandBurst(b orktypes.AutoscaleBaseline) *orktypes.AutoscaleSpec {
	threshold := int(float64(b.QueueDepth) * 3.0)
	workers := int(float64(b.Workers) * 4.0)
	queue := int(float64(b.QueueDepth) * 10.0)

	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: 5 * 1e9},  // 5s
		Cooldown: orktypes.Duration{Duration: 30 * 1e9}, // 30s
		Conditions: orktypes.AutoscaleConditions{
			When: []orktypes.Condition{
				{
					Field:       "metrics.queueDepth",
					GreaterThan: fmt.Sprintf("%d", threshold),
				},
			},
		},
		Do: orktypes.AutoscaleAction{
			Workers:    intPtr(workers),
			QueueDepth: intPtr(queue),
		},
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile: steady
// ────────────────────────────────────────────────────────────────────────────────
//
// Goal: Smooth, predictable scaling.
// Strategy:
//   - moderate interval + cooldown
//   - threshold = queueDepth * 1.5
//   - workers = baseline * 2
//   - queueDepth = baseline * 3
//

func expandSteady(b orktypes.AutoscaleBaseline) *orktypes.AutoscaleSpec {
	threshold := int(float64(b.QueueDepth) * 1.5)
	workers := int(float64(b.Workers) * 2.0)
	queue := int(float64(b.QueueDepth) * 3.0)

	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: 30 * 1e9},  // 30s
		Cooldown: orktypes.Duration{Duration: 120 * 1e9}, // 2m
		Conditions: orktypes.AutoscaleConditions{
			When: []orktypes.Condition{
				{
					Field:       "metrics.queueDepth",
					GreaterThan: fmt.Sprintf("%d", threshold),
				},
				{
					Field:       "metrics.workersBusyPercent",
					GreaterThan: "70",
				},
			},
		},
		Do: orktypes.AutoscaleAction{
			Workers:    intPtr(workers),
			QueueDepth: intPtr(queue),
		},
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile: batch
// ────────────────────────────────────────────────────────────────────────────────
//
// Goal: Scale during scheduled batch windows.
// Strategy:
//   - cron window 23:00 → 02:00
//   - workers = baseline * 3
//   - queueDepth = baseline * 8
//   - long cooldown
//

func expandBatch(b orktypes.AutoscaleBaseline) *orktypes.AutoscaleSpec {
	workers := int(float64(b.Workers) * 3.0)
	queue := int(float64(b.QueueDepth) * 8.0)

	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: 60 * 1e9},  // 60s
		Cooldown: orktypes.Duration{Duration: 300 * 1e9}, // 5m
		Conditions: orktypes.AutoscaleConditions{
			AnyOf: []orktypes.AutoscaleCondition{
				{
					Cron:     "0 23 * * *",
					Duration: orktypes.Duration{Duration: 3 * 3600 * 1e9}, // 3h
				},
			},
		},
		Do: orktypes.AutoscaleAction{
			Workers:    intPtr(workers),
			QueueDepth: intPtr(queue),
		},
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile: latency-sensitive
// ────────────────────────────────────────────────────────────────────────────────
//
// Goal: Keep reconcile latency low.
// Strategy:
//   - threshold = 200ms P95
//   - workers = ceil(baseline * 2.5)
//

func expandLatencySensitive(b orktypes.AutoscaleBaseline) *orktypes.AutoscaleSpec {
	workers := int(math.Ceil(float64(b.Workers) * 2.5))

	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: 15 * 1e9}, // 15s
		Cooldown: orktypes.Duration{Duration: 60 * 1e9}, // 1m
		Conditions: orktypes.AutoscaleConditions{
			When: []orktypes.Condition{
				{
					Field:       "metrics.reconcileDurationP95Ms",
					GreaterThan: "200",
				},
			},
		},
		Do: orktypes.AutoscaleAction{
			Workers: intPtr(workers),
		},
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Profile: cost-optimized
// ────────────────────────────────────────────────────────────────────────────────
//
// Goal: Minimize resource usage.
// Strategy:
//   - workers = max(1, baseline * 0.5)
//   - queueDepth = baseline * 0.5
//   - idlePercent > 60
//

func expandCostOptimized(b orktypes.AutoscaleBaseline) *orktypes.AutoscaleSpec {
	workers := int(math.Max(1, float64(b.Workers)*0.5))
	queue := int(float64(b.QueueDepth) * 0.5)

	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: 30 * 1e9},  // 30s
		Cooldown: orktypes.Duration{Duration: 600 * 1e9}, // 10m
		Conditions: orktypes.AutoscaleConditions{
			When: []orktypes.Condition{
				{
					Field:       "metrics.workersIdlePercent",
					GreaterThan: "60",
				},
			},
		},
		Do: orktypes.AutoscaleAction{
			Workers:    intPtr(workers),
			QueueDepth: intPtr(queue),
		},
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────────
//

func intPtr(v int) *int { return &v }
