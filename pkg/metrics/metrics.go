// Package metrics provides all Prometheus metrics emitted by the Orkestra
// operator runtime.
//
// These metrics are intentionally minimal, high‑signal, and CRD‑aware.
// Unlike generic Kubernetes metrics, Orkestra exposes *per‑CRD* operational
// insights that reveal:
//   - reconcile volume
//   - reconcile latency
//   - queue pressure
//   - informer resource counts
//   - worker utilization
//   - CRD activation behavior
//
// These metrics are unique to Orkestra because they reflect the *declarative
// operator model* — every CRD is treated as a first‑class unit of work, and
// metrics are labeled by CRD name.
//
// The goal is to give platform engineers deep visibility into operator
// behavior without requiring custom instrumentation or Go code.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ─────────────────────────────────────────────────────────────────────────────
// ReconcileTotal
// Counts every reconcile attempt for every CRD.
// Labeled by:
//   - crd: CRD name
//   - result: "success" or "failure"
//
// This is the primary volume metric for operator activity.
// ─────────────────────────────────────────────────────────────────────────────
var ReconcileTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "controller_reconcile_total",
	Help: "Total number of reconciliations",
}, []string{"crd", "result"})

// ─────────────────────────────────────────────────────────────────────────────
// ReconcileDuration
// Histogram of reconcile latency per CRD.
// Helps identify slow reconcilers, API bottlenecks, or template evaluation cost.
// ─────────────────────────────────────────────────────────────────────────────
var ReconcileDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "controller_reconcile_duration_seconds",
	Help:    "Duration of reconciliations",
	Buckets: prometheus.DefBuckets,
}, []string{"crd"})

// ─────────────────────────────────────────────────────────────────────────────
// ResourceCount
// Gauge of how many CRs exist for each CRD (from informer cache).
// Useful for:
//   - capacity planning
//   - detecting runaway CR creation
//   - validating informer correctness
//
// ─────────────────────────────────────────────────────────────────────────────
var ResourceCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "controller_resource_count",
	Help: "Number of custom resources (CR) per CRD",
}, []string{"crd"})

// ─────────────────────────────────────────────────────────────────────────────
// QueueDepth
// Gauge of the current work queue depth per CRD.
// High queue depth indicates backpressure or insufficient workers.
// ─────────────────────────────────────────────────────────────────────────────
var QueueDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "controller_queue_depth",
	Help: "Current queue depth per CRD",
}, []string{"crd"})

// ─────────────────────────────────────────────────────────────────────────────
// WorkersActive
// Gauge of how many workers are actively reconciling per CRD.
// Helps tune worker counts and detect worker starvation.
// ─────────────────────────────────────────────────────────────────────────────
var WorkersActive = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "controller_workers_active",
	Help: "Number of active workers per CRD",
}, []string{"crd"})

// ─────────────────────────────────────────────────────────────────────────────
// CRDActivationLatency
// Histogram measuring how long it takes for a CRD to become "active"
// after the operator starts (i.e., informers synced + reconciler ready).
// Useful for startup diagnostics and readiness tuning.
// ─────────────────────────────────────────────────────────────────────────────
var CRDActivationLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "controller_crd_activation_latency_seconds",
	Help:    "Time from controller start to CRD becoming active",
	Buckets: prometheus.DefBuckets,
}, []string{"crd"})

// ─────────────────────────────────────────────────────────────────────────────
// CRDActivationTotal
// Counts CRD activation attempts after startup.
// Labeled by:
//   - crd
//   - result: "success" or "failure"
//
// Helps detect CRDs that repeatedly fail to initialize.
// ─────────────────────────────────────────────────────────────────────────────
var CRDActivationTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "controller_crd_activation_total",
	Help: "Total number of CRD activations after startup",
}, []string{"crd", "result"})

// ─────────────────────────────────────────────────────────────────────────────
// ConversionTotal
// Counts every conversion request.
// Labeled by:
//   - kind: CRD kind (e.g., "Website")
//   - from_version: source API version (e.g., "v1alpha1")
//   - to_version: target API version (e.g., "v1")
//   - result: "success" or "failure"
// ─────────────────────────────────────────────────────────────────────────────
var ConversionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "orkestra_conversion_requests_total",
	Help: "Total number of conversion requests processed",
}, []string{"kind", "from_version", "to_version", "result"})

// ─────────────────────────────────────────────────────────────────────────────
// ConversionDuration
// Histogram of conversion latency per kind and direction.
// Helps identify slow conversions that might affect API server performance.
// ─────────────────────────────────────────────────────────────────────────────
var ConversionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "orkestra_conversion_duration_seconds",
	Help:    "Duration of conversion requests",
	Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
}, []string{"kind", "from_version", "to_version"})

// ─────────────────────────────────────────────────────────────────────────────
// ConversionErrors
// Counts conversion errors by kind and error type.
// Helps debug conversion rule misconfigurations.
// ─────────────────────────────────────────────────────────────────────────────
var ConversionErrors = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "orkestra_conversion_errors_total",
	Help: "Total number of conversion errors",
}, []string{"kind", "error_type"})

// ─────────────────────────────────────────────────────────────────────────────
// ConversionActiveRequests
// Gauge of currently in‑flight conversion requests.
// Helps detect conversion backpressure.
// ─────────────────────────────────────────────────────────────────────────────
var ConversionActiveRequests = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "orkestra_conversion_active_requests",
	Help: "Number of conversion requests currently being processed",
})