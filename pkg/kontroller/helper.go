package kontroller

import (
	"context"

	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
)

// runWorker is a long-running function that processes items from the queue
func (c *Controller) runWorker(ctx context.Context, gvk string, worker int) {
	for c.processNextItem(ctx) {
	}
}

// processNextItem processes one item from the queue
func (c *Controller) processNextItem(ctx context.Context) bool {
	wq := c.wq.Queue

	// Wait until there's an item or the queue is shut down
	item, shutdown := wq.Get()
	if shutdown {
		return false
	}

	// We call Done at the end of this function to mark the item as processed
	defer wq.Done(item)

	// Direct lookup
	rec := c.reconcilers[item.GVK]
	if rec == nil {
		logger.Error().
			Str("gvk", item.GVK).
			Str("key", item.Key).
			Msg("no reconciler found")
		wq.Forget(item)
		return true
	}

	// Reconcile
	if err := rec.Reconcile(ctx, item.Key); err != nil {
		logger.Error().
			Err(err).
			Str("gvk", item.GVK).
			Str("key", item.Key).
			Msg("reconcile failed")
		wq.AddRateLimited(item)
		return true
	}

	wq.Forget(item)
	return true
}

// Worker that only processes items for a specific GVK
func (c *Controller) runWorkerForGVK(ctx context.Context, targetGVK string, workerID string) {
	logger.Debug().Msgf("worker %s started for %s", workerID, targetGVK)

	for {
		select {
		case <-ctx.Done():
			logger.Debug().Msgf("worker %d for %s stopping", workerID, targetGVK)
			return
		default:
			if !c.processNextItemForGVK(ctx, targetGVK) {
				return
			}
		}
		depth := float64(c.wq.Depth())
		metrics.QueueDepth.WithLabelValues(targetGVK).Set(depth)
	}
}

// Process next item, but only for the specified GVK
func (c *Controller) processNextItemForGVK(ctx context.Context, targetGVK string) bool {
	wq := c.wq.Queue

	// Get item from queue with timeout to allow context cancellation
	item, shutdown := wq.Get()
	if shutdown {
		return false
	}
	defer wq.Done(item)

	// Skip if this item isn't for our GVK
	if item.GVK != targetGVK {
		// Re-queue? No - put it back for other workers
		wq.AddRateLimited(item)
		return true
	}

	// Find reconciler for this GVK
	reconciler := c.reconcilers[item.GVK]
	if reconciler == nil {
		logger.Error().Str("gvk", item.GVK).Msg("no reconciler found")
		wq.Forget(item)
		return true
	}

	// Reconcile
	if err := reconciler.Reconcile(ctx, item.Key); err != nil {
		logger.Error().Err(err).Str("gvk", item.GVK).Str("key", item.Key).Msg("reconcile failed")
		wq.AddRateLimited(item)
		c.failed[targetGVK]++ // increment failure to track error rate
		return true
	}

	wq.Forget(item)
	return true
}
