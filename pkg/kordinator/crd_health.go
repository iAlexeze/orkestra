// pkg/kordinator/crd_health.go
package kordinator

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/queue"
)

// CRDHealth tracks the runtime health of a single CRD's reconciler.
// It is fully concurrency‑safe and designed to be updated from multiple goroutines.
//
// Fields tracked:
//
//   - started:           whether the reconciler has begun processing events
//
//   - healthy:           whether the reconciler is currently considered healthy
//
//   - totalReconciles:   total number of reconcile attempts (success + failure)
//
//   - failedReconciles:  number of failed reconciles
//
//   - consecutiveFails:  number of failures in a row (used for degradation)
//
//   - lastError:         last error message (string)
//
//   - lastReconcile:     timestamp of last reconcile attempt
//
//   - startTime:         timestamp when the reconciler first started
//
//   - The activeWarnings: tracks current warn-mode violations per CR.
//     It answers the question: "which CRs are currently violating advisory rules?"
//
//     Map key: "namespace/name" — unique per CR
//     Map value: slice of ActiveWarning, one per violated warn rule
//
// This struct powers:
//   - /katalog/<crd> endpoint
//   - /katalog/<crd>/health endpoint
//   - dashboard status
//   - operator self‑diagnostics
type CRDHealth struct {
	name             string
	started          atomic.Bool
	pending          atomic.Bool
	healthy          atomic.Bool
	degraded         atomic.Bool
	totalReconciles  atomic.Int64
	failedReconciles atomic.Int64
	consecutiveFails atomic.Int64
	lastError        atomic.Value // stores string
	lastReconcile    atomic.Value // stores time.Time
	startTime        atomic.Value // stores time.Time
	queueReg         *queue.QueueRegistry

	// track CRD readines
	lastCRDCheck time.Time
	crdExists    atomic.Bool
	crdCheckMu   sync.RWMutex

	// track workers
	totalWorkers      atomic.Int32
	idleWorkers       atomic.Int32
	processingWorkers atomic.Int32

	// Track individual worker states for debugging
	workerStates sync.Map // workerID -> state (idle, processing, stopped)
	gvk          string   // Store GVK for metrics

	// Dependency tracking
	dependencies     map[string]DependencyStatus
	dependenciesMu   sync.RWMutex
	hasUnhealthyDeps atomic.Bool // Overall dependency health status
	healthySignaled  atomic.Bool
}

type DependencyStatus struct {
	Name                string `json:"name"`
	State               string `json:"state"`               // "healthy", "degraded", "missing", "started"
	Condition           string `json:"condition"`           // "started", "healthy", "ready"
	AcceptableCondition string `json:"acceptableCondition"` // "started", "healthy", "ready"
	Satisfied           bool   `json:"satisfied"`
	LastCheck           string `json:"lastCheck,omitempty"`
}

type OrkestraHealth struct {
	name     string
	orkReady atomic.Bool
	katReady atomic.Bool
}

// NewOrkestraHEalth initializes a CRDHealth tracker for Orkestra
func NewOrkestraHealth() *OrkestraHealth {
	h := &OrkestraHealth{name: konfig.Ork}
	h.orkReady.Store(false)
	h.katReady.Store(false)
	return h
}

// NewCRDHealth initializes a CRDHealth tracker for a given CRD name.
// The reconciler starts in an "unhealthy" state until the first successful reconcile.
func NewCRDHealth(name string) *CRDHealth {
	h := &CRDHealth{name: name}
	h.healthy.Store(false)
	h.pending.Store(true)
	h.degraded.Store(false)
	return h
}

// RecordSuccess marks a successful reconcile event.
// It resets the consecutive failure counter and updates timestamps.
func (h *CRDHealth) RecordSuccess() {
	h.totalReconciles.Add(1)
	h.consecutiveFails.Store(0)
	h.lastReconcile.Store(time.Now())
	h.healthy.Store(true)
	h.pending.Store(false)
	h.degraded.Store(false)
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
		h.pending.Store(false)
		h.degraded.Store(true)
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

	// Case 1: nothing stored yet
	if v == nil {
		if h.Started() {
			h.pending.Store(true)
			return "no reconciles yet"
		}
		return "not started"
	}

	// Case 2: stored value is nil inside interface{}
	t, ok := v.(time.Time)
	if !ok || t.IsZero() {
		if h.Started() {
			h.pending.Store(true)
			return "no reconciles yet"
		}
		return "not started"
	}

	return t.UTC().Format(time.RFC3339)
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

// Pending reports the reconciler has started but yet not yet reconciled.
func (h *CRDHealth) Pending() bool {
	return h.pending.Load()
}

// StartedAt returns the timestamp when the reconciler first started.
// If not started, returns "not started". If starting, returns "starting".
func (h *CRDHealth) StartedAt() string {
	v := h.startTime.Load()
	t, ok := v.(time.Time)
	if !ok {
		return "not started"
	}
	if t.IsZero() {
		return "starting"
	}
	return t.UTC().Format(time.RFC3339)
}

// SetStarted marks the reconciler as started and records the start time.
// CompareAndSwap ensures the timestamp is only set once.
func (h *CRDHealth) SetStarted() {
	h.startTime.CompareAndSwap(nil, time.Now())
	h.started.Store(true)
	h.pending.Store(true)
}

// SetDegraded marks the reconciler as degraded
// This is used when a CRD goes missing at runtime
func (h *CRDHealth) SetDegraded() {
	h.healthy.Store(false)
	h.pending.Store(false)
	h.degraded.Store(true)
}

func (h *CRDHealth) SignaledHealthy() bool {
	return h.healthySignaled.Load()
}

func (h *CRDHealth) MarkHealthySignaled() {
	h.healthySignaled.Store(true)
}

// SetOrkReady marks orkestra engine as ready
func (h *OrkestraHealth) SetOrkReady() {
	h.orkReady.Store(true)
}

// SetOrkDegraded marks orkestra engine as degraded
func (h *OrkestraHealth) SetOrkDegraded() {
	h.orkReady.Store(false)
}

// IsOrkReady is used to track ready state of orkestra
func (h *OrkestraHealth) IsOrkReady() bool {
	return h.orkReady.Load()
}

// SetKatalogReady marks a katalog as ready
func (h *OrkestraHealth) SetKatalogReady() {
	h.katReady.Store(true)
}

// SetKatalogDegraded marks a katalog as degraded
func (h *OrkestraHealth) SetKatalogDegraded() {
	h.katReady.Store(false)
}

// IsKatalogReady is used to track ready state of a katalog
func (h *OrkestraHealth) IsKatalogReady() bool {
	return h.katReady.Load()
}

// SetNotStarted marks the reconciler as not started.
func (h *CRDHealth) SetNotStarted() {
	h.started.Store(false)
}

func (h *CRDHealth) SetMissingAtRuntime() {
	h.crdCheckMu.Lock()
	defer h.crdCheckMu.Unlock()

	h.lastCRDCheck = time.Now()
	h.crdExists.Store(false)
	h.healthy.Store(false)
	h.pending.Store(false)
	h.consecutiveFails.Add(1)
	h.lastError.Store("CRD missing at runtime")
}

func (h *CRDHealth) IsMissing() bool {
	return h.crdExists.Load()
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
// If no error has occurred, it returns an empty string.
func (h *CRDHealth) LastError() string {
	if v, ok := h.lastError.Load().(string); ok {
		return v
	}
	return ""
}

// ConsecutiveFails returns the number of consecutive failed reconciles.
func (h *CRDHealth) ConsecutiveFails() int64 {
	return h.consecutiveFails.Load()
}

// Uptime returns how long the reconciler has been running.
// If not started, returns "not started".
func (h *CRDHealth) Uptime() string {
	v := h.startTime.Load()
	t, ok := v.(time.Time)
	if !ok {
		return "not started"
	}
	return time.Since(t).Round(time.Second).String()
}

// QueueDepth returns the queue for this CRD
func (h *CRDHealth) QueueDepth(gvk string) int {
	if h.queueReg == nil {
		return 0
	}
	depth := h.queueReg.Depth(gvk)
	if depth < 0 {
		return 0
	}
	return depth
}

// SetCRDExists records that the CRD exists in the cluster.
func (h *CRDHealth) SetCRDExists(exists bool) {
	h.crdExists.Store(exists)
	h.crdCheckMu.Lock()
	defer h.crdCheckMu.Unlock()
	h.lastCRDCheck = time.Now()
}

// CRDExists returns whether the CRD exists in the cluster.
func (h *CRDHealth) CRDExists() bool {
	return h.crdExists.Load()
}

// LastCRDCheck returns when the CRD existence was last verified.
func (h *CRDHealth) LastCRDCheck() time.Time {
	h.crdCheckMu.RLock()
	defer h.crdCheckMu.RUnlock()
	return h.lastCRDCheck
}

// Dependency tracking
func (h *CRDHealth) UpdateDependencyStatus(depName string, status DependencyStatus) {
	h.dependenciesMu.Lock()
	defer h.dependenciesMu.Unlock()

	if h.dependencies == nil {
		h.dependencies = make(map[string]DependencyStatus)
	}
	status.LastCheck = time.Now().Format(time.RFC3339)
	h.dependencies[depName] = status
}

// SetDependencyHealth updates a single dependency's status
func (h *CRDHealth) SetDependencyHealth(depName string, status DependencyStatus) {
	h.dependenciesMu.Lock()
	defer h.dependenciesMu.Unlock()

	if h.dependencies == nil {
		h.dependencies = make(map[string]DependencyStatus)
	}

	status.LastCheck = time.Now().Format(time.RFC3339)
	h.dependencies[depName] = status

	// Recalculate overall health after updating
	h.recalculateOverallDependencyHealth()
}

// recalculateOverallDependencyHealth checks all dependencies and updates the atomic bool
// It uses the long-running updates from dependencyHealthChecker() goroutine
func (h *CRDHealth) recalculateOverallDependencyHealth() {
	anyUnhealthy := false
	for _, dep := range h.dependencies {
		if !dep.Satisfied {
			anyUnhealthy = true
			break
		}
	}
	h.hasUnhealthyDeps.Store(anyUnhealthy)
}

// HasUnhealthyDependencies returns true if any dependency is not satisfied
func (h *CRDHealth) HasUnhealthyDependencies() bool {
	return h.hasUnhealthyDeps.Load()
}

// GetDependencyStatuses returns a copy of all dependency statuses
func (h *CRDHealth) GetDependencyStatuses() map[string]DependencyStatus {
	h.dependenciesMu.RLock()
	defer h.dependenciesMu.RUnlock()

	result := make(map[string]DependencyStatus, len(h.dependencies))
	for k, v := range h.dependencies {
		result[k] = v
	}
	return result
}
