# 01 — Workqueue

`Workqueue` is the per-CRD work buffer. It wraps a `workqueue.TypedRateLimitingInterface[QueueItem]`
and adds two things: an atomic depth limit and structured logging on Enqueue.

## Structure

```go
type Workqueue struct {
    name          string
    Queue         workqueue.TypedRateLimitingInterface[QueueItem]
    started       atomic.Bool
    maxQueueDepth atomic.Int32  // 0 = unlimited
}
```

`Queue` is the client-go rate-limiting workqueue. It handles deduplication,
rate limiting on re-queues, and the `Done`/`Forget` acknowledgement protocol.

`maxQueueDepth` is an `atomic.Int32` so reads in `Enqueue` (called from
informer goroutines) and writes in `SetMaxQueueDepth` (called from the
autoscaler goroutine) are race-free without a mutex.

## QueueItem

```go
type QueueItem struct {
    Key string  // namespace/name or just name for cluster-scoped CRDs
    GVK string  // GVK string identifies which CRD owns this item
}
```

The `GVK` field exists because the default queue is shared across all CRDs
(before per-CRD queues are registered). Workers that read from the default queue
check `item.GVK` against their own GVK and re-queue items that belong elsewhere.

## Enqueue

```go
func (q *Workqueue) Enqueue(obj interface{}, gvk string) {
    // Handle tombstone (deleted objects from informer cache)
    if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
        obj = tombstone.Obj
    }

    key, err := cache.MetaNamespaceKeyFunc(obj)
    if err != nil { ... return }

    // Depth limit: drop if at or beyond limit (0 = unlimited)
    if limit := q.maxQueueDepth.Load(); limit > 0 && int32(q.Queue.Len()) >= limit {
        logger.Warn(). ... .Msg("enqueue: queue depth limit reached — item dropped")
        return
    }

    q.Queue.Add(QueueItem{Key: key, GVK: gvk})
}
```

The depth check is a soft ceiling — it applies only to incoming enqueues. Items
already in the queue are not evicted. This matches the autoscaler use case:
when load drops and the override reverts, the limit goes back to 0 (unlimited)
and the queue drains normally.

The tombstone handling is required for deleted objects. When the informer's
local cache misses the final state of a deleted object, the informer wraps it
in a `DeletedFinalStateUnknown` tombstone. Without unwrapping, `MetaNamespaceKeyFunc`
would fail on the wrapper type.

## SetMaxQueueDepth

```go
func (q *Workqueue) SetMaxQueueDepth(n int) {
    q.maxQueueDepth.Store(int32(n))
}
```

Called by `GenericReconciler.SetQueueDepthLimit` when the autoscaler applies or
reverts a `do.queueDepth` override. The change is immediately visible to the
next `Enqueue` call — no lock needed.

## Rate limiting and acknowledgement

The workqueue's rate limiter applies exponential backoff when `AddRateLimited`
is called (i.e., on reconcile failure). The sequence is:

```
reconcile succeeds → Queue.Forget(item)     ← resets backoff counter
reconcile fails    → Queue.AddRateLimited(item)  ← re-queues with backoff
```

`Forget` is important: without it, a key that fails once and then succeeds on
retry carries a residual failure penalty for future failures. `Forget` resets
this so the next failure starts fresh.

`Done(item)` must be called after every dequeue — whether the reconcile
succeeded, failed, or panicked. The `processItemForGVK` loop calls `Done` via
`defer` inside a closure to ensure this even on panic.

---

**Next →** [02 — Registry](02-registry.md)
