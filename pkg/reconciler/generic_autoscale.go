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

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/logger"
	orkqueue "github.com/ialexeze/orkestra/pkg/queue"
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

var _ AutoscaleTarget = (*GenericReconciler[domain.Object])(nil)
var _ AutoscalerRunner = (*GenericReconciler[domain.Object])(nil)
var _ ResyncLoopStarter = (*GenericReconciler[domain.Object])(nil)
var _ QueueInjector = (*GenericReconciler[domain.Object])(nil)
var _ QueueDepthReporter = (*GenericReconciler[domain.Object])(nil)

// ── AutoscaleTarget ───────────────────────────────────────────────────────────

// ResizeWorkers adjusts the semaphore capacity, changing how many goroutines
// may be inside Reconcile simultaneously. In-flight reconciles are never
// interrupted — they complete normally before the new limit takes effect.
func (r *GenericReconciler[T]) ResizeWorkers(n int) {
	r.workerSem.Resize(n)
	logger.Info().
		Str("crd", r.crd.GVKString()).
		Int("workers", n).
		Msg("autoscaler: worker pool resized")
}

// SetQueueDepthLimit updates the workqueue's depth limit. New enqueue calls
// that would push the queue beyond this limit are dropped with a warning.
// 0 means unlimited (the default). Safe to call at any time.
func (r *GenericReconciler[T]) SetQueueDepthLimit(n int) {
	if r.queue != nil {
		r.queue.SetMaxQueueDepth(n)
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
func (r *GenericReconciler[T]) SetResyncInterval(d time.Duration) {
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
func (r *GenericReconciler[T]) RunAutoscaler(ctx context.Context) {
	if r.autoscaler != nil {
		r.autoscaler.Run(ctx)
	}
}

// ── ResyncLoopStarter ─────────────────────────────────────────────────────────

// StartResyncLoop launches the adjustable resync goroutine. Called once by
// startCRDWorkers when autoscale: is declared on the operatorbox.
func (r *GenericReconciler[T]) StartResyncLoop(ctx context.Context) {
	go r.resyncLoop(ctx)
}

// resyncLoop periodically re-enqueues all objects from the informer cache.
// It is idle (polling every 500 ms) while resyncNs == 0 and fires at the
// stored interval otherwise. This is an additive resync — it runs alongside
// the informer's built-in resync period. The workqueue deduplicates items so
// double-enqueue is safe.
func (r *GenericReconciler[T]) resyncLoop(ctx context.Context) {
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
func (r *GenericReconciler[T]) requeue() {
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
func (r *GenericReconciler[T]) SetQueue(wq *orkqueue.Workqueue) {
	r.queue = wq
}

// ── QueueDepthReporter ────────────────────────────────────────────────────────

// ReportQueueDepth updates the live queue depth metric read by the autoscaler.
// Called by the worker loop after each item is processed.
func (r *GenericReconciler[T]) ReportQueueDepth(depth int64) {
	r.autoMetrics.SetQueueDepth(depth)
}
