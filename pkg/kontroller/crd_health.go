// pkg/kontroller/crd_health.go
package kontroller

import (
	"sync/atomic"
	"time"
)

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

func NewCRDHealth(name string) *CRDHealth {
	h := &CRDHealth{name: name}
	h.healthy.Store(false) // starts false — set true after first successful reconcile
	return h
}

func (h *CRDHealth) RecordSuccess() {
	h.totalReconciles.Add(1)
	h.consecutiveFails.Store(0)
	h.lastReconcile.Store(time.Now())
	h.healthy.Store(true)
}

func (h *CRDHealth) RecordFailure(err error, degradeThreshold int) {
	h.totalReconciles.Add(1)
	h.failedReconciles.Add(1)
	h.consecutiveFails.Add(1)
	h.lastError.Store(err.Error())
	h.lastReconcile.Store(time.Now())

	// Degrade after N consecutive failures — configurable per CRD
	if h.consecutiveFails.Load() >= int64(degradeThreshold) {
		h.healthy.Store(false)
	}
}

func (h *CRDHealth) RecordStartupFailure(err error, degradeThreshold int) {
	h.consecutiveFails.Add(1)
	h.lastError.Store(err.Error())
}

func (h *CRDHealth) ErrorRate() float64 {
	total := h.totalReconciles.Load()
	if total == 0 {
		return 0
	}
	return float64(h.failedReconciles.Load()) / float64(total)
}

func (h *CRDHealth) IsHealthy() bool {
	return h.healthy.Load()
}

func (h *CRDHealth) Started() bool {
	return h.started.Load()
}

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

func (h *CRDHealth) SetStarted() {
	h.startTime.CompareAndSwap(nil, time.Now()) // set once, never overwrite
	h.started.Store(true)
}

func (h *CRDHealth) Name() string {
	return h.name
}

func (h *CRDHealth) TotalReconciles() int64 {
	return h.totalReconciles.Load()
}

func (h *CRDHealth) FailedReconciles() int64 {
	return h.failedReconciles.Load()
}

func (h *CRDHealth) LastError() string {
	return h.lastError.Load().(string)
}

func (h *CRDHealth) LastReconcile() time.Time {
	v := h.lastReconcile.Load()
	if v == nil {
		return time.Time{}
	}

	return v.(time.Time)
}

func (h *CRDHealth) ConsecutiveFails() int64 {
	return h.consecutiveFails.Load()
}

func (h *CRDHealth) Uptime() string {
	v := h.startTime.Load()
	if v == nil {
		return "not started"
	}
	return time.Since(v.(time.Time)).Round(time.Second).String()
}
