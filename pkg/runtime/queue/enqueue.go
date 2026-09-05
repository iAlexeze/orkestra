package queue

// Enqueue adds the object's key to the workqueue.
//
// When no event identity is requested, QueueItem equality is based on Key/GVK,
// so multiple events for the same resource are coalesced by the underlying
// client-go workqueue.
func (q *Workqueue) Enqueue(obj interface{}, gvk string) {
	key := resolveKeyFromCache(obj, gvk)
	if key == "" {
		return
	}

	q.addWithEval(QueueItem{
		Key: key,
		GVK: gvk,
	})
}

// EnqueueKey adds a pre-computed key directly to the workqueue.
// Used when the key is resolved from an ownerReference or another indirect source.
func (q *Workqueue) EnqueueKey(key, gvk string) {
	q.addWithEval(QueueItem{
		Key: key,
		GVK: gvk,
	})
}

// EnqueueWithSentinels adds a key to the workqueue alongside sentinel values
// computed at event time.
//
// This method preserves normal queue deduplication. Sentinel values are stored
// separately from QueueItem, so two sentinel-bearing events for the same
// Key/GVK still compare equal and may be coalesced by the workqueue.
//
// Use EnqueueWithEventSentinels when the reconcile gate has explicitly requested
// event-aware evaluation.
func (q *Workqueue) EnqueueWithSentinels(
	obj interface{},
	gvk string,
	sentinels map[string]string,
) {
	key := resolveKeyFromCache(obj, gvk)
	if key == "" {
		return
	}

	q.enqueueWithSentinels(key, gvk, sentinels, false)
}

// EnqueueWithEventSentinels adds a key to the workqueue with event-aware
// identity.
//
// Each invocation receives a unique EventID. Consequently, two events for the
// same Key/GVK are different QueueItems and survive client-go queue
// deduplication as separate work items.
//
// The associated sentinel values are retained with that event until the item
// is finally forgotten.
func (q *Workqueue) EnqueueWithEventSentinels(
	obj interface{},
	gvk string,
	sentinels map[string]string,
) {
	key := resolveKeyFromCache(obj, gvk)
	if key == "" {
		return
	}

	q.enqueueWithSentinels(key, gvk, sentinels, true)
}

// Sentinels returns the sentinel values associated with a queue item.
//
// A copy is returned so callers cannot mutate queue-owned event context while
// another goroutine is using it.
func (q *Workqueue) Sentinels(item QueueItem) map[string]string {
	if q == nil {
		return nil
	}

	identity := queueItemIdentity{
		Key:     item.Key,
		GVK:     item.GVK,
		EventID: item.EventID,
	}

	q.sentinelMu.RLock()
	sentinels := q.sentinels[identity]
	q.sentinelMu.RUnlock()

	return cloneSentinels(sentinels)
}
