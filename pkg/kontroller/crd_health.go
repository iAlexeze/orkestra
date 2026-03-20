// pkg/kontroller/crd_health.go
package kontroller

import (
	"sync/atomic"
	"time"
)

// CRDHealth tracks the runtime health of a single CRD's reconciler.
// It is fully concurrency‑safe and designed to be updated from multiple goroutines.
//
// Fields tracked:
//   - started:           whether the reconciler has begun processing events
//   - healthy:           whether the reconciler is currently considered healthy
//   - totalReconciles:   total number of reconcile attempts (success + failure)
//   - failedReconciles:  number of failed reconciles
//   - consecutiveFails:  number of failures in a row (used for degradation)
//   - lastError:         last error message (string)
//   - lastReconcile:     timestamp of last reconcile attempt
//   - startTime:         timestamp when the reconciler first started
//
// This struct powers:
//   - /healthz endpoint
//   - metrics
//   - dashboard status
//   - operator self‑diagnostics
type CRDHealth struct {
	name             string
	started          atomic.Bool
	healthy          atomic.Bool
	totalReconciles  atomic.Int64
	failedReconciles atomic.Int64
	consecutiveFails atomic.Int64
	lastError        atomic.Value // stores string
	lastReconcile    atomic.Value // stores time.Time
	startTime        atomic.Value // stores time.Time
}

// NewCRDHealth initializes a CRDHealth tracker for a given CRD name.
// The reconciler starts in an "unhealthy" state until the first successful reconcile.
func NewCRDHealth(name string) *CRDHealth {
	h := &CRDHealth{name: name}
	h.healthy.Store(false)
	return h
}

// RecordSuccess marks a successful reconcile event.
// It resets the consecutive failure counter and updates timestamps.
func (h *CRDHealth) RecordSuccess() {
	h.totalReconciles.Add(1)
	h.consecutiveFails.Store(0)
	h.lastReconcile.Store(time.Now())
	h.healthy.Store(true)
}

// RecordFailure marks a failed reconcile event.
// It increments failure counters, stores the error, and may degrade health
// if the number of consecutive failures exceeds the configured threshold.
func (h *CRDHealth) RecordFailure(err error, degradeThreshold int) {
	h.totalReconciles.Add(1)
	h.failedReconciles.Add(1)
	h.consecutiveFails.Add(1)
	h.lastError.Store(err.Error())
	h.lastReconcile.Store(time.Now())

	// If too many failures in a row, mark the reconciler unhealthy.
	if h.consecutiveFails.Load() >= int64(degradeThreshold) {
		h.healthy.Store(false)
	}
}

// RecordStartupFailure is used when the reconciler fails before it has fully started.
// It does not affect total reconcile counts, only consecutive failure tracking.
func (h *CRDHealth) RecordStartupFailure(err error, degradeThreshold int) {
	h.consecutiveFails.Add(1)
	h.lastError.Store(err.Error())
}

// ErrorRate returns the ratio of failed reconciles to total reconciles.
// If no reconciles have occurred, the error rate is 0.
func (h *CRDHealth) ErrorRate() float64 {
	total := h.totalReconciles.Load()
	if total == 0 {
		return 0
	}
	return float64(h.failedReconciles.Load()) / float64(total)
}

// LastReconcile returns a human‑readable timestamp of the last reconcile.
// If the reconciler has started but not yet reconciled, it returns a placeholder.
func (h *CRDHealth) LastReconcile() string {
	v := h.lastReconcile.Load()
	if v == nil && h.Started() {
		return "no reconciles yet"
	}

	return v.(time.Time).Round(time.Second).String()
}

// IsHealthy reports whether the reconciler is currently considered healthy.
// Health is degraded after N consecutive failures.
func (h *CRDHealth) IsHealthy() bool {
	return h.healthy.Load()
}

// Started reports whether the reconciler has begun processing events.
func (h *CRDHealth) Started() bool {
	return h.started.Load()
}

// StartedAt returns the timestamp when the reconciler first started.
// If not started, returns "not started". If starting, returns "starting".
func (h *CRDHealth) StartedAt() string {
	v := h.startTime.Load()
	if v == nil {
		return "not started"
	}

	if v.(time.Time).IsZero() {
		return "starting"
	}

	return v.(time.Time).Round(time.Second).String()
}

// SetStarted marks the reconciler as started and records the start time.
// CompareAndSwap ensures the timestamp is only set once.
func (h *CRDHealth) SetStarted() {
	h.startTime.CompareAndSwap(nil, time.Now())
	h.started.Store(true)
}

// Name returns the CRD name associated with this health tracker.
func (h *CRDHealth) Name() string {
	return h.name
}

// TotalReconciles returns the total number of reconcile attempts.
func (h *CRDHealth) TotalReconciles() int64 {
	return h.totalReconciles.Load()
}

// FailedReconciles returns the number of failed reconcile attempts.
func (h *CRDHealth) FailedReconciles() int64 {
	return h.failedReconciles.Load()
}

// LastError returns the last recorded error message.
// If no error has occurred, this will panic — callers should check before use.
func (h *CRDHealth) LastError() string {
	return h.lastError.Load().(string)
}

// ConsecutiveFails returns the number of consecutive failed reconciles.
func (h *CRDHealth) ConsecutiveFails() int64 {
	return h.consecutiveFails.Load()
}

// Uptime returns how long the reconciler has been running.
// If not started, returns "not started".
func (h *CRDHealth) Uptime() string {
	v := h.startTime.Load()
	if v == nil {
		return "not started"
	}
	return time.Since(v.(time.Time)).Round(time.Second).String()
}
