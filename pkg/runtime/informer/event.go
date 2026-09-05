package informer

import (
	"context"

	"github.com/orkspace/orkestra/pkg/logger"
)

// handleEvent resolves the GVK from the scheme and routes the event
// to the correct per-CRD queue. Falls back to the default queue if
// no per-CRD queue is registered for this GVK.
func (f *Factory) handleEvent(ctx context.Context, obj interface{}) {
	// Block until factory is ready — List/Watch have started
	<-f.ready

	gvk, err := gvkFromObj(obj, f.scheme)
	if err != nil {
		return
	}

	gvkStr := gvk.String()

	// ── Tier 2: Pre-enqueue namespace filter ─────────────────────────────
	// Check namespace restriction BEFORE the item enters the queue.
	// Items that fail this check are dropped — they do no work and create
	// no queue pressure. The reconciler check (Tier 3) remains as a safety
	// net for race conditions during startup.
	namespace := extractNamespace(obj)
	if !f.namespaceAllowed(gvkStr, namespace) {
		logger.Debug().
			Str("gvk", gvkStr).
			Str("namespace", namespace).
			Msg("informer: event dropped — namespace not allowed")
		return
	}

	// ── Tier 2b: Pre-enqueue condition filter ────────────────────────────
	// Evaluate operatorBox.preReconcile.filter conditions before enqueue.
	// Objects that fail the filter are dropped — they never enter the queue.
	if !f.enqueueAllowed(ctx, gvkStr, obj) {
		return
	}

	// Route to per-CRD queue if registered, otherwise fall back to default
	wq, ok := f.queueRegistry.For(gvkStr)
	if !ok {
		logger.Warn().
			Str("gvk", gvkStr).
			Msg("no per-CRD queue registered — falling back to default queue")
		f.defaultWq.Enqueue(obj, gvkStr)
		return
	}

	wq.Enqueue(obj, gvkStr)
}

// handleUpdate routes an update event for oldObj→newObj to the correct queue.
// When a sentinel-aware update filter is registered for the GVK, it is evaluated
// first — both oldObj and newObj are available here for sentinel computation.
// If the filter passes, EnqueueWithSentinels carries the sentinel map through.
// When no update filter is registered, falls through to the standard enqueue path.
func (f *Factory) handleUpdate(ctx context.Context, gvkStr string, oldObj, newObj interface{}) {
	<-f.ready

	namespace := extractNamespace(newObj)
	if !f.namespaceAllowed(gvkStr, namespace) {
		logger.Debug().
			Str("gvk", gvkStr).
			Str("namespace", namespace).
			Msg("informer: update dropped — namespace not allowed")
		return
	}

	wq, qFound := f.queueRegistry.For(gvkStr)
	sentinels := f.computeSentinels(gvkStr, oldObj, newObj)
	allowed, hasUpdateFilter := f.allowEnqueue(ctx, gvkStr, newObj, wq, sentinels)

	if hasUpdateFilter {
		if !allowed {
			return
		}

		eventAware := f.katalog.IsEventAware(gvkStr)

		if !qFound {
			logger.Warn().Str("gvk", gvkStr).Msg("no per-CRD queue — falling back to default queue")
			if eventAware {
				f.defaultWq.EnqueueWithEventSentinels(
					newObj,
					gvkStr,
					sentinels,
				)
			} else {
				f.defaultWq.EnqueueWithSentinels(
					newObj,
					gvkStr,
					sentinels,
				)
			}

			return
		}
		if eventAware {
			wq.EnqueueWithEventSentinels(
				newObj,
				gvkStr,
				sentinels,
			)
			return
		}

		wq.EnqueueWithSentinels(
			newObj,
			gvkStr,
			sentinels,
		)
	}

	// No update filter — standard path (same as handleEvent).
	if !f.enqueueAllowed(ctx, gvkStr, newObj) {
		return
	}

	if !qFound {
		logger.Warn().Str("gvk", gvkStr).Msg("no per-CRD queue — falling back to default queue")
		f.defaultWq.Enqueue(newObj, gvkStr)
		return
	}
	wq.Enqueue(newObj, gvkStr)
}
