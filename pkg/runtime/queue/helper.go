package queue

import "github.com/orkspace/orkestra/pkg/logger"

// evaluateQueueBehaviour starts the queue behaviour evaluation when configured
// and delegates to the informer for when/or condition evaluation — the informer has
// access to the full preReconcile resolver context that the queue itself does not.
//
// Fast path: IsUnlimited() — where maxDepth == 0, all items are always enqueued.
func (q *Workqueue) evaluateQueueBehaviour(item QueueItem) bool {
	cfg := q.queueCfg
	if !cfg.IsUnlimited() {
		switch {
		case cfg.HasOnThreshold():
			if cfg.ThresholdReached(q.QueueInfo().Depth) {
				// Delegate to the informer to evaluate when/or conditions.
				// It has access to all preReconcile resolver context.
				if cfg.HasOnThresholdConditions() {
					q.evaluateCond.OnThreshold.Store(true)
					return true
				}
			}
		case q.QueueInfo().DepthReached:
			if cfg.HasOnLimitConditions() {
				// Delegate to the informer to evaluate when/or conditions.
				q.evaluateCond.OnLimit.Store(true)
				return true
			}
		}

		// No conditions declared — drop immediately.
		logger.Warn().
			Str("key", item.Key).
			Str("gvk", item.GVK).
			Int("limit", q.QueueInfo().Limit).
			Int("depth", q.QueueInfo().Depth).
			Int("threshold", cfg.ThresholdValue()).
			Msg("enqueue: queue depth threshold/limit reached — item dropped")
		return false
	}

	return true
}
