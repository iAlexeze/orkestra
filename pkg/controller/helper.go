package controller

import (
	"context"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
)

// runWorker is a long-running function that processes items from the queue
func (c *Controller) runWorker(ctx context.Context) {
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
