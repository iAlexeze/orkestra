// pkg/metrics/metrics.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ReconcileTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "controller_reconcile_total",
		Help: "Total number of reconciliations",
	}, []string{"crd", "result"})

	ReconcileDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "controller_reconcile_duration_seconds",
		Help:    "Duration of reconciliations",
		Buckets: prometheus.DefBuckets,
	}, []string{"crd"})

	ResourceCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "controller_resource_count",
		Help: "Number of custom resources (CR) per CRD",
	}, []string{"crd"})

	QueueDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "controller_queue_depth",
		Help: "Current queue depth per CRD",
	}, []string{"crd"})

	WorkersActive = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "controller_workers_active",
		Help: "Number of active workers per CRD",
	}, []string{"crd"})

	CRDActivationLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "controller_crd_activation_latency_seconds",
		Help:    "Time from controller start to CRD becoming active",
		Buckets: prometheus.DefBuckets,
	}, []string{"crd"})

	CRDActivationTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "controller_crd_activation_total",
		Help: "Total number of CRD activations after startup",
	}, []string{"crd", "result"}) // result=success|failure
)
