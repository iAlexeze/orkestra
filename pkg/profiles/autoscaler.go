package profiles

import (
	"fmt"
	"math"
	"strings"
	"time"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// AutoscaleProfile is a named autoscale behavior preset.
type AutoscaleProfile string

const (
	AutoscaleBurst            AutoscaleProfile = "burst"
	AutoscaleSteady           AutoscaleProfile = "steady"
	AutoscaleBatch            AutoscaleProfile = "batch"
	AutoscaleLatencySensitive AutoscaleProfile = "latency-sensitive"
	AutoscaleCostOptimized    AutoscaleProfile = "cost-optimized"
)

// profileConfig holds the static multipliers and timing for each autoscale profile.
// Thresholds are computed relative to maxQueueDepth; overrides relative to baseline.
type profileConfig struct {
	queueThresholdPct float64
	workerMultiplier  float64
	queueMultiplier   float64
	interval          time.Duration
	cooldown          time.Duration
}

var configs = map[AutoscaleProfile]profileConfig{
	AutoscaleBurst: {
		queueThresholdPct: 0.60,
		workerMultiplier:  4.0,
		queueMultiplier:   10.0,
		interval:          5 * time.Second,
		cooldown:          30 * time.Second,
	},
	AutoscaleSteady: {
		queueThresholdPct: 0.40,
		workerMultiplier:  2.0,
		queueMultiplier:   3.0,
		interval:          30 * time.Second,
		cooldown:          2 * time.Minute,
	},
	AutoscaleBatch: {
		queueThresholdPct: 1.00,
		workerMultiplier:  3.0,
		queueMultiplier:   8.0,
		interval:          60 * time.Second,
		cooldown:          5 * time.Minute,
	},
	AutoscaleLatencySensitive: {
		queueThresholdPct: 0.00,
		workerMultiplier:  2.5,
		queueMultiplier:   1.0,
		interval:          15 * time.Second,
		cooldown:          1 * time.Minute,
	},
	AutoscaleCostOptimized: {
		queueThresholdPct: 0.80,
		workerMultiplier:  0.5,
		queueMultiplier:   0.5,
		interval:          30 * time.Second,
		cooldown:          10 * time.Minute,
	},
}

// ApplyAutoscalerProfile expands a named autoscale profile into a complete
// AutoscaleSpec using the CRD's declared baseline values. Returns an error
// for unknown profile names.
func ApplyAutoscalerProfile(name string, b orktypes.AutoscaleBaseline) (*orktypes.AutoscaleSpec, error) {
	p := AutoscaleProfile(strings.ToLower(name))
	cfg, ok := configs[p]
	if !ok {
		return nil, fmt.Errorf("unknown autoscale profile: %q — allowed: burst, steady, batch, latency-sensitive, cost-optimized", name)
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
		return nil, fmt.Errorf("unknown autoscale profile: %q", name)
	}
}

// IsValidAutoscaleProfile reports whether name is a recognized autoscale profile.
func IsValidAutoscaleProfile(name string) bool {
	_, ok := configs[AutoscaleProfile(strings.ToLower(name))]
	return ok
}

func expandBurst(b orktypes.AutoscaleBaseline, cfg profileConfig) *orktypes.AutoscaleSpec {
	threshold := int(float64(b.MaxQueueDepth) * cfg.queueThresholdPct)
	workers := int(float64(b.Workers) * cfg.workerMultiplier)
	queue := int(float64(b.MaxQueueDepth) * cfg.queueMultiplier)
	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: cfg.interval},
		Cooldown: orktypes.Duration{Duration: cfg.cooldown},
		Conditions: orktypes.AutoscaleConditions{
			When: []orktypes.Condition{{Field: "metrics.queueDepth", GreaterThan: fmt.Sprintf("%d", threshold)}},
		},
		Do: orktypes.AutoscaleAction{Workers: intPtr(workers), QueueDepth: intPtr(queue)},
	}
}

func expandSteady(b orktypes.AutoscaleBaseline, cfg profileConfig) *orktypes.AutoscaleSpec {
	threshold := int(float64(b.MaxQueueDepth) * cfg.queueThresholdPct)
	workers := int(float64(b.Workers) * cfg.workerMultiplier)
	queue := int(float64(b.MaxQueueDepth) * cfg.queueMultiplier)
	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: cfg.interval},
		Cooldown: orktypes.Duration{Duration: cfg.cooldown},
		Conditions: orktypes.AutoscaleConditions{
			When: []orktypes.Condition{
				{Field: "metrics.queueDepth", GreaterThan: fmt.Sprintf("%d", threshold)},
				{Field: "metrics.workersBusyPercent", GreaterThan: "70"},
			},
		},
		Do: orktypes.AutoscaleAction{Workers: intPtr(workers), QueueDepth: intPtr(queue)},
	}
}

func expandBatch(b orktypes.AutoscaleBaseline, cfg profileConfig) *orktypes.AutoscaleSpec {
	workers := int(float64(b.Workers) * cfg.workerMultiplier)
	queue := int(float64(b.MaxQueueDepth) * cfg.queueMultiplier)
	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: cfg.interval},
		Cooldown: orktypes.Duration{Duration: cfg.cooldown},
		Conditions: orktypes.AutoscaleConditions{
			AnyOf: []orktypes.Condition{{Cron: "0 23 * * *", Duration: orktypes.Duration{Duration: 3 * time.Hour}}},
		},
		Do: orktypes.AutoscaleAction{Workers: intPtr(workers), QueueDepth: intPtr(queue)},
	}
}

func expandLatencySensitive(b orktypes.AutoscaleBaseline, cfg profileConfig) *orktypes.AutoscaleSpec {
	workers := int(math.Ceil(float64(b.Workers) * cfg.workerMultiplier))
	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: cfg.interval},
		Cooldown: orktypes.Duration{Duration: cfg.cooldown},
		Conditions: orktypes.AutoscaleConditions{
			When: []orktypes.Condition{{Field: "metrics.reconcileDurationP95Ms", GreaterThan: "200"}},
		},
		Do: orktypes.AutoscaleAction{Workers: intPtr(workers)},
	}
}

func expandCostOptimized(b orktypes.AutoscaleBaseline, cfg profileConfig) *orktypes.AutoscaleSpec {
	workers := int(math.Max(1, float64(b.Workers)*cfg.workerMultiplier))
	queue := int(float64(b.MaxQueueDepth) * cfg.queueMultiplier)
	threshold := int(float64(b.MaxQueueDepth) * cfg.queueThresholdPct)
	return &orktypes.AutoscaleSpec{
		Interval: orktypes.Duration{Duration: cfg.interval},
		Cooldown: orktypes.Duration{Duration: cfg.cooldown},
		Conditions: orktypes.AutoscaleConditions{
			When: []orktypes.Condition{
				{Field: "metrics.workersIdlePercent", GreaterThan: "60"},
				{Field: "metrics.queueDepth", GreaterThan: fmt.Sprintf("%d", threshold)},
			},
		},
		Do: orktypes.AutoscaleAction{Workers: intPtr(workers), QueueDepth: intPtr(queue)},
	}
}

func intPtr(v int) *int { return &v }
