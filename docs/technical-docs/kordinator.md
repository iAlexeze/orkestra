# Kordinator

`pkg/kordinator` contains the components that sit between the informer factory and the reconcilers: the dependency-ordered worker startup system, the queue and Kordinator registries, and the per-CRD health tracking.

---

## Components at a glance

| Type | File | Responsibility |
|---|---|---|
| `DependencyKordinator` | `dependency_kordinator.go` | Starts CRD workers in dependency order |
| `KordinatorRegistry` | `Kordinator_registry.go` | Stores (informer, factory) per GVK |
| `QueueRegistry` | `queue_registry.go` | Creates and owns per-CRD workqueues |
| `CRDHealth` | `health_tracker.go` | Tracks reconcile success/failure per CRD |
| `BuildCRDHealthHandler` | `handlers.go` | `/katalog/{crd}/health` HTTP handler |
| `BuildCRDInfoHandler` | `handlers.go` | `/katalog/{crd}` HTTP handler |
| `BuildKatalogHandler` | `handlers.go` | `/katalog` aggregate handler |

---

## DependencyKordinator

`DependencyKordinator` is the last komponent started. Its `Start()` method is where workers actually begin processing reconcile events. Before it starts, informers run and caches warm, but no CRs are reconciled.

### Start sequence

```go
func (k *DependencyKordinator) Start(ctx context.Context) error
```

1. Waits for `infFactory.WaitForCacheSync(ctx)` — all informers must complete their initial list before any worker starts
2. Starts CRDs in topological order from the dependency graph
3. For each CRD, in order:
   - Calls `factory()` to produce the `domain.Reconciler`
   - Starts `workers` goroutines, each running `runWorker`
   - Waits for the CRD's ready signal (informer synced) before starting the next dependent CRD
4. Calls `hs.SetReady()` after all CRDs are started — this makes `/ready` return 200

### runWorker loop

```go
func (k *DependencyKordinator) runWorker(ctx context.Context, rec domain.Reconciler, wq workqueue.RateLimitingInterface, health *CRDHealth) {
    for {
        key, shutdown := wq.Get()
        if shutdown {
            return
        }
        err := safeReconcile(ctx, rec.Reconcile, key.(string))
        if err != nil {
            health.RecordFailure()
            wq.AddRateLimited(key)
        } else {
            health.RecordSuccess()
            wq.Done(key)
        }
    }
}
```

The worker calls `safeReconcile` (which wraps `rec.Reconcile` in a panic-catching defer), records the outcome to `CRDHealth`, and either requeues with backoff or marks done. `wq.Done(key)` is called only on success — this is correct client-go workqueue semantics.

### Shutdown

On context cancellation:

```go
func (k *DependencyKordinator) Shutdown(ctx context.Context) {
    // Signal all queues to stop accepting new items
    k.queueRegistry.ShutDownAll()
    k.defaultWq.ShutDown()
    // Workers drain — each runWorker exits when Get() returns shutdown=true
    k.wg.Wait() // blocks until all workers have exited
}
```

`wg.Wait()` blocks until every worker goroutine has exited. There is no timeout — a worker blocked on a slow reconcile will hold shutdown. Use `context.WithTimeout` in reconcile functions that call external APIs.

---

## KordinatorRegistry

`KordinatorRegistry` is a simple in-memory map from GVK string to a `RegistryEntry`:

```go
type RegistryEntry struct {
    CRD      orktypes.CRDEntry
    Informer cache.SharedIndexInformer  // used by health handlers for resource count
    Factory  func() domain.Reconciler  // called once per CRD in Start()
}

type KordinatorRegistry struct {
    mu      sync.RWMutex
    entries map[string]*RegistryEntry
}
```

It is written once during `konstructOrkestra` and read many times:
- `DependencyKordinator.Start()` reads it to start workers
- `/katalog` route reads it to build the aggregate health response
- `/katalog/{crd}` route reads the `Informer` for live resource count

The map is never modified after `konstructOrkestra` returns — the `sync.RWMutex` exists for safe concurrent reads from the HTTP handlers.

```go
func (r *KordinatorRegistry) Register(gvk string, crd orktypes.CRDEntry, inf cache.SharedIndexInformer, factory func() domain.Reconciler)
func (r *KordinatorRegistry) Get(gvk string) (*RegistryEntry, bool)
func (r *KordinatorRegistry) All() []*RegistryEntry
```

---

## QueueRegistry

`QueueRegistry` creates, owns, and provides access to one workqueue per CRD:

```go
type QueueRegistry struct {
    mu     sync.RWMutex
    queues map[string]workqueue.RateLimitingInterface
}
```

```go
// Register creates a new rate-limiting workqueue for the given GVK.
// maxDepth is the max number of items before new items are dropped.
// Called once per CRD in konstructOrkestra.
func (r *QueueRegistry) Register(gvk string, maxDepth int) workqueue.RateLimitingInterface

// Get returns the queue for a GVK. Used by informer event handlers.
func (r *QueueRegistry) Get(gvk string) (workqueue.RateLimitingInterface, bool)

// ShutDownAll calls ShutDown() on every registered queue.
// Called during DependencyKordinator.Shutdown() to signal all workers to stop.
func (r *QueueRegistry) ShutDownAll()
```

### How events reach queues

The informer factory registers an event handler for each CRD that calls `handleEvent`:

```go
func handleEvent(obj interface{}, queueRegistry *QueueRegistry, gvk string) {
    key, err := cache.MetaNamespaceKeyFunc(obj)
    if err != nil {
        return
    }
    if wq, ok := queueRegistry.Get(gvk); ok {
        wq.Add(key) // deduplicated by the workqueue
    }
}
```

Three handlers are registered per informer — `AddFunc`, `UpdateFunc`, `DeleteFunc` — all calling `handleEvent`. The workqueue's `Add` deduplicates: if `"default/my-website"` is already queued, a second `Add` is a no-op.

This deduplication is what makes level-triggered reconciliation work efficiently. Three rapid updates to the same CR produce one reconcile, not three.

### Start and Shutdown as a Komponent

`QueueRegistry` implements `domain.Komponent`. Its `Start()` starts the internal rate limiter goroutines. Its `Shutdown()` calls `ShutDownAll()` and then `Wait()` to confirm all queues are drained.

---

## CRDHealth

`CRDHealth` tracks the reconcile health of one CRD. It uses `sync/atomic` for all counter operations — no locks, safe for high-frequency updates from multiple worker goroutines.

```go
type CRDHealth struct {
    Name string

    // Atomic counters — updated on every reconcile
    totalReconciles    atomic.Int64
    successReconciles  atomic.Int64
    failureReconciles  atomic.Int64
    consecutiveFails   atomic.Int64

    // Health state — updated when consecutiveFails crosses thresholds
    state atomic.Int32  // 0=starting, 1=healthy, 2=degraded
}
```

```go
func (h *CRDHealth) RecordSuccess() {
    h.totalReconciles.Add(1)
    h.successReconciles.Add(1)
    h.consecutiveFails.Store(0)         // reset on success
    h.state.Store(int32(StateHealthy))  // recover from degraded
}

func (h *CRDHealth) RecordFailure(threshold int) {
    h.totalReconciles.Add(1)
    h.failureReconciles.Add(1)
    fails := h.consecutiveFails.Add(1)
    if int(fails) >= threshold {
        h.state.Store(int32(StateDegraded))
    }
}
```

`RecordSuccess` resets `consecutiveFails` to zero and recovers health state. A CRD that was degraded recovers immediately on the first successful reconcile — there is no hysteresis. This is intentional: a transient spike of failures that resolves should not leave the CRD degraded.

### Health states

```go
const (
    StateStarting  = 0 // informer not yet synced, workers not started
    StateHealthy   = 1 // at least one successful reconcile, no current streak of failures
    StateDegraded  = 2 // consecutiveFails >= degradeThreshold
)
```

The HTTP handler returns:
- `200 {"healthy": true}`  when state is `StateHealthy`
- `503 {"healthy": false}` when state is `StateStarting` or `StateDegraded`

---

## HTTP handlers

### `BuildCRDHealthHandler`

```go
func BuildCRDHealthHandler(crd orktypes.CRDEntry, health *CRDHealth) http.HandlerFunc
```

Returns a handler that reads `health.state` atomically and writes:
```json
{"healthy": true}   // 200
{"healthy": false}  // 503
```

Used by external health checks and `ork status`.

### `BuildCRDInfoHandler`

```go
func BuildCRDInfoHandler(
    crd    orktypes.CRDEntry,
    kfg    *konfig.Konfig,
    inf    cache.SharedIndexInformer,
    health *CRDHealth,
    convStats *health.ConversionStats,
    admStats  *health.AdmissionStats,
) http.HandlerFunc
```

Returns the full CRD detail as JSON. The response is assembled on every request — it reads live counters from `health`, live resource count from `inf.GetStore().List()`, and live stats snapshots from `convStats.GetStats()` and `admStats.GetStats(webhooksEnabled)`.

!!! note "`inf.GetStore().List()` is the resource count"
    The informer's local cache contains all CRs of this type. `len(inf.GetStore().List())`
    is the current CR count — updated in real time as CRs are created and deleted.
    Zero API calls needed.

### `BuildKatalogHandler`

```go
func BuildKatalogHandler(
    kat      *katalog.Katalog,
    kfg      *konfig.Konfig,
    registry *KordinatorRegistry,
    healthMap map[string]*CRDHealth,
) http.HandlerFunc
```

Aggregates all CRDs into one response — the source of truth for `ork status`. Includes the dependency graph, aggregate health (degraded if any CRD is degraded), and a summary row per CRD.

---

## Dependency graph

```go
// pkg/katalog/dependency_graph.go
type DependencyGraph struct {
    order []string  // CRD names in topological start order
    deps  map[string][]string  // name → names it depends on
}

func NewDependencyGraph(kat *Katalog) *DependencyGraph
```

Built from the `dependsOn` fields of all enabled CRD entries. The topological sort uses Kahn's algorithm — iteratively removes nodes with no remaining dependencies and appends them to the order. A cycle is detected when the algorithm terminates with unprocessed nodes.

The `DependencyKordinator` reads the `order` slice to start workers in the correct sequence. For each CRD name in `order`, it looks up the informer and factory from `KordinatorRegistry` and starts the workers.

The ready channel mechanism:

```go
// Each CRD gets a ready channel
readyChannels := make(map[string]chan struct{})
for _, name := range graph.order {
    readyChannels[name] = make(chan struct{})
}

// When starting CRD C which depends on A and B:
for _, dep := range graph.deps["C"] {
    <-readyChannels[dep]  // wait for A, then B
}
// Now start C's workers
startWorkers(...)
close(readyChannels["C"])  // signal to C's dependents
```

This ensures that a dependent CRD never starts reconciling before its dependencies have a warm cache.
