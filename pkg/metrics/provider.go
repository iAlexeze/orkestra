package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ─────────────────────────────────────────────────────────────────────────────
// Provider metrics — per CRD, per provider name, per resource kind.
//
// Labels:
//   - crd      : CRD name (GVK string)
//   - provider : provider block name ("aws", "mongodb", "postgres", …)
//   - kind     : resource kind within the block ("s3", "database", "user", …)
//   - result   : "success" | "failure"
//
// These metrics are structurally identical to controller_reconcile_total but
// scoped to external infrastructure calls, giving operators independent
// visibility into provider error rates without conflating them with
// Kubernetes resource reconcile errors.
// ─────────────────────────────────────────────────────────────────────────────

var providerReconcileTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "orkestra_provider_reconcile_total",
		Help: "Total provider reconcile calls per CRD, provider, kind, and result.",
	},
	[]string{"crd", "provider", "kind", "result"},
)

var providerDeleteTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "orkestra_provider_delete_total",
		Help: "Total provider delete calls per CRD, provider, kind, and result.",
	},
	[]string{"crd", "provider", "kind", "result"},
)

var providerReconcileDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "orkestra_provider_reconcile_duration_seconds",
		Help:    "Duration of provider reconcile calls.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	},
	[]string{"crd", "provider", "kind"},
)

// RecordProviderReconcile increments the provider reconcile counter.
// Call after each provider.Reconcile() or provider.Delete() returns.
func RecordProviderReconcile(crd, provider, kind, result string) {
	providerReconcileTotal.WithLabelValues(crd, provider, kind, result).Inc()
}

// RecordProviderDelete increments the provider delete counter.
func RecordProviderDelete(crd, provider, kind, result string) {
	providerDeleteTotal.WithLabelValues(crd, provider, kind, result).Inc()
}

// ObserveProviderDuration records how long a provider reconcile call took.
func ObserveProviderDuration(crd, provider, kind string, seconds float64) {
	providerReconcileDuration.WithLabelValues(crd, provider, kind).Observe(seconds)
}
