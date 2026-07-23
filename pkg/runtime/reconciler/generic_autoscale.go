// pkg/reconciler/generic_autoscale.go
//
// AutoscaleTarget implementation for GenericReconciler.
//
// GenericReconciler implements the three AutoscaleTarget methods so it can be
// passed directly to NewAutoscaler. The autoscaler calls these methods; user
// code should not call them directly.
//
// ResizeWorkers — fully wired via ResizableSemaphore.
// SetQueueDepthLimit — enforced atomically in Workqueue.Enqueue.
// SetResyncInterval — drives an independent resync goroutine that re-enqueues
//
//	all cached objects at the declared interval. 0 = idle (informer handles
//	baseline resync). The goroutine is started once by startCRDWorkers via the
//	ResyncLoopStarter interface; SetResyncInterval adjusts its rate at runtime.
package reconciler

import (
	"context"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/runtime/autoscaler"
	orkqueue "github.com/orkspace/orkestra/pkg/runtime/queue"
)

// AutoscalerRunner is an optional interface implemented by reconcilers that
// carry an embedded autoscaler. DependencyKordinator checks for this after
// calling ReconcilerFactory() and launches the goroutine if present.
type AutoscalerRunner interface {
	RunAutoscaler(ctx context.Context)
}

// ResyncLoopStarter is an optional interface for reconcilers that support an
// adjustable resync goroutine. DependencyKordinator starts it once per CRD.
type ResyncLoopStarter interface {
	StartResyncLoop(ctx context.Context)
}

// QueueInjector is an optional interface for injecting the per-CRD workqueue
// after the reconciler is constructed. Called by startCRDWorkers.
type QueueInjector interface {
	SetQueue(wq *orkqueue.Workqueue)
}

// QueueDepthReporter is an optional interface reconcilers may implement to
// receive live queue depth updates from the worker loop.
type QueueDepthReporter interface {
	ReportQueueDepth(depth int64)
}

var _ autoscaler.AutoscaleTarget = (*GenericReconciler[domain.Object])(nil)
var _ AutoscalerRunner = (*GenericReconciler[domain.Object])(nil)
var _ ResyncLoopStarter = (*GenericReconciler[domain.Object])(nil)
var _ QueueInjector = (*GenericReconciler[domain.Object])(nil)
var _ QueueDepthReporter = (*GenericReconciler[domain.Object])(nil)

// Note: interface-satisfaction checks use domain.Object as the type argument
// because it satisfies the PTR domain.Object constraint and is the widest valid
// instantiation. Concrete operators use PTR = *TheirType; the interfaces are
// identical regardless of which pointer type PTR resolves to.

// ── AutoscaleTarget ───────────────────────────────────────────────────────────

// ResizeWorkers adjusts the semaphore capacity and, when scaling up, starts
// additional goroutines via the injected spawnWorker function so that goroutine
// count stays equal to the effective concurrency limit.
func (r *GenericReconciler[PTR]) ResizeWorkers(n int) {
	old := r.workerSem.Capacity()
	r.workerSem.Resize(n)
	if n > old && r.spawnWorker != nil {
		for i := old; i < n; i++ {
			go r.spawnWorker()
		}
	}
	logger.Info().
		Str("crd", r.crd.GVKString()).
		Int("workers", n).
		Msg("autoscaler: worker pool resized")
}

// SetSpawnWorker injects the goroutine-spawn function from kordinator.
// Called once after construction — before the autoscaler starts.
func (r *GenericReconciler[PTR]) SetSpawnWorker(fn func()) {
	r.spawnWorker = fn
}

// SetRollbackNotifiers injects CRDHealth callbacks for rollback tracking.
// Called once by kordinator after constructing the reconciler.
func (r *GenericReconciler[PTR]) SetRollbackNotifiers(onTrigger, onClear func()) {
	r.rollbackTriggerFn = onTrigger
	r.rollbackClearFn = onClear
}

// GetAutoMetrics returns the live AutoMetrics for this reconciler.
// Called by kordinator to register it in the cross-metrics registry.
func (r *GenericReconciler[PTR]) GetAutoMetrics() *autoscaler.AutoMetrics {
	return r.autoMetrics
}

// WorkerInfo returns a live WorkerInfo snapshot for the /katalog/{crd} endpoint.
// configuredWorkers and configuredQueueDepth come from the CRD entry at startup.
func (r *GenericReconciler[PTR]) WorkerInfo(configuredResync string, configuredWorkers, configuredQueueDepth int) *autoscaler.WorkerInfo {
	maxWorkers := configuredWorkers
	if r.autoscaler != nil {
		if snap := r.autoscaler.Snapshot(); snap != nil && snap.EffectiveWorkers > maxWorkers {
			maxWorkers = snap.EffectiveWorkers
		}
	}
	info := autoscaler.BuildWorkerInfo(
		r.workerSem,
		r.autoMetrics,
		configuredWorkers,
		configuredQueueDepth,
		configuredResync,
		maxWorkers,
		r.autoscaler != nil,
		r.autoscaler.Snapshot(),
	)
	return &info
}

// SetQueueDepthLimit updates the workqueue's depth limit. New enqueue calls
// that would push the queue beyond this limit are dropped with a warning.
// 0 means unlimited (the default). Safe to call at any time.
func (r *GenericReconciler[PTR]) SetQueueDepthLimit(n int) {
	if r.queue != nil {
		r.queue.SetQueueDepth(n)
	}
	logger.Info().
		Str("crd", r.crd.GVKString()).
		Int("queueDepth", n).
		Msg("autoscaler: queue depth limit updated")
}

// SetResyncInterval sets the resync goroutine's fire rate. 0 idles the
// goroutine — the informer's built-in resync handles the baseline cadence.
// Non-zero values trigger an additional re-enqueue of all cached objects
// at that interval, independently of the informer's resync period.
func (r *GenericReconciler[PTR]) SetResyncInterval(d time.Duration) {
	r.resyncNs.Store(d.Nanoseconds())
	logger.Info().
		Str("crd", r.crd.GVKString()).
		Dur("resync", d).
		Msg("autoscaler: resync interval updated")
}

// ── AutoscalerRunner ──────────────────────────────────────────────────────────

// RunAutoscaler starts the autoscale evaluation loop. Blocks until ctx is
// cancelled. Called by DependencyKordinator in a dedicated goroutine.
// No-op when no autoscale spec is declared.
func (r *GenericReconciler[PTR]) RunAutoscaler(ctx context.Context) {
	if r.autoscaler != nil {
		r.autoscaler.Run(ctx)
	}
}

// ── ResyncLoopStarter ─────────────────────────────────────────────────────────

// StartResyncLoop launches the adjustable resync goroutine. Called once by
// startCRDWorkers when autoscale: is declared on the operatorbox.
func (r *GenericReconciler[PTR]) StartResyncLoop(ctx context.Context) {
	go r.resyncLoop(ctx)
}

// resyncLoop periodically re-enqueues all objects from the informer cache.
// It is idle (polling every 500 ms) while resyncNs == 0 and fires at the
// stored interval otherwise. This is an additive resync — it runs alongside
// the informer's built-in resync period. The workqueue deduplicates items so
// double-enqueue is safe.
func (r *GenericReconciler[PTR]) resyncLoop(ctx context.Context) {
	logger.Debug().Str("crd", r.crd.GVKString()).Msg("autoscaler: resync loop started")

	for {
		ns := r.resyncNs.Load()
		if ns == 0 {
			// Idle — wait a short tick then re-check the interval.
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(ns)):
			r.requeue()
		}
	}
}

// requeue enqueues all objects currently in the informer's store.
// Called by resyncLoop on each fire.
func (r *GenericReconciler[PTR]) requeue() {
	if r.queue == nil {
		return
	}
	gvk := r.crd.GVKString()
	items := r.informer.GetIndexer().List()
	for _, obj := range items {
		r.queue.Enqueue(obj, gvk)
	}
	logger.Debug().
		Str("crd", gvk).
		Int("count", len(items)).
		Dur("interval", time.Duration(r.resyncNs.Load())).
		Msg("autoscaler: resync re-enqueued all objects")
}

// ── QueueInjector ─────────────────────────────────────────────────────────────

// SetQueue injects the per-CRD workqueue. Called by startCRDWorkers after
// ReconcilerFactory() so both SetQueueDepthLimit and the resync goroutine
// have a reference to the right queue.
func (r *GenericReconciler[PTR]) SetQueue(wq *orkqueue.Workqueue) {
	r.queue = wq
}

// ── QueueDepthReporter ────────────────────────────────────────────────────────

// ReportQueueDepth updates the live queue depth metric read by the autoscaler.
// Called by the worker loop after each item is processed.
func (r *GenericReconciler[PTR]) ReportQueueDepth(depth int64) {
	r.autoMetrics.SetQueueDepth(depth)
}
