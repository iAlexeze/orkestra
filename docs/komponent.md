# Orkestra Komponents

Orkestra is composed of discrete komponents — each with a single
responsibility, a defined startup order, and a clean shutdown path.
This document describes each one and how they connect.

---

## Komponent index

| Komponent | Package | Responsibility |
|---|---|---|
| HealthServer | `pkg/health` | Liveness, readiness, Katalog API, metrics |
| Kubeclient | `pkg/kubeclient` | REST client, dynamic client, clientset |
| EventRecorder | `pkg/event` | Kubernetes events |
| QueueRegistry | `pkg/queue` | Per-CRD isolated workqueues |
| DefaultWorkqueue | `pkg/queue` | Shared fallback queue |
| SharedInformerFactory | `pkg/informer` | Per-CRD informers with per-CRD resync |
| DependencyKontroller | `pkg/kontroller` | Topological startup, event dispatch, workers |
| KonductorElection | `pkg/konductor` | Leader election — post-start hook |

Startup is sequential in this order. Shutdown runs in reverse.
`KonductorElection` is a post-start hook — it starts after all komponents
are ready and runs the kontroller only on the elected leader.

---

## HealthServer

`pkg/health`

First to start, last to stop. Routes are registered before `Start()` — the
HTTP mux is the same object used at both registration time and request time.

Routes registered by `konstructOrkestra`:

```
GET /healthz                 Orkestra Liveness — 200 always when running
GET /readyz                  Orkestra Readiness — 200 after all komponents ready, 503 before
GET /metrics                 Prometheus metrics endpoint
GET /katalog                 All CRDs — health, config, dependency graph
GET /katalog/{crd}           Single CRD — config + live reconcile stats
GET /katalog/{crd}/health    200 healthy / 503 degraded
```

The `/katalog/*` routes are registered per-CRD in `konstructOrkestra`
before the HealthServer starts. The `crdHealthMap` is shared across the
kontroller (writes) and the route handlers (reads).

---

## Kubeclient

`pkg/kubeclient`

Generic Kubernetes client wrapping REST, dynamic, and clientset clients.
Started early — downstream services need the REST config and dynamic client
during wiring, before `orkestra.Start()` runs.

**Capabilities:**

```go
kube.RESTClient()           // configured with full scheme — typed CRDs
kube.DynamicClient()        // for *unstructured.Unstructured — dynamic CRDs
kube.Clientset()            // for built-in Kubernetes types
kube.RestConfig()           // *rest.Config — used by SharedInformerFactory
kube.NewClient(...)         // creates a REST client for any CRD (typed path)
kube.NewDynamicListerWatcher(...) // creates a ListerWatcher via dynamic client
kube.PatchFinalizers(...)   // JSON merge patch for finalizer add/remove
kube.NewClientProvider()    // creates a ClientProvider for typed CRD registration
```

**Context injection:**

```go
// konstructOrkestra injects kube into context before calling reconcile hooks
ctx = kubeclient.WithKubeclient(ctx, kube)
kube, ok := kubeclient.FromContext(ctx)
```

OrkestraRegistry functions retrieve kube from context — hook signatures
stay clean without kube as an explicit parameter.

---

## EventRecorder

`pkg/event`

Broadcasts Kubernetes events. Depends on kubeclient being started.

Events appear in `kubectl describe` and `kubectl get events --watch`.
Used by GenericReconciler for lifecycle events (reconciled, deleted,
finalizer added/removed) and by KonductorElection for leadership transitions.

---

## QueueRegistry and DefaultWorkqueue

`pkg/queue`

**QueueRegistry** creates and holds a dedicated workqueue per CRD GVK.
Each queue has its own depth metric and exponential backoff.

**DefaultWorkqueue** is the shared fallback for CRDs with `queue.default: true`.

```yaml
# Per-CRD queue (default)
queue:
  maxQueueDepth: 500        # — maximum number of items in the per-CRD queue
  degradeThreshold: 5       # — number of consecutive reconcile failures before the CRD health state transitions from healthy to degraded

# Use shared default queue instead
queue:
  default: true                 
  # You can still configure queuedepth and degradeThreshold — 0 uses the default
```

Queue items carry both the object key (`namespace/name`) and the GVK so
the DependencyKontroller can route to the correct reconciler.

---

## SharedInformerFactory

`pkg/informer`

Creates and manages `cache.SharedIndexInformer` instances per CRD.
The factory routes API server events into the correct queue via
event handlers registered at creation time.

**Two paths depending on CRD mode:**

**Typed CRD** — uses `ClientProvider` (REST client registered per-type):

```go
inf = infFactory.For(object, ctx, opts)
```

**Dynamic CRD** — uses `NewDynamicListerWatcher` (bypasses scheme):

```go
lw := kube.NewDynamicListerWatcher(crdInfo)
inf = infFactory.ForListerWatcher(lw, object, ctx, opts)
```

Both produce a `cache.SharedIndexInformer`. The rest of the system is
identical — the same KontrollerRegistry, DependencyKontroller, and
GenericReconciler work with either.

Per-CRD resync is applied per informer:

```go
infFactory.For(object, ctx, informer.Options{
    Name:   crd.APITypes.Kind,
    Resync: crd.Resync,    // per-CRD — overrides the factory default
    Wq:     wq,
})
```

The factory is not started in `konstructOrkestra`. `orkestra.Start()`
calls `infFactory.Start()` in the correct sequence.

---

## KontrollerRegistry

`pkg/kontroller`

Maps GVK to runtime components — informer, reconciler factory, and CRD
metadata. Built by `konstructOrkestra` before the DependencyKontroller
is created.

```go
ktrlRegistry.Register(gvk, crd, inf, factory)
entry, ok := ktrlRegistry.Get(gvk)
```

The reconciler `factory` is a closure — it captures `kube`, `ev`, `inf`,
and `crd` but is not called until `startCRDWorkers` runs after
`orkestra.Start()` guarantees everything is live.

---

## DependencyKontroller

`pkg/kontroller`

Orchestrates CRD lifecycle in topological dependency order.
The most complex komponent — but it presents a single clean interface:

```go
ktrl.RunOrDie(ctx)   // called by KonductorElection on the elected leader
```

**Startup sequence:**

1. Compute topological order from the dependency graph
2. For each CRD in dependency order:
   - Call `reconcilerFactory()` to build the reconciler
   - Wait for the informer cache to sync
   - Start `N` worker goroutines (N from `crd.Workers`)
   - Signal readiness — unblocks dependents
3. Start `retryMissingCRDs` once after the startup loop — handles CRDs
   not yet installed on the cluster
4. Call `SetReady()` after all enabled CRDs have started

**Worker dispatch:**

Each worker calls `safeReconcile` — a wrapper around the reconciler that:
- recovers from panics (one CRD cannot crash others)
- records metrics (duration, success/error count)
- updates CRD health state

```go
func (c *DependencyKontroller) processNextItem(ctx context.Context) bool {
    item, shutdown := c.queue.Get()
    if shutdown { return false }
    defer c.queue.Done(item)

    reconciler := c.registry.GetReconciler(item.GVK)
    if err := c.safeReconcile(ctx, reconciler, item); err != nil {
        c.queue.AddRateLimited(item)
        return true
    }
    c.queue.Forget(item)
    return true
}
```

**Missing CRD handling:**

If a CRD is declared in the Katalog but not yet installed on the cluster,
`retryMissingCRDs` runs once after the startup loop completes. It retries
in the background without blocking healthy CRDs. When the CRD appears,
`activateCRD` closes the readiness channel, which unblocks any dependents
waiting in `RunOrDie`.

**Shutdown:**

Stops accepting new items, drains in-flight reconciliations, shuts down
CRDs in reverse dependency order.

---

## KonductorElection

`pkg/konductor`

Kubernetes leader election. Runs as a post-start hook — only called after
all komponents are ready.

```go
ko := konductor.NewKonductorElection(
    startup.kube,
    startup.event,
    func(ctx context.Context) { startup.kontroller.RunOrDie(ctx) },
    func(konductor string) { printBanner(startup, konductor) },
    konductor.Options{
        Namespace:     kfg.Cluster().DefaultNamespace,
        LeaseDuration: kfg.Konductor().LeaseDuration,
        RenewDeadline: kfg.Konductor().RenewDeadline,
        RetryPeriod:   kfg.Konductor().RetryPeriod,
    })

startup.orkestra.AddPostStartHook(ko, func(ctx context.Context) {
    ko.Start(ctx)
})
```

**Behaviour:**

- All pods start informers — caches are warm on every replica
- Only the elected leader calls `kontroller.RunOrDie(ctx)`
- The banner prints only on the leader
- On leadership loss or context cancellation, the lease is released
  and a follower takes over with an already-warm cache
- Leadership transitions emit Kubernetes events

---

## Orkestra

`pkg/orkestra`

The lifecycle manager for all komponents. Not a komponent itself — it
owns and orchestrates them.

```go
o := ork.NewOrkestra(kfg.Cluster().DefaultResync, kfg.Ork().LogLevel)
o.Register(komponents)
o.Start(ctx)      // sequential — each must succeed before the next
o.Wait()          // blocks until context cancelled or fatal error
```

`AddPostStartHook` registers a function to run after `Start()` completes.
The KonductorElection is always registered as a post-start hook — it must
not run before all komponents (especially the informer factory and kontroller)
are fully started.

Shutdown on SIGTERM or context cancellation runs komponents in reverse
registration order. Every komponent has a `Stop()` method called in sequence.