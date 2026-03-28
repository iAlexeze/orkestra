# Informer Factory and Queue Registry

Every CRD managed by Orkestra gets its own dedicated informer and workqueue. This document explains how they are created, how they work together, and why per-CRD isolation at this level is the architectural foundation of the super-operator model.

---

## The informer

An informer is a long-lived background process that:
1. Lists all existing objects of a specific GVK from the API server (initial sync)
2. Opens a watch connection to receive change events in real time
3. Maintains a local in-memory cache of all objects
4. Calls registered event handlers when objects change

Orkestra creates one informer per CRD entry via the informer factory:

```go
// Each informer watches exactly one GVK with its own resync interval
informer := factory.ForResource(gvr).Informer()
informer.AddEventHandlerWithResyncPeriod(handler, resyncInterval)
```

The informer is keyed by the exact GVK in the Katalog entry. An informer registered for `demo.orkestra.io/v1alpha1/Website` receives objects in `v1alpha1` format — the API server converts from storage to this version for each watch event. This is why each CRD version gets its own informer entry: `website-v1alpha1` and `website-v1` have different informers watching different versions of the same Kind.

### What the informer cache provides

The informer cache eliminates redundant API calls. When the reconciler processes a work item, it reads the object from the local cache rather than making an API call:

```go
// In the worker — reads from cache, zero API call
obj, exists, err := r.informer.GetStore().GetByKey(namespace + "/" + name)
```

This is the "read from cache, write to API" pattern — the reconciler reads the desired state from the cache (cheap, fast, no API quota) and writes the reconciled state back to the API server (API call, necessary).

### Resync

The resync interval triggers a full re-list of all objects at the configured interval. Every object in the cache is re-queued for reconciliation even without a change event. This is the drift correction mechanism — objects whose child resources were manually changed will be corrected on the next resync.

Default: 15 seconds. Configurable per-CRD via `resync: 5m`.

!!! tip "Resync vs reconcile triggers"
    Resync is not the only way reconciliation is triggered. An Update event on
    the CR (e.g. spec change), a change to a child resource via owner reference
    watch, or an explicit `ork reconcile` command all trigger reconciliation
    independently of the resync interval.

---

## The workqueue

Each CRD has its own `workqueue.RateLimitingInterface`:

```go
queue := workqueue.NewRateLimitingQueue(
    workqueue.NewItemExponentialFailureRateLimiter(
        baseDelay,   // 5ms default
        maxDelay,    // 1000s default
    ),
)
```

The event handler adds items to the queue as `namespace/name` key strings. The queue:

- **Deduplicates** — if an object is updated three times before a worker processes it, only one reconcile runs (for the latest state)
- **Rate limits** — failed items are requeued with exponential backoff
- **Isolates** — each CRD's queue is independent; a backlog in one CRD cannot starve another

### Queue depth

`maxQueueDepth` sets the maximum items the queue will hold. When the queue is full, new items are dropped with a warning. This prevents memory exhaustion when a CRD has a very large number of CRs all needing reconciliation simultaneously.

Default: 2000. The current queue depth is reported by `controller_queue_depth{crd}` and visible in `ork status`.

### Degrade threshold

`degradeThreshold` is the number of consecutive reconcile failures before the CRD health transitions to degraded. Degraded is not fatal — workers continue running. It is a signal that something is systematically wrong with reconciliation for this CRD.

Default: 10 consecutive failures.

---

## The worker pool

Each CRD has `workers` goroutines processing items from its queue:

```go
for i := 0; i < entry.Workers; i++ {
    go r.runWorker(ctx, stopCh)
}
```

Each worker runs a tight loop: `queue.Get()`, `safeReconcile()`, `queue.Done()` (success) or `queue.AddRateLimited()` (failure). Workers are completely independent — they do not share state except through the Kubernetes API server.

The worker count is configurable per-CRD (`workers: 8`). More workers increase throughput — multiple CRs can be reconciled simultaneously. However, more workers also mean more concurrent API calls. For clusters with API server rate limits, a high worker count across many CRDs can cause rate limiting.

Default: 3 workers per CRD.

---

## Dependency ordering

CRDs with `dependsOn` declarations are started in topological order. The dependency resolution:

1. Builds a directed graph from all `dependsOn` declarations
2. Detects cycles (fatal error if found)
3. Computes topological order
4. Starts CRDs in that order, waiting for each to signal readiness before starting dependents

**Readiness signal:** A CRD is ready when its informer has completed its initial sync (list + cache populated) and its workers have started. The ready signal is sent on a channel that dependent CRDs wait on.

**Missing dependencies:** If a CRD declared in `dependsOn` is not present in the current Katalog, the dependent CRD waits in the background with periodic retries. It does not block other independent CRDs from starting.

```
CRD A (no deps) ─── starts immediately ────────────────► running
CRD B (no deps) ─── starts immediately ────────────────► running
CRD C (depends on A, B) ─── waits for A and B ─────────► running after A and B ready
```

---

## Shutdown sequence

When Orkestra receives SIGTERM or the context is cancelled, shutdown runs in reverse dependency order:

1. Workers are signalled to stop accepting new items (`queue.ShutDown()`)
2. In-flight reconciles complete (workers drain their current items)
3. Workers exit
4. Informers stop watching
5. The HTTPS server drains open connections
6. The HTTP server shuts down

This ensures that no reconcile is abandoned mid-execution. A CR that is being reconciled when SIGTERM arrives will complete its reconcile before the process exits.

!!! note "In-flight reconcile timeout"
    There is no explicit timeout on in-flight reconciles during shutdown. If a
    reconcile is waiting on a slow external API call or has entered an infinite
    loop, shutdown will wait indefinitely. Use `context.WithTimeout` in hooks
    to prevent this.
