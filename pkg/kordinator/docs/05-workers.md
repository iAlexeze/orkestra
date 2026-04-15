# 05 — Workers and drain

Workers are the innermost loop of Orkestra. Each worker goroutine runs `runWorkerForGVK` for the lifetime of its CRD's context. The worker's only job is to dequeue items and call `processItemForGVK`.

## The worker loop

```go
func (k *Kontroller) runWorkerForGVK(ctx context.Context, gvk string, workerID string) {
    wq, ok := k.queueRegistry.For(gvk)
    if !ok {
        wq = k.defaultWorkqueue
    }

    for {
        select {
        case <-ctx.Done():
            return
        default:
            item, shutdown := wq.Queue.Get()
            if shutdown {
                return
            }

            k.crdHealthMap[gvk].MarkWorkerProcessing(workerID)
            func() {
                defer wq.Queue.Done(item)
                k.processItemForGVK(ctx, gvk, item)
            }()
            k.crdHealthMap[gvk].MarkWorkerIdle(workerID)

            // Update metrics after each item
            metrics.SetQueueDepth(gvk, float64(wq.Depth()))
            if entry, ok := k.katalog.Get(gvk); ok && entry.Informer != nil {
                metrics.SetResourceCount(gvk, float64(len(entry.Informer.GetIndexer().List())))
            }
        }
    }
}
```

`wq.Queue.Done(item)` is called inside a deferred closure — it runs whether `processItemForGVK` returns normally or panics. Without this, a panic would permanently remove an item from the queue without ever marking it done, causing the workqueue's internal bookkeeping to diverge.

The worker exits in two ways: the context is cancelled (`ctx.Done()`), or the queue is shut down (`shutdown == true`). Both paths lead to `return` — the WaitGroup is decremented and the goroutine exits.

## processItemForGVK

`processItemForGVK` resolves the reconciler and calls it. On error, it re-queues the item with rate-limit backoff:

```go
wq.Queue.AddRateLimited(item)
health.RecordFailure(err.Error())
```

On success:

```go
wq.Queue.Forget(item)
health.RecordSuccess()
```

`Forget` removes the item from the rate-limiter's failure tracking — the next time the same key is enqueued, it starts fresh without exponential backoff penalty.

## Queue and worker count per CRD

Every CRD gets its own queue. Workers for a CRD read only from that CRD's queue. This means a slow CRD (large CRs, slow external API calls) cannot starve workers for a fast CRD.

Worker count is configured per CRD in the Katalog:

```yaml
spec:
  crds:
    website:
      workers: 5        # 5 goroutines for this CRD
    route-record:
      workers: 2
```

`ResourceKatalog.GetWorkers(gvk, defaultWorkers)` returns the configured count, or the operator-wide default if the CRD does not specify one.

When `autoscale:` is declared, `startCRDWorkers` starts
`max(baseline.workers, do.workers)` goroutines instead, so scale-up is instant
— no new goroutines are spawned at runtime.

## Semaphore-gated concurrency

For all CRDs, `GenericReconciler.Reconcile` acquires a `ResizableSemaphore`
before doing any work and releases it when done:

```
worker goroutine  →  dequeue  →  processItemForGVK  →  rec.Reconcile(ctx, key)
                                                              │
                                                     workerSem.Acquire()  ← blocks if at capacity
                                                     reconcileCore()
                                                     workerSem.Release()
```

For non-autoscaled CRDs the semaphore capacity equals the worker count so it is
always uncontested. For autoscaled CRDs the autoscaler resizes the semaphore
without touching the goroutines — goroutines blocked at `Acquire` wake up
immediately on scale-up; excess goroutines finish their current reconcile and
then block on the next `Acquire` during scale-down.

## Queue depth reporting

After each item, the worker loop type-asserts the reconciler against
`queueDepthReporter` and calls `ReportQueueDepth`:

```go
if reporter, ok := rec.(queueDepthReporter); ok {
    reporter.ReportQueueDepth(int64(wq.Depth()))
}
```

This pushes live depth into `AutoMetrics` so `metrics.queueDepth` conditions
in the autoscaler evaluate against the actual queue state rather than a stale
snapshot. The interface is defined locally in `worker.go` to avoid importing
the `reconciler` package.

## stopCRDWorkers

`stopCRDWorkers` is called during both `deactivateCRD` (runtime) and the shutdown sequence. The order of operations matters:

```go
// Step 1: cancel the CRD context
cancel()

// Step 2: shut down the queue
wq.Queue.ShutDown()

// Step 3: wait for workers to drain
wg.Wait()  // with drainTimeout
```

Step 2 is required. A worker that has finished its current item and is blocking on `wq.Queue.Get()` waiting for the next item will never observe the context cancellation from Step 1 — `Get()` does not accept a context. The queue shutdown unblocks `Get()` by returning `shutdown = true`, which causes the worker to exit on the next iteration.

Without Step 2, workers that are idle at shutdown time would hang until a new item arrived or the drain timeout expired.

## Drain timeout

`stopCRDWorkers` waits on the WaitGroup with a configurable timeout:

```go
select {
case <-done:
    logger.Info().Str("gvk", gvk).Msg("workers drained cleanly")
case <-time.After(k.drainTimeout):
    logger.Warn().Str("gvk", gvk).
        Dur("timeout", k.drainTimeout).
        Msg("drain timeout exceeded — workers may still be running")
}
```

The timeout is a safety net, not an execution budget. Normal reconciles finish well within the timeout. The warning fires only when a reconcile is blocked on a slow external API call that does not respect context cancellation. The fix is to ensure all external calls use the context passed to `rec.Reconcile(ctx, req)`.

## Worker state constants

```go
const (
    WorkerStateIdle       = "idle"
    WorkerStateProcessing = "processing"
    WorkerStateStopped    = "stopped"
)
```

These are the values stored in `CRDHealth.workerStates` (a `sync.Map`). The `/katalog/{crd}` handler reads them for the worker breakdown in the response body.

---

**Next →** [06 — HTTP handlers](06-handlers.md)
