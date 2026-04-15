// pkg/types/autoscale.go
//
// Autoscale configuration for an operatorbox.
//
// The autoscaler adjusts workers, queueDepth, and resync at runtime based on
// metric conditions, time windows, and cron expressions. The CRD's declared
// values are always the baseline — overrides are temporary and automatically
// reverted when conditions are no longer met.
//
// YAML shape (inside operatorBox:):
//
//	operatorBox:
//	  autoscale:
//	    interval: 15s
//	    cooldown: 2m
//	    conditions:
//	      anyOf:
//	        - time:
//	            after: "08:00"
//	            before: "17:00"
//	        - cron: "0 20 * * 1-5"
//	          duration: 4h
//	      when:
//	        - field: metrics.queueDepth
//	          greaterThan: "200"
//	    do:
//	      workers: 12
//	      queueDepth: 1000
//	      resync: 20s
package types

import "time"

// AutoscaleSpec declares the autoscale behavior for one operatorbox.
// Declared inside OperatorBoxConfig.
type AutoscaleSpec struct {
	// Interval is how often the autoscaler evaluates conditions.
	// Shorter intervals respond faster to load changes at the cost of
	// more frequent evaluation overhead (negligible in practice).
	// Default: 15s
	Interval Duration `yaml:"interval,omitempty"`

	// Cooldown is the minimum time conditions must be continuously false
	// before the baseline is restored. Prevents oscillation when metrics
	// fluctuate around a threshold.
	// Default: 2m
	Cooldown Duration `yaml:"cooldown,omitempty"`

	// Conditions declares the trigger conditions. Both blocks must pass
	// when both are declared (AND between blocks, OR within anyOf).
	Conditions AutoscaleConditions `yaml:"conditions"`

	// Do declares the override values applied when conditions are met.
	// Fields omitted in Do are not changed — only declared fields are overridden.
	Do AutoscaleAction `yaml:"do"`

	// A profile is:
	// 	a named preset that expands into a complete autoscale configuration
	// using computed heuristics tuned for a specific behavior pattern
	Profile string `yaml:"profile,omitempty"`
}

// AutoscaleConditions holds the condition blocks for autoscale evaluation.
type AutoscaleConditions struct {
	// AnyOf — OR semantics. At least one condition in this list must be true.
	// Supports: metric conditions, time conditions, dayOfWeek conditions, cron conditions.
	AnyOf []AutoscaleCondition `yaml:"anyOf,omitempty"`

	// When — AND semantics. All conditions in this list must be true.
	// Supports: metric conditions (metrics.*).
	// Time-based conditions belong in AnyOf.
	When []Condition `yaml:"when,omitempty"`
}

// AutoscaleCondition is one condition in the anyOf block.
// Exactly one of the fields should be set per entry.
type AutoscaleCondition struct {
	// Time — active when the current time is within the declared window.
	// After and Before are both optional; omit one for a half-open range.
	Time *TimeWindow `yaml:"time,omitempty"`

	// DayOfWeek — active on the specified days of the week.
	DayOfWeek *DayOfWeekCondition `yaml:"dayOfWeek,omitempty"`

	// Cron — a standard cron expression (5-field) that defines when the
	// window opens. Duration defines how long the window stays open.
	// Without Duration, the window closes after one evaluation interval.
	Cron string `yaml:"cron,omitempty"`

	// Duration — how long a cron-opened window remains active.
	// Required when Cron is set; ignored otherwise.
	Duration Duration `yaml:"duration,omitempty"`

	// Field — metric condition field path (metrics.*).
	// Used when the anyOf entry is a metric condition rather than a time condition.
	Field string `yaml:"field,omitempty"`

	// GreaterThan — comparison value for metric conditions.
	GreaterThan string `yaml:"greaterThan,omitempty"`

	// LessThan — comparison value for metric conditions.
	LessThan string `yaml:"lessThan,omitempty"`
}

// TimeWindow declares a clock-based active window.
type TimeWindow struct {
	// After — active after this time (format: "HH:MM" in 24h).
	After string `yaml:"after,omitempty"`

	// Before — active before this time (format: "HH:MM" in 24h).
	Before string `yaml:"before,omitempty"`
}

// DayOfWeekCondition declares which days the condition is active.
type DayOfWeekCondition struct {
	// In — active on these days. Full English names: Monday, Tuesday, etc.
	In []string `yaml:"in,omitempty"`

	// NotIn — active on all days except these.
	NotIn []string `yaml:"notIn,omitempty"`
}

// AutoscaleAction declares the override values to apply when conditions are met.
// All fields are optional pointers — only set fields are overridden.
// Unset fields retain their current value (baseline or previous override).
type AutoscaleAction struct {
	// Workers — number of concurrent reconcile goroutines.
	Workers *int `yaml:"workers,omitempty"`

	// QueueDepth — maximum queue depth before backpressure.
	QueueDepth *int `yaml:"queueDepth,omitempty"`

	// Resync — resync interval override. How frequently all CRs are
	// re-enqueued regardless of changes.
	Resync *Duration `yaml:"resync,omitempty"`
}

// AutoscaleBaseline captures the CRD's declared configuration before any
// autoscale override is applied. Always restored when conditions are false.
type AutoscaleBaseline struct {
	Workers    int
	QueueDepth int
	Resync     time.Duration
}

// AutoscaleState tracks the runtime state of the autoscaler for one operatorbox.
// Not serialized — ephemeral, reset on restart.
type AutoscaleState struct {
	// OverrideActive is true when an override is currently applied.
	OverrideActive bool

	// ConditionsFalseAt is the time when conditions last became continuously false.
	// Used to enforce the cooldown period before restoring the baseline.
	// Zero when conditions are currently true or have never been false.
	ConditionsFalseAt time.Time

	// CronWindowsOpenAt tracks when each cron-condition window opened.
	// Key is the cron expression string.
	CronWindowsOpenAt map[string]time.Time
}

// Duration is a time.Duration that unmarshals from YAML strings like "15s", "2m", "1h".
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	if s == "" {
		d.Duration = 0
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = parsed
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}

// EffectiveInterval returns the autoscale evaluation interval, applying the
// default of 15s when not declared.
func (a *AutoscaleSpec) EffectiveInterval() time.Duration {
	if a.Interval.Duration == 0 {
		return 15 * time.Second
	}
	return a.Interval.Duration
}

// EffectiveCooldown returns the cooldown duration, applying the default of 2m
// when not declared.
func (a *AutoscaleSpec) EffectiveCooldown() time.Duration {
	if a.Cooldown.Duration == 0 {
		return 2 * time.Minute
	}
	return a.Cooldown.Duration
}
