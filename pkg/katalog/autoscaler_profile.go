// Package katalog provides validation, enrichment, and defaulting for Katalog
// entries before they are merged into the final Orkestra runtime configuration.
//
// This file implements Autoscaler Profiles — pre-built autoscale behaviors that
// expand into a complete AutoscaleSpec based on the CRD's declared baseline.
//
// Profiles are computed presets. They:
//   - validate the profile name
//   - compute thresholds relative to maxQueueDepth
//   - compute worker/queue overrides from baseline values
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
	"time"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

type AutoscaleProfile string

const (
	AutoscaleBurst            AutoscaleProfile = "burst"
	AutoscaleSteady           AutoscaleProfile = "steady"
	AutoscaleBatch            AutoscaleProfile = "batch"
	AutoscaleLatencySensitive AutoscaleProfile = "latency-sensitive"
	AutoscaleCostOptimized    AutoscaleProfile = "cost-optimized"
)

// ApplyAutoscalerProfile expands a named autoscale profile into a complete
// AutoscaleSpec using the CRD's declared baseline values.
//
// This runs BEFORE merge, so the runtime only ever sees a fully-formed spec.
func ApplyAutoscalerProfile(profile string, b orktypes.AutoscaleBaseline) (*orktypes.AutoscaleSpec, error) {
	p := AutoscaleProfile(strings.ToLower(profile))
	cfg, ok := profileConfig[p]
	if !ok {
		return nil, fmt.Errorf("unknown autoscale profile: %q", profile)
	}

	switch p {
	case AutoscaleBurst:
		return expandBurst(b, cfg), nil
	case AutoscaleSteady:
		return expandSteady(b, cfg), nil
	case AutoscaleBatch:
		return expandBatch(b, cfg), nil
	case AutoscaleLatencySensitive:
		return expandLatencySensitive(b, cfg), nil
	case AutoscaleCostOptimized:
		return expandCostOptimized(b, cfg), nil
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
//   - threshold = pct(maxQueueDepth)
//   - workers   = baseline * multiplier
//   - queue     = baseline * multiplier
//   - very short interval + cooldown
//
// Threshold is computed relative to maxQueueDepth so scaling happens BEFORE
// items are dropped.
//

func expandBurst(b orktypes.AutoscaleBaseline, cfg ProfileConfig) *orktypes.AutoscaleSpec {
	threshold := int(float64(b.MaxQueueDepth) * cfg.QueueThresholdPct)
	workers := int(float64(b.Workers) * cfg.WorkerMultiplier)
	queue := int(float64(b.MaxQueueDepth) * cfg.QueueMultiplier)

	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: cfg.Interval},
		Cooldown: orktypes.Duration{Duration: cfg.Cooldown},
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
//   - threshold = pct(maxQueueDepth)
//   - workers   = baseline * multiplier
//   - queue     = baseline * multiplier
//   - moderate interval + cooldown
//

func expandSteady(b orktypes.AutoscaleBaseline, cfg ProfileConfig) *orktypes.AutoscaleSpec {
	threshold := int(float64(b.MaxQueueDepth) * cfg.QueueThresholdPct)
	workers := int(float64(b.Workers) * cfg.WorkerMultiplier)
	queue := int(float64(b.MaxQueueDepth) * cfg.QueueMultiplier)

	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: cfg.Interval},
		Cooldown: orktypes.Duration{Duration: cfg.Cooldown},
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
//   - workers = baseline * multiplier
//   - queue   = baseline * multiplier
//   - long cooldown
//
// QueueThresholdPct is unused for batch profiles.
//

func expandBatch(b orktypes.AutoscaleBaseline, cfg ProfileConfig) *orktypes.AutoscaleSpec {
	workers := int(float64(b.Workers) * cfg.WorkerMultiplier)
	queue := int(float64(b.MaxQueueDepth) * cfg.QueueMultiplier)

	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: cfg.Interval},
		Cooldown: orktypes.Duration{Duration: cfg.Cooldown},
		Conditions: orktypes.AutoscaleConditions{
			AnyOf: []orktypes.Condition{
				{
					Cron:     "0 23 * * *",
					Duration: orktypes.Duration{Duration: 3 * time.Hour},
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
//   - threshold = 200ms P95 (queue threshold unused)
//   - workers   = ceil(baseline * multiplier)
//

func expandLatencySensitive(b orktypes.AutoscaleBaseline, cfg ProfileConfig) *orktypes.AutoscaleSpec {
	workers := int(math.Ceil(float64(b.Workers) * cfg.WorkerMultiplier))

	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: cfg.Interval},
		Cooldown: orktypes.Duration{Duration: cfg.Cooldown},
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
//   - threshold = pct(maxQueueDepth)
//   - workers   = max(1, baseline * multiplier)
//   - queue     = baseline * multiplier
//   - idlePercent > 60
//

func expandCostOptimized(b orktypes.AutoscaleBaseline, cfg ProfileConfig) *orktypes.AutoscaleSpec {
	workers := int(math.Max(1, float64(b.Workers)*cfg.WorkerMultiplier))
	queue := int(float64(b.MaxQueueDepth) * cfg.QueueMultiplier)
	threshold := int(float64(b.MaxQueueDepth) * cfg.QueueThresholdPct)

	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: cfg.Interval},
		Cooldown: orktypes.Duration{Duration: cfg.Cooldown},
		Conditions: orktypes.AutoscaleConditions{
			When: []orktypes.Condition{
				{
					Field:       "metrics.workersIdlePercent",
					GreaterThan: "60",
				},
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
// Helpers
// ────────────────────────────────────────────────────────────────────────────────
//

func intPtr(v int) *int { return &v }
