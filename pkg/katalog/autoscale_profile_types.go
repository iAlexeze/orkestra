// Package katalog defines shared types used by autoscale profile expansion.
//
// ProfileConfig describes the static, pre-runtime configuration for an
// autoscale profile. Profiles are computed presets that:
//   - define how thresholds relate to maxQueueDepth
//   - define how worker and queueDepth overrides scale from baseline
//   - define interval and cooldown defaults
//
// These values are used during katalog load to expand a profile into a full
// AutoscaleSpec. The autoscaler runtime never sees profiles — only the fully
// expanded spec.

package katalog

import "time"

// ProfileConfig defines the knobs for a single autoscale profile.
//
// QueueThresholdPct:
//
//	Percentage of maxQueueDepth at which scaling should trigger.
//	Example: 0.60 means "scale when queueDepth exceeds 60% of maxQueueDepth".
//
// WorkerMultiplier:
//
//	Multiplier applied to baseline workers to compute the override.
//
// QueueMultiplier:
//
//	Multiplier applied to baseline queueDepth to compute the override.
//
// Interval / Cooldown:
//
//	Default evaluation and cooldown windows for the profile.
type ProfileConfig struct {
	QueueThresholdPct float64
	WorkerMultiplier  float64
	QueueMultiplier   float64
	Interval          time.Duration
	Cooldown          time.Duration
}

// profileConfig maps each AutoscaleProfile to its static configuration.
//
// These values are intentionally simple and predictable. They scale relative to
// maxQueueDepth (for thresholds) and baseline values (for overrides).
var profileConfig = map[AutoscaleProfile]ProfileConfig{
	AutoscaleBurst: {
		QueueThresholdPct: 0.60, // scale at 60% of maxQueueDepth
		WorkerMultiplier:  4.0,
		QueueMultiplier:   10.0,
		Interval:          5 * time.Second,
		Cooldown:          30 * time.Second,
	},
	AutoscaleSteady: {
		QueueThresholdPct: 0.40,
		WorkerMultiplier:  2.0,
		QueueMultiplier:   3.0,
		Interval:          30 * time.Second,
		Cooldown:          2 * time.Minute,
	},
	AutoscaleBatch: {
		QueueThresholdPct: 1.00, // unused (cron-based)
		WorkerMultiplier:  3.0,
		QueueMultiplier:   8.0,
		Interval:          60 * time.Second,
		Cooldown:          5 * time.Minute,
	},
	AutoscaleLatencySensitive: {
		QueueThresholdPct: 0.00, // unused (latency-based)
		WorkerMultiplier:  2.5,
		QueueMultiplier:   1.0, // no queue override
		Interval:          15 * time.Second,
		Cooldown:          1 * time.Minute,
	},
	AutoscaleCostOptimized: {
		QueueThresholdPct: 0.80,
		WorkerMultiplier:  0.5,
		QueueMultiplier:   0.5,
		Interval:          30 * time.Second,
		Cooldown:          10 * time.Minute,
	},
}
