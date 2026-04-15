// pkg/reconciler/autoscaler.go
//
// Operatorbox autoscaler — evaluates conditions on a ticker and applies or
// restores worker/queue/resync overrides.
//
// One Autoscaler is created per operatorbox that declares autoscale: in its
// operatorBox: block. It runs a single goroutine for the lifetime of the
// operatorbox and stops cleanly when the context is cancelled.
//
// The autoscaler has no persistent state. A restart always begins from the
// declared CRD baseline. Overrides applied before a restart are lost — the
// next evaluation tick will re-apply them if conditions are still met.
package reconciler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/robfig/cron/v3"
)

// AutoscaleTarget is implemented by the operatorbox runtime components that
// the autoscaler controls. Kept minimal — autoscaler only calls these three
// methods, nothing else.
type AutoscaleTarget interface {
	// ResizeWorkers adjusts the worker pool to the given concurrency.
	ResizeWorkers(n int)

	// SetQueueDepthLimit adjusts the maximum queue depth.
	SetQueueDepthLimit(n int)

	// SetResyncInterval adjusts how frequently all CRs are re-enqueued.
	SetResyncInterval(d time.Duration)
}

// Autoscaler evaluates conditions and applies/restores overrides for one operatorbox.
type Autoscaler struct {
	crdKind  string
	spec     *orktypes.AutoscaleSpec
	baseline orktypes.AutoscaleBaseline
	target   AutoscaleTarget
	metrics  *AutoMetrics

	// state — not exported, not persisted
	state orktypes.AutoscaleState
}

// NewAutoscaler constructs an Autoscaler for one operatorbox.
// baseline is captured from the CRD's declared configuration before startup.
func NewAutoscaler(
	crdKind string,
	spec *orktypes.AutoscaleSpec,
	baseline orktypes.AutoscaleBaseline,
	target AutoscaleTarget,
	metrics *AutoMetrics,
) *Autoscaler {
	return &Autoscaler{
		crdKind:  crdKind,
		spec:     spec,
		baseline: baseline,
		target:   target,
		metrics:  metrics,
		state: orktypes.AutoscaleState{
			CronWindowsOpenAt: make(map[string]time.Time),
		},
	}
}

// Run starts the autoscale evaluation loop. Blocks until ctx is cancelled.
// Intended to be called in a dedicated goroutine.
func (a *Autoscaler) Run(ctx context.Context) {
	interval := a.spec.EffectiveInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info().
		Str("crd", a.crdKind).
		Dur("interval", interval).
		Dur("cooldown", a.spec.EffectiveCooldown()).
		Int("baselineWorkers", a.baseline.Workers).
		Msg("autoscaler: started")

	for {
		select {
		case <-ctx.Done():
			// Restore baseline on clean shutdown so that the next startup
			// begins from declared values even if the target persists state.
			if a.state.OverrideActive {
				a.restoreBaseline()
			}
			logger.Info().Str("crd", a.crdKind).Msg("autoscaler: stopped")
			return

		case <-ticker.C:
			a.evaluate(ctx)
		}
	}
}

// evaluate is one evaluation tick. Applies overrides or restores baseline.
func (a *Autoscaler) evaluate(ctx context.Context) {
	met := a.conditionsMet(ctx)

	if met {
		if !a.state.OverrideActive {
			a.applyOverride()
		}
		// Reset cooldown clock — conditions are currently true
		a.state.ConditionsFalseAt = time.Time{}
		return
	}

	// Conditions are false
	if !a.state.OverrideActive {
		return // baseline already active, nothing to do
	}

	// Start or continue the cooldown clock
	if a.state.ConditionsFalseAt.IsZero() {
		a.state.ConditionsFalseAt = time.Now()
	}

	cooldown := a.spec.EffectiveCooldown()
	if time.Since(a.state.ConditionsFalseAt) >= cooldown {
		a.restoreBaseline()
		a.state.ConditionsFalseAt = time.Time{}
	}
}

// conditionsMet evaluates the full condition expression and returns true when
// the override should be applied.
//
// Logic: (anyOf OR-block passes OR anyOf is empty) AND (when AND-block passes OR when is empty)
func (a *Autoscaler) conditionsMet(_ context.Context) bool {
	spec := a.spec
	now := time.Now()

	// Evaluate anyOf block (OR)
	anyOfPassed := len(spec.Conditions.AnyOf) == 0 // empty = always pass
	for _, cond := range spec.Conditions.AnyOf {
		if a.evalAnyOfCond(cond, now) {
			anyOfPassed = true
			break
		}
	}
	if !anyOfPassed {
		return false
	}

	// Evaluate when block (AND) — metric conditions
	for _, cond := range spec.Conditions.When {
		if !a.evalMetricCond(cond) {
			return false
		}
	}

	return true
}

// evalAnyOfCond evaluates one entry in the anyOf block.
func (a *Autoscaler) evalAnyOfCond(cond orktypes.AutoscaleCondition, now time.Time) bool {
	// Time window
	if cond.Time != nil {
		return evalTimeWindow(cond.Time, now)
	}

	// Day of week
	if cond.DayOfWeek != nil {
		return evalDayOfWeek(cond.DayOfWeek, now)
	}

	// Cron expression with duration
	if cond.Cron != "" {
		return a.evalCronWindow(cond, now)
	}

	// Inline metric condition in anyOf
	if cond.Field != "" {
		return a.evalMetricCond(orktypes.Condition{
			Field:       cond.Field,
			GreaterThan: cond.GreaterThan,
			LessThan:    cond.LessThan,
		})
	}

	return false
}

// evalTimeWindow returns true when now is within the declared time window.
func evalTimeWindow(tw *orktypes.TimeWindow, now time.Time) bool {
	if tw.After != "" {
		t, err := parseHHMM(tw.After)
		if err != nil {
			return false
		}
		threshold := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
		if now.Before(threshold) {
			return false
		}
	}
	if tw.Before != "" {
		t, err := parseHHMM(tw.Before)
		if err != nil {
			return false
		}
		threshold := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
		if now.After(threshold) {
			return false
		}
	}
	return true
}

// evalDayOfWeek returns true when today matches the declared day constraint.
func evalDayOfWeek(d *orktypes.DayOfWeekCondition, now time.Time) bool {
	today := now.Weekday().String()
	if len(d.In) > 0 {
		for _, day := range d.In {
			if strings.EqualFold(day, today) {
				return true
			}
		}
		return false
	}
	if len(d.NotIn) > 0 {
		for _, day := range d.NotIn {
			if strings.EqualFold(day, today) {
				return false
			}
		}
		return true
	}
	return false
}

// evalCronWindow returns true when a cron-opened window is currently active.
// The window opens when the cron expression fires and stays open for Duration.
func (a *Autoscaler) evalCronWindow(cond orktypes.AutoscaleCondition, now time.Time) bool {
	schedule, err := cron.ParseStandard(cond.Cron)
	if err != nil {
		logger.Warn().Str("crd", a.crdKind).Str("cron", cond.Cron).Err(err).
			Msg("autoscaler: invalid cron expression — skipping")
		return false
	}

	duration := cond.Duration.Duration
	if duration == 0 {
		// No duration — use one evaluation interval as the window.
		// Document this behavior clearly: without duration, cron is point-in-time.
		duration = a.spec.EffectiveInterval()
	}

	key := cond.Cron

	// Check if a previously opened window is still active
	if opened, ok := a.state.CronWindowsOpenAt[key]; ok {
		if now.Before(opened.Add(duration)) {
			return true // window still open
		}
		// Window closed
		delete(a.state.CronWindowsOpenAt, key)
	}

	// Check if the cron fired within the last evaluation interval
	interval := a.spec.EffectiveInterval()
	prev := schedule.Next(now.Add(-interval))
	if prev.Before(now) {
		// Cron fired within the last interval — open a new window
		a.state.CronWindowsOpenAt[key] = prev
		return true
	}

	return false
}

// evalMetricCond evaluates a single metric condition (metrics.* field).
func (a *Autoscaler) evalMetricCond(cond orktypes.Condition) bool {
	if !IsMetricField(cond.Field) {
		return false
	}

	val := a.metrics.Get(cond.Field)
	if val == "" {
		return false // unknown metric — conservative: treat as not met
	}

	fVal, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return false
	}

	if cond.GreaterThan != "" {
		threshold, err := strconv.ParseFloat(cond.GreaterThan, 64)
		if err != nil {
			return false
		}
		return fVal > threshold
	}

	if cond.LessThan != "" {
		threshold, err := strconv.ParseFloat(cond.LessThan, 64)
		if err != nil {
			return false
		}
		return fVal < threshold
	}

	return false
}

// applyOverride applies the do: block to the target operatorbox.
func (a *Autoscaler) applyOverride() {
	do := a.spec.Do

	log := logger.Info().Str("crd", a.crdKind)

	if do.Workers != nil {
		a.target.ResizeWorkers(*do.Workers)
		log = log.Int("workers", *do.Workers)
		metrics.RecordAutoscaleOverride(a.crdKind, *do.Workers)
	}

	if do.QueueDepth != nil {
		a.target.SetQueueDepthLimit(*do.QueueDepth)
		log = log.Int("queueDepth", *do.QueueDepth)
	}
	if do.Resync != nil {
		a.target.SetResyncInterval(do.Resync.Duration)
		log = log.Dur("resync", do.Resync.Duration)
	}

	a.state.OverrideActive = true
	metrics.SetAutoscaleActive(a.crdKind, true)
	log.Msg("autoscaler: override applied")
}

// restoreBaseline restores the CRD's declared configuration.
func (a *Autoscaler) restoreBaseline() {
	a.target.ResizeWorkers(a.baseline.Workers)
	a.target.SetQueueDepthLimit(a.baseline.QueueDepth)
	// Pass 0 for resync: the informer's built-in resync handles the baseline
	// cadence. 0 idles the autoscaler resync goroutine so it does not add a
	// redundant second trigger on top of the informer's own period.
	a.target.SetResyncInterval(0)

	a.state.OverrideActive = false
	metrics.RecordAutoscaleRestore(a.crdKind, a.baseline.Workers)

	logger.Info().
		Str("crd", a.crdKind).
		Int("workers", a.baseline.Workers).
		Int("queueDepth", a.baseline.QueueDepth).
		Dur("resync", a.baseline.Resync).
		Msg("autoscaler: baseline restored")
}

// parseHHMM parses a "HH:MM" string into a time.Time on an arbitrary date.
// Only the hour and minute are meaningful — the date is discarded.
func parseHHMM(s string) (time.Time, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time %q: expected HH:MM", s)
	}
	return t, nil
}
