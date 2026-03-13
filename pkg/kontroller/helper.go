package kontroller

import (
	"context"
	"fmt"
	"runtime"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
)

// Worker that only processes items for a specific GVK
func (c *Controller) runWorkerForGVK(ctx context.Context, gvk string, workerID string) {
	wq, ok := c.queueRegistry.For(gvk)

	if !ok {
		logger.Warn().Str("gvk", gvk).Msg("no queue for CRD. Using default queue")
		wq = c.defaultWorkqueue
	}

	logger.Debug().Msgf("worker %s started for %s", workerID, gvk)

	for {
		select {
		case <-ctx.Done():
			logger.Debug().Msgf("worker %s for %s stopping", workerID, gvk)
			return
		default:
			if !c.processNextItemForGVK(ctx, gvk) {
				return
			}
		}
		depth := float64(wq.Depth())
		metrics.QueueDepth.WithLabelValues(gvk).Set(depth)
	}
}

// Process next item, but only for the specified GVK
func (c *Controller) processNextItemForGVK(ctx context.Context, gvk string) bool {
	c.crdHealth[gvk] = NewCRDHealth(gvk)

	// Resolve queue — per-CRD if registered, default otherwise
	wq, ok := c.queueRegistry.For(gvk)
	if !ok {
		wq = c.defaultWorkqueue
	}

	item, shutdown := wq.Queue.Get()
	if shutdown {
		return false
	}
	defer wq.Queue.Done(item)

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
		return true
	}

	// Look up the pre-built reconciler
	c.mu.RLock()
	rec := c.reconcilers[gvk]
	c.mu.RUnlock()

	if rec == nil {
		logger.Error().Str("gvk", gvk).Msg("no reconciler found — dropping item")
		wq.Queue.Forget(item)
		return true
	}

	// safeReconcile catches panics
	if err := c.safeReconcile(rec, c.crdHealth[gvk], ctx, item.Key); err != nil {
		logger.Error().Err(err).Str("gvk", gvk).Str("key", item.Key).Msg("reconcile failed")
		wq.Queue.AddRateLimited(item)
		c.failed[gvk]++
		return true
	}

	wq.Queue.Forget(item)
	return true
}

func (c *Controller) safeReconcile(
	rec domain.Reconciler,
	health *CRDHealth,
	ctx context.Context,
	key string,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			err = fmt.Errorf("reconciler panic: %v", r)
			logger.Error().
				Str("key", key).
				Str("stack", string(buf[:n])).
				Msgf("panic recovered: %v", r)
		}
	}()

	err = rec.Reconcile(ctx, key)
	if err != nil {
		health.RecordFailure(err, c.degradeThreshold[key])
		metrics.ReconcileTotal.WithLabelValues(health.name, "error").Inc()
		return err
	}

	health.RecordSuccess()
	metrics.ReconcileTotal.WithLabelValues(health.name, "success").Inc()
	return nil
}
