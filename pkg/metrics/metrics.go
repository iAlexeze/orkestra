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
//   - mutation/defaulting activity
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
// MutationTotal
// Counts reconciles where at least one mutation/defaulting rule was applied.
// Helps understand how often CRs rely on defaults or auto‑correction.
// ─────────────────────────────────────────────────────────────────────────────
var MutationTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "controller_mutation_total",
		Help: "Total reconciles where at least one mutation rule was applied.",
	},
	[]string{"crd"},
)

// ─────────────────────────────────────────────────────────────────────────────
// MutationAppliedDetail
// Counts individual field‑level mutations.
// Labeled by:
//   - crd
//   - field: the field mutated
//   - type: mutation type (default, override, normalize, etc.)
//
// Helps identify:
//   - which fields are most often missing
//   - which defaults are most active
//   - where schema drift or user misconfiguration occurs
//
// ─────────────────────────────────────────────────────────────────────────────
var MutationAppliedDetail = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "controller_mutation_applied_total",
		Help: "Mutations applied per field, labeled by CRD, field, and mutation type.",
	},
	[]string{"crd", "field", "type"},
)

// ─────────────────────────────────────────────────────────────────────────────
// CleanupTotal
// controller_validation_cleanup_total{crd, field, rule, dry_run}
//
//	Counts every cleanup action taken (or would-have-been-taken in dry-run mode).
//	The dry_run label lets you compare live cleanups vs dry-run observations:
//
//	  # How many live deletions happened?
//	  sum(controller_validation_cleanup_total{dry_run="false"})
//
//	  # How many would dry-run have caught?
//	  sum(controller_validation_cleanup_total{dry_run="true"})
//
//	When rolling out a cleanup rule:
//	1. Deploy with dryRun: true — observe the metric for a reconcile period
//	2. If the count stabilises (no new violations), enable live deletion
//	3. If the count keeps rising, the rule may be too broad — revise it first
//
// ─────────────────────────────────────────────────────────────────────────────
var CleanupTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "controller_validation_cleanup_total",
		Help: "Resources cleaned up (deleted) by validation cleanup rules. " +
			"dry_run=true means the rule would have deleted but did not.",
	},
	[]string{"crd", "field", "rule", "dry_run"},
)
