package kordinator

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
	"github.com/ialexeze/orkestra/pkg/queue"
)

// queueDepthReporter is a local interface satisfied by GenericReconciler so the
// worker loop can push live queue depth into AutoMetrics for autoscale evaluation.
// Defined here (not imported from reconciler) to avoid import cycles.
type queueDepthReporter interface {
	ReportQueueDepth(depth int64)
}

// Worker that only processes items for a specific GVK
func (k *Kontroller) runWorkerForGVK(ctx context.Context, gvk string, workerID string) {
	wq, ok := k.queueRegistry.For(gvk)

	if !ok {
		logger.Warn().Str("gvk", gvk).Msg("no queue for CRD. Using default queue")
		wq = k.defaultWorkqueue
	}

	logger.Debug().Str("worker_id", workerID).Str("gvk", gvk).Msg("worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Debug().Str("worker_id", workerID).Str("gvk", gvk).Msg("worker stopped")
			return
		default:
			// Worker is idle - waiting for work
			// (already idle from previous iteration or initialization)

			// Wait for an item
			item, shutdown := wq.Queue.Get()
			if shutdown {
				logger.Debug().Str("worker_id", workerID).Str("gvk", gvk).Msg("worker stopping (queue shutdown)")
				return
			}

			// Worker is now processing
			k.crdHealthMap[gvk].MarkWorkerProcessing(workerID)

			// Process the item
			func() {
				defer wq.Queue.Done(item)
				k.processItemForGVK(ctx, gvk, item)
			}()

			// Back to idle after processing
			k.crdHealthMap[gvk].MarkWorkerIdle(workerID)

			// Update metrics (outside of processing state)
			depth := float64(wq.Depth())
			metrics.SetQueueDepth(gvk, depth)

			// Push live depth into the reconciler's AutoMetrics so the autoscaler
			// can read it without an API call. No-op for non-autoscaled CRDs.
			k.mu.RLock()
			rec := k.reconcilers[gvk]
			k.mu.RUnlock()
			if reporter, ok := rec.(queueDepthReporter); ok {
				reporter.ReportQueueDepth(int64(wq.Depth()))
			}

			// Resource count — read from this CRD's informer cache
			if entry, ok := k.katalog.Get(gvk); ok && entry.Informer != nil {
				count := float64(len(entry.Informer.GetIndexer().List()))
				metrics.SetResourceCount(gvk, count)
			}
		}
	}
}

// processItemForGVK handles a single reconciliation item
func (k *Kontroller) processItemForGVK(ctx context.Context, gvk string, item queue.QueueItem) {
	if err := ctx.Err(); err != nil {
		logger.Debug().Msg("process item: context cancelled")
		return
	}

	// Resolve queue — per-CRD if registered, default otherwise
	wq, ok := k.queueRegistry.For(gvk)
	if !ok {
		wq = k.defaultWorkqueue
	}

	// Added to help shutdown workers and preserve the queue on missing crds
	// After dequeuing, they check ctx.Done() at the top of the loop and exit.
	// The queue is intact for reactivation. No ShutDown() called.
	// TODO: Currently does not shutdown the workers
	if item.Key == drainSentinel {
		wq.Queue.Forget(item)
		return
	}

	// With per-CRD queues this check is only needed for the default queue path
	// where multiple GVKs share one queue
	if item.GVK != gvk {
		if ok {
			// This item is in the wrong per-CRD queue — should not happen
			// Log and drop rather than spin
			logger.Error().
				Str("expected", gvk).
				Str("got", item.GVK).
				Msg("GVK mismatch in per-CRD queue — dropping item")
			wq.Queue.Forget(item)
		} else {
			// Default queue — item belongs to a different GVK, put it back
			// This is the only valid re-queue case
			wq.Queue.AddRateLimited(item)
		}
		return
	}

	// Look up the pre-built reconciler
	k.mu.RLock()
	rec := k.reconcilers[gvk]
	k.mu.RUnlock()

	if rec == nil {
		logger.Error().Str("gvk", gvk).Str("key", item.Key).Msg("no reconciler found — dropping item")
		wq.Queue.Forget(item)
		return
	}

	// safeReconcile catches panics
	if err := k.safeReconcile(rec, k.crdHealthMap[gvk], ctx, item.Key, gvk); err != nil {
		logger.Error().Err(err).Str("gvk", gvk).Str("key", item.Key).Msg("reconcile failed")
		wq.Queue.AddRateLimited(item)
		k.failedReconcile(gvk)
		return
	}

	wq.Queue.Forget(item)
}

// // Process next item, but only for the specified GVK
// func (k *Kontroller) processNextItemForGVK(ctx context.Context, gvk string) bool {
// 	if err := ctx.Err(); err != nil {
// 		logger.Debug().Msg("process item: context cancelled")
// 		return false
// 	}

// 	// Resolve queue — per-CRD if registered, default otherwise
// 	wq, ok := k.queueRegistry.For(gvk)
// 	if !ok {
// 		wq = k.defaultWorkqueue
// 	}

// 	item, shutdown := wq.GetWithContext(ctx)
// 	if shutdown {
// 		return false
// 	}
// 	defer wq.Queue.Done(item)

// 	// With per-CRD queues this check is only needed for the default queue path
// 	// where multiple GVKs share one queue
// 	if item.GVK != gvk {
// 		if ok {
// 			// This item is in the wrong per-CRD queue — should not happen
// 			// Log and drop rather than spin
// 			logger.Error().
// 				Str("expected", gvk).
// 				Str("got", item.GVK).
// 				Msg("GVK mismatch in per-CRD queue — dropping item")
// 			wq.Queue.Forget(item)
// 		} else {
// 			// Default queue — item belongs to a different GVK, put it back
// 			// This is the only valid re-queue case
// 			wq.Queue.AddRateLimited(item)
// 		}
// 		return true
// 	}

// 	// Look up the pre-built reconciler
// 	k.mu.RLock()
// 	rec := k.reconcilers[gvk]
// 	k.mu.RUnlock()

// 	if rec == nil {
// 		logger.Error().Str("gvk", gvk).Str("key", item.Key).Msg("no reconciler found — dropping item")
// 		wq.Queue.Forget(item)
// 		return true
// 	}

// 	// safeReconcile catches panics
// 	if err := k.safeReconcile(rec, k.crdHealthMap[gvk], ctx, item.Key, gvk); err != nil {
// 		logger.Error().Err(err).Str("gvk", gvk).Str("key", item.Key).Msg("reconcile failed")
// 		wq.Queue.AddRateLimited(item)
// 		k.failed[gvk]++
// 		return true
// 	}

// 	wq.Queue.Forget(item)
// 	return true
// }

func (k *Kontroller) safeReconcile(
	rec domain.Reconciler,
	health *CRDHealth,
	ctx context.Context,
	key string,
	gvk string,
) (err error) {

	// record duration
	start := time.Now()
	defer func() {
		metrics.ObserveReconcileDuration(gvk, time.Since(start).Seconds())

		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			err = fmt.Errorf("reconciler panic: %v", r)
			logger.Error().
				Str("gvk", gvk).
				Str("key", key).
				Str("panic", fmt.Sprint(r)).
				Str("stack", string(buf[:n])).
				Msg("reconciler panic recovered")
		}
	}()

	err = rec.Reconcile(ctx, key)
	if err != nil {
		health.RecordFailure(err, k.degradeThreshold[gvk])
		metrics.RecordReconcile(gvk, "error")
		return err
	}

	health.RecordSuccess()
	k.successReconcile(gvk)
	metrics.RecordReconcile(gvk, "success")
	return nil
}
