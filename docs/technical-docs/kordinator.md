# Kordinator

`pkg/kordinator` contains the components that sit between the informer factory and the reconcilers: the dependency-ordered lifecycle engine, the CRD registry, per-CRD health tracking, and the HTTP handlers that surface this information at runtime.

---

## Components at a glance

| Type | File | Responsibility |
|---|---|---|
| `DependencyKordinator` | `dependency_kordinator.go` | Starts CRD workers in dependency order; self-heals on CRD add/remove |
| `Kontroller` | `kontroller.go` | Base worker manager — runs reconcile loops per GVK |
| `ResourceKatalog` | `kordinator_registry.go` | Stores `(informer, reconciler factory, CRD entry)` per GVK |
| `CRDHealth` | `crd_health.go` | Tracks reconcile health, worker states, and queue depth per CRD |
| `OrkestraHealth` | `crd_worker_health.go` | Operator-level aggregate health (ready / degraded) |
| `BuildCRDHealthHandler` | `crd_health_handers.go` | `/katalog/{crd}/health` HTTP handler |
| `BuildCRDInfoHandler` | `crd_health_handers.go` | `/katalog/{crd}` HTTP handler |
| `BuildKatalogHandler` | `crd_health_handers.go` | `/katalog` aggregate handler |

---

## DependencyKordinator

`DependencyKordinator` is the last component started. Its `Kordinate()` method is where workers actually begin processing reconcile events. Before it runs, informers warm their caches, but no CRs are reconciled.

```go
type DependencyKordinator struct {
    *Kontroller

    depGraph       *katalog.DependencyGraph
    defaultWorkers int
    startedAt      time.Time
    queueReg       *queue.QueueRegistry
    drainTimeout   time.Duration

    anyOnline atomic.Bool
    allOnline atomic.Bool
    orkHealth *OrkestraHealth

    // startedCh[gvk] is closed when a CRD has fully started its workers.
    startedCh map[string]chan struct{}

    // healthyCh[gvk] is closed after the CRD handles first successful reconcile.
    healthyCh map[string]chan struct{}
}
```

### Constructor

```go
func NewDependencyKordinator(
    kube           *kubeclient.Kubeclient,
    factory        *informer.Factory,
    katalog        *ResourceKatalog,
    events         *event.Event,
    hs             domain.Health,
    queueRegistry  *queue.QueueRegistry,
    defaultWq      *queue.Workqueue,
    crdHealthMap   map[string]*CRDHealth,
    orkHealth      *OrkestraHealth,
    defaultWorkers int,
    depGraph       *katalog.DependencyGraph,
    drainTimeout   time.Duration,
) *DependencyKordinator
```

### Kordinate sequence

```go
func (k *DependencyKordinator) Kordinate(ctx context.Context)
```

1. Calls `orkHealth.SetOrkReady()` — operator can serve HTTP requests immediately
2. Reads topological order from `depGraph.StartupOrder()`
3. Creates `startedCh` and `healthyCh` channels for every CRD
4. Launches `retryMissingCRDs` goroutine in background
5. Launches `dependencyHealthChecker` goroutine in background
6. Iterates over CRDs in topological order — **non-blocking**:
   - If dependencies are not yet satisfied → log and skip (retry loop handles it)
   - If CRD is missing from cluster → skip (retry loop monitors for arrival)
   - Otherwise → call `startCRDWorkers`, then close `startedCh[gvk]`
7. Calls `orkHealth.SetKatalogReady()` or `SetKatalogDegraded()` depending on how many CRDs started
8. Blocks on `<-ctx.Done()` (context cancelled when leadership is lost)
9. Shuts down CRDs in reverse topological order via `stopCRDWorkers`

The non-blocking design is critical: a CRD with a `healthy` dependency may take minutes to satisfy. Blocking the main loop would starve other CRDs that are ready to start immediately.

### Worker loop

Workers run in `runWorkerForGVK` (defined in `worker.go`):

```go
func (k *Kontroller) runWorkerForGVK(ctx context.Context, gvk string, workerID string) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            item, shutdown := wq.Queue.Get()
            if shutdown { return }
            k.crdHealthMap[gvk].MarkWorkerProcessing(workerID)
            func() {
                defer wq.Queue.Done(item)
                k.processItemForGVK(ctx, gvk, item)
            }()
            k.crdHealthMap[gvk].MarkWorkerIdle(workerID)
        }
    }
}
```

Workers mark themselves processing/idle on each item. `wq.Queue.Done(item)` is called in a deferred function — always called whether the reconcile succeeds or fails. Rate-limited requeue is handled inside `processItemForGVK` on error.

### Self-healing: retryMissingCRDs

The retry loop runs for the operator's lifetime. It handles three scenarios:

| Scenario | What happens |
|---|---|
| CRD missing at startup | Polls until CRD appears, then calls `activateCRD` |
| CRD deleted while running | Detects via `crdExists` → calls `deactivateCRD` |
| CRD reappears after deletion | Detected in next retry tick → `activateCRD` again |
| Deferred (dependency not ready) | Re-checks `dependenciesReady` each tick; activates when ready |

`activateCRD` safely closes `startedCh[gvk]` using `select/default` to handle the case where the channel was already closed from a previous activation:

```go
select {
case <-k.startedCh[gvk]:
    // already closed — no-op
default:
    close(k.startedCh[gvk])
}
```

`deactivateCRD` **never** closes `startedCh` — dependents continue running in degraded state rather than blocking.

### Dependency conditions

```go
// dependsOn in Katalog:
dependsOn:
  other-crd:
    condition: started   # default
  another-crd:
    condition: healthy   # wait for first successful reconcile
```

`dependenciesReady` is a non-blocking check on both channel types:

```go
case string(types.DependencyConditionHealthy):
    select {
    case <-k.healthyCh[depGVK]: // closed → healthy
    default: return false
    }
default: // started
    select {
    case <-k.startedCh[depGVK]: // closed → started
    default: return false
    }
```

### Shutdown

On `ctx.Done()`:

```go
shutdownOrder := k.depGraph.ShutdownOrder()
for _, name := range shutdownOrder {
    gvk := k.depGraph.GetNode(name).CRD.GroupVersionKind.String()
    k.stopCRDWorkers(gvk)
}
```

`stopCRDWorkers` cancels the CRD context, shuts down its queue (unblocks any `Queue.Get()` waiting for items), and calls `wg.Wait()` with a `drainTimeout`. Stuck reconciles that exceed the timeout produce a warning log but do not hang the process.

---

## ResourceKatalog

`ResourceKatalog` is the per-GVK registry written once during `konstructOrkestra` and read many times thereafter.

```go
type RegistryEntry struct {
    CRD               orktypes.CRDEntry
    Informer          cache.SharedIndexInformer  // used for resource count in health handlers
    ReconcilerFactory func() domain.Reconciler   // called once per CRD in startCRDWorkers
    DegradeThreshold  int
}

type ResourceKatalog struct {
    mu      sync.Mutex
    entries map[string]RegistryEntry
}
```

```go
func NewKordinatorRegistry() *ResourceKatalog
func (r *ResourceKatalog) Register(gvk string, crd orktypes.CRDEntry, inf cache.SharedIndexInformer, rec func() domain.Reconciler)
func (r *ResourceKatalog) Get(gvk string) (RegistryEntry, bool)
func (r *ResourceKatalog) ListGVKs() []string
func (r *ResourceKatalog) GetWorkers(gvk string, defaultWorkers int) int
func (r *ResourceKatalog) Unregister(gvk string)
```

It is written once during `konstructOrkestra` and read many times:
- `DependencyKordinator.startCRDWorkers` reads `ReconcilerFactory`
- `/katalog` route reads all entries to build the aggregate response
- `/katalog/{crd}` route reads `Informer` for live resource count

---

## CRDHealth

`CRDHealth` tracks the runtime health of one CRD reconciler. Every field that can be read from multiple goroutines uses `atomic` operations or `sync.Map` — no locks in the hot path.

```go
type CRDHealth struct {
    name             string
    started          atomic.Bool
    pending          atomic.Bool
    healthy          atomic.Bool
    degraded         atomic.Bool
    totalReconciles  atomic.Int64
    failedReconciles atomic.Int64
    consecutiveFails atomic.Int64
    lastError        atomic.Value  // string
    lastReconcile    atomic.Value  // time.Time
    startTime        atomic.Value  // time.Time
    queueReg         *queue.QueueRegistry

    // CRD presence tracking
    lastCRDCheck time.Time
    crdExists    atomic.Bool
    crdCheckMu   sync.RWMutex

    // Worker state tracking
    totalWorkers      atomic.Int32
    idleWorkers       atomic.Int32
    processingWorkers atomic.Int32
    workerStates      sync.Map  // workerID → WorkerStateIdle|Processing|Stopped
    gvk               string

    // Dependency health
    dependencies     map[string]DependencyStatus
    dependenciesMu   sync.RWMutex
    hasUnhealthyDeps atomic.Bool
    healthySignaled  atomic.Bool
}
```

Key operations:

```go
// Called from DependencyKordinator.startCRDWorkers
func (h *CRDHealth) SetStarted()
func (h *CRDHealth) SetTotalWorkers(n int32)

// Called from worker loop
func (h *CRDHealth) MarkWorkerProcessing(workerID string)
func (h *CRDHealth) MarkWorkerIdle(workerID string)

// Called from processItemForGVK after reconcile
func (h *CRDHealth) RecordSuccess()
func (h *CRDHealth) RecordFailure(errMsg string)

// Called from BuildCRDHealthHandler / BuildCRDInfoHandler
func (h *CRDHealth) Started() bool
func (h *CRDHealth) Pending() bool
func (h *CRDHealth) IsHealthy() bool
func (h *CRDHealth) QueueDepth() int
func (h *CRDHealth) ErrorRate() float64
```

### Health state

The `/katalog/{crd}/health` endpoint derives state from the atomic fields:

| State | Condition | HTTP |
|---|---|---|
| `not started` | `!started && !pending` | 503 |
| `pending` | `pending` | 200 |
| `started` | `started && !healthy` | 200 |
| `healthy` | `started && healthy` | 200 |
| `degraded` | `degraded` | 503 |

A single successful reconcile transitions the CRD to `healthy` and closes `healthyCh[gvk]` — unblocking dependents that require `condition: healthy`.

---

## HTTP handlers

All three handlers are defined in `crd_health_handers.go`.

### `BuildCRDHealthHandler`

```go
func BuildCRDHealthHandler(
    crd orktypes.CRDEntry,
    kfg *konfig.Konfig,
    inf cache.SharedIndexInformer,
    h   *CRDHealth,
) http.HandlerFunc
```

Returns the live health summary for one CRD as `CRDHealthResponse`. Used by external health checks and `ork status`.

```json
{
  "name": "website",
  "state": "healthy",
  "healthy": true,
  "started": true,
  "queueDepth": 0,
  "errorRate": 0.0,
  "totalReconciles": 142,
  "resourceCount": 7
}
```

### `BuildCRDInfoHandler`

```go
func BuildCRDInfoHandler(
    crd         orktypes.CRDEntry,
    kfg         *konfig.Konfig,
    inf         cache.SharedIndexInformer,
    h           *CRDHealth,
    convStats   *health.ConversionStats,
    admStats    *health.AdmissionStats,
    protStats   *health.ProtectionStats,
    isProtected bool,
    provStats   *health.ProviderStats,
) http.HandlerFunc
```

Returns the full CRD detail as JSON. The response is assembled on every request from live atomic reads — no caching.

When `provStats` is non-nil, the response includes a `providers` array: one entry per declared `ProviderBlock` with the provider name, declared kinds (from Katalog static metadata), and runtime totals + error rate from `provStats.GetSnapshot()`. When no provider blocks are declared, the field is omitted.

`protStats` may be nil — if so, a zero `ProtectionStatsResponse` with `enabled: isProtected` is returned.

Resource count comes from `len(inf.GetStore().List())` — the informer's local cache, updated in real time.

### `BuildKatalogHandler`

```go
func BuildKatalogHandler(
    kat       *katalog.Katalog,
    kfg       *konfig.Konfig,
    registry  *ResourceKatalog,
    healthMap map[string]*CRDHealth,
) http.HandlerFunc
```

Aggregates all CRDs into one response — the source of truth for `ork status`. Includes the dependency graph, aggregate health (degraded if any CRD is degraded), and a summary row per CRD. Each row includes `providerCount` (number of declared provider blocks) when non-zero.

---

## Dependency graph

Built from the `dependsOn` fields of all enabled CRD entries in the Katalog. Implemented in `pkg/katalog/dependency_graph.go`.

```go
type DependencyGraph struct { ... }

func NewDependencyGraph(kat *katalog.Katalog) *DependencyGraph
func (g *DependencyGraph) StartupOrder() []string   // Kahn's topological sort
func (g *DependencyGraph) ShutdownOrder() []string  // reverse of StartupOrder
func (g *DependencyGraph) GetNode(name string) *DependencyNode
```

Kahn's algorithm produces a deterministic topological order. CRDs with no remaining dependencies are sorted alphabetically for predictability. A cycle is detected when the algorithm terminates with unprocessed nodes — this is a fatal error at startup.

The `DependencyKordinator` reads `StartupOrder()` once at the beginning of `Kordinate()` and caches a `nameToGVK` map. Workers start in that order; shutdown uses `ShutdownOrder()` (the reverse).

---

## QueueRegistry

Lives in `pkg/queue`, used by the kordinator to manage one workqueue per CRD.

```go
// Creates and returns the per-CRD queue for a GVK.
func (r *QueueRegistry) Register(gvk string) *Workqueue

// Returns the queue for a GVK. Used by informer event handlers and workers.
func (r *QueueRegistry) For(gvk string) (*Workqueue, bool)

// Shuts down all queues — called from stopCRDWorkers.
func (r *QueueRegistry) ShutDownAll()
```

Three event handlers are registered per informer (`AddFunc`, `UpdateFunc`, `DeleteFunc`) and call `wq.Queue.Add(key)`. The workqueue deduplicates: if `"default/my-website"` is already queued, a second `Add` is a no-op. This is what makes level-triggered reconciliation efficient — three rapid updates to the same CR produce one reconcile.

---

## Key design rules

| Rule | Reason |
|---|---|
| `startedCh` is never closed during deactivation | Dependents continue running degraded rather than blocking |
| Main `Kordinate` loop never blocks on dependency conditions | A `healthy` dependency can take minutes; blocking would starve other CRDs |
| `activateCRD` closes `startedCh` with `select/default` | Channel may already be closed from a previous activation cycle |
| `stopCRDWorkers` shuts down the queue before calling `wg.Wait()` | Workers blocked on `Queue.Get()` waiting for items would otherwise never exit |
| Drain timeout is a safety net, not an execution budget | Normal reconciles finish well before the timeout; only stuck external API calls hit it |
