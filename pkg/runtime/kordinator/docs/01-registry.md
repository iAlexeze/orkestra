# 01 — ResourceKatalog

`ResourceKatalog` is the per-GVK registry at the centre of kordinator. It is the answer to the question: "what do I need to start reconciling this CRD?"

## What it holds

Each GVK maps to a `RegistryEntry`:

```go
type RegistryEntry struct {
    CRD               orktypes.CRDEntry
    Informer          cache.SharedIndexInformer
    ReconcilerFactory func() domain.Reconciler
    DegradeThreshold  int
}
```

- `CRD` — the full Katalog configuration for this CRD: group, version, kind, worker count, queue settings, dependency declarations, provider blocks
- `Informer` — the running informer for this CRD; used by the worker loop to get resource count and by the health handlers to serve live CR counts without any API calls
- `ReconcilerFactory` — a closure that builds a new `Reconciler` when called; captures the provider registry, kube client, and per-CRD provider stats at construction time so none of those dependencies pass through kordinator's own API
- `DegradeThreshold` — how many consecutive failures before this CRD's health transitions to degraded

## When it is written

`ResourceKatalog` is written exactly once during `konstructRuntime`, before `DependencyKordinator` starts. The factory loop calls `Register` once per enabled CRD:

```go
resourceKatalog.Register(gvk, crd, informer, func() domain.Reconciler {
    return reconciler.NewGenericReconciler(
        gvk, kube, ev, kfg, kat, providerRegistry, provStats,
    )
})
```

After `konstructRuntime` returns, the registry is never written again. All subsequent access is reads.

## How it is read

Three callers read from `ResourceKatalog` during the operator's lifetime:

**`DependencyKordinator.startCRDWorkers`** calls `katalog.Get(gvk)` and invokes `entry.ReconcilerFactory()` to produce the reconciler for each new worker.

**`/katalog/{crd}` handler** calls `katalog.Get(gvk)` to read the static CRD configuration (provider blocks, dependency declarations, queue settings) for the response body.

**`runWorkerForGVK`** calls `katalog.Get(gvk)` after each reconcile to read `entry.Informer.GetIndexer().List()` and update the resource count metric.

## API

```go
func NewKordinatorRegistry() *ResourceKatalog

func (r *ResourceKatalog) Register(
    gvk string,
    crd orktypes.CRDEntry,
    inf cache.SharedIndexInformer,
    rec func() domain.Reconciler,
)

func (r *ResourceKatalog) Get(gvk string) (RegistryEntry, bool)
func (r *ResourceKatalog) ListGVKs() []string
func (r *ResourceKatalog) GetWorkers(gvk string, defaultWorkers int) int
func (r *ResourceKatalog) Unregister(gvk string)
```

`GetWorkers` returns `entry.CRD.Workers` if set, otherwise `defaultWorkers`. This is the value passed to `startCRDWorkers` so each CRD can tune its own concurrency from the Katalog.

---

**Next →** [02 — CRDHealth](02-health.md)
