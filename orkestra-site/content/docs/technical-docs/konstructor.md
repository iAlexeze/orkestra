---
title: "Konstructor"
weight: 162
---

# konstructOrkestra — Startup and Wiring

`konstructOrkestra` is the assembly function that wires the entire Orkestra runtime. It is called once at process start, before any komponent is running. Its job is to build all the pieces, connect them together, and hand the result to `ork.Orkestra` which then manages the lifecycle.

Understanding this function is understanding Orkestra.

---

## The critical constraint

:::warning[No live Kubernetes connections in konstructOrkestra]
    Nothing that requires a running cluster runs in `konstructOrkestra`.
    Reconciler factories are closures — they capture the objects they need
    but are not called until `orkestra.Start()` has started the kubeclient
    and made the REST config available.

    The one exception is `kube.Start(ctx)` — the kubeclient is started
    early because the informer factory needs the REST config to check for
    missing CRDs. All other komponents start later, in the komponent list.
:::

This constraint is what makes the wiring safe: you cannot call `reconcile` before the informer has synced, because the reconciler factory is not invoked until the `DependencyKordinator` starts workers, which happens after `infFactory.Start()`, which happens after `kube.Start()`.

---

## Step by step

### Step 1 — Katalog

```go
kat := katalog.NewKatalog(m, kfg.Katalog().Paths...)
```

Calls `KomposeKatalogFromYaml`. Resolves all sources, enriches all CRD entries, validates the dependency graph, populates both registries (conversion and admission). Returns a fully-validated `*Katalog` where every `CRDEntry` in `kat.Enabled()` is ready for wiring.

If Katalog construction fails (bad YAML, missing CRD, cycle in dependsOn), the process exits here with a descriptive error.

### Step 2 — Scheme

```go
scheme, err := katalog.NewSchemeRegistry(kat)
```

Builds a `*runtime.Scheme` by calling `AddToScheme` for every typed-mode CRD in the Katalog. The scheme is passed to the kubeclient and used by the REST client to decode API server responses into typed Go structs.

Dynamic-mode CRDs do not appear in the scheme — they use `*unstructured.Unstructured` which bypasses scheme decoding entirely.

### Step 3 — Core komponents (created, not started)

```go
kube := kubeclient.NewKubeclient(...)
kube.Start(ctx)         // ← exception: started early
hs   := health.NewHealthServer(...)
ev   := event.NewEvent(kube)
defaultWq := queue.NewWorkqueue()
queueRegistry := queue.NewQueueRegistry()
```

Each is created but not started (except `kube`). Routes are registered on `hs` before its Start is called — the HTTP mux accepts registrations at any time, they just aren't served until `hs.Start()` opens the listener.

`hs` receives both `kat.ConversionRegistry()` and `kat.AdmissionRegistry()` at construction. The handlers look up rules from these registries on every request — they are populated at Katalog construction time and immutable at runtime.

### Step 4a — Client provider

```go
provider := kube.NewClientProvider()
for _, crd := range kat.Enabled() {
    if crd.IsDynamic() { continue }
    provider.Register(object, func(k *kubeclient.Kubeclient) (informer.GenericClient, error) {
        return k.NewClient(...)
    })
}
```

The client provider is a deferred factory. For each typed-mode CRD, it registers a constructor that, when called, creates a REST client for that GVK. The constructor is called the first time the informer factory needs that client — after kube is running.

Dynamic CRDs skip this step. They get a `ListerWatcher` from `kube.NewDynamicListerWatcher` in the next step.

### Step 4b — Shared informer factory

```go
infFactory := informer.SharedInformerFactory(
    provider, kube.RestConfig(), queueRegistry, defaultWq,
    scheme, kfg.Cluster().DefaultNamespace, kfg.Cluster().DefaultResync,
)
```

The informer factory is created but not started. `infFactory.Start()` is called by the `DependencyKordinator` after all other komponents are running. The factory routes watch events to queues via `handleEvent`.

### Step 4c — Kordinator registry and per-CRD wiring

This is the most important loop in `konstructOrkestra`. For each enabled CRD:

```go
for _, crd := range kat.Enabled() {
    crd := crd  // ← capture loop variable — required for closures

    // 1. Register a per-CRD queue with its max depth
    wq := queueRegistry.Register(gvk, crd.SetMaxQueueDepth(...))

    // 2. Create the informer — typed or dynamic
    if crd.IsDynamic() {
        lw := kube.NewDynamicListerWatcher(...)
        inf = infFactory.ForListerWatcher(lw, object, ctx, opts)
    } else {
        inf = infFactory.For(object, ctx, opts)
    }

    // 3. Build the reconciler factory — a closure, not called yet
    if crd.DefaultReconcile() {
        factory = func() domain.Reconciler {
            return reconciler.NewGenericReconciler(crdInfo, infCopy, ev, kube, anyHooks, objFactory)
        }
    } else {
        factory = func() domain.Reconciler {
            return crd.ReconcilerConfig.Constructor(kube, infCopy, ev)
        }
    }

    // 4. Register in the Kordinator registry
    kordRegistry.Register(gvk, crd, inf, factory)
}
```

**Why `crd := crd` is critical:** Go loop variables are shared across iterations. Without this capture, every closure would reference the same `crd` — the last one in the loop. This is one of the most common Go bugs in loop-closure code. Every CRD must capture its own copy.

**Why the factory is a closure, not an immediate call:** `NewGenericReconciler` needs a live event recorder (`ev`) and kubeclient. Neither is running yet — they start in the komponent list. The closure captures the references; the call happens later when the `DependencyKordinator` starts workers.

### Step 5a — CRD health map

```go
crdHealthMap := make(map[string]*kordinator.CRDHealth)
for _, crd := range kat.Enabled() {
    crdHealthMap[gvk] = kordinator.NewCRDHealth(crd.Name)
}
```

One `CRDHealth` instance per CRD, keyed by GVK string. Three components share pointers to these instances:
- Workers: call `RecordSuccess` / `RecordFailure` on every reconcile
- Route handlers: read health state on every HTTP request
- `DependencyKordinator`: uses health state for degradation logic

Shared pointers mean no copying — a worker incrementing a counter is immediately visible to the health handler reading it. `CRDHealth` uses `sync/atomic` for all counter operations.

### Step 5b — Route registration

```go
for _, crd := range kat.Enabled() {
    hs.Register("/katalog/"+crdName+"/health", kordinator.BuildCRDHealthHandler(...))
    hs.Register("/katalog/"+crdName,           kordinator.BuildCRDInfoHandler(...))
}
hs.Register("/katalog", kordinator.BuildKatalogHandler(...))
```

Routes are registered before `hs.Start()`. The handlers close over the `CRDHealth` pointers, the informer (for cache-based resource count), and the conversion/admission stats.

`BuildCRDInfoHandler` receives `hs.GetConversionStats()` and `hs.GetAdmissionStats()` — the same stats objects that the conversion and admission handlers write to. The info endpoint always shows the live stats.

### Step 6 — Dependency Kordinator

```go
kord := kordinator.NewDependencyKordinator(
    kube, infFactory, kordRegistry, ev, hs,
    queueRegistry, defaultWq, crdHealthMap,
    kfg.Cluster().DefaultWorkers, katalog.NewDependencyGraph(kat),
)
```

`NewDependencyGraph(kat)` computes the topological start order from `dependsOn` declarations. The `DependencyKordinator` uses this to start CRD workers in the correct order — a CRD that depends on `project` does not start workers until `project`'s informer has synced and its workers are running.

### Step 7 — Komponent list

```go
komponents := []domain.Komponent{
    hs,            // 1. health endpoints — up immediately, ready=false
    kube,          // 2. kubeclient (already started, Start() is idempotent)
    ev,            // 3. event recorder — needs kube
    queueRegistry, // 4. per-CRD queues — starts their internal loops
    defaultWq,     // 5. shared fallback queue
    infFactory,    // 6. informer factory — watches start, caches populate
    kord,          // 7. dependency Kordinator — starts workers in order
}
```

**Startup order is declared explicitly.** `Orkestra` calls `Start()` on each in slice order. Each `Start()` must not return until the komponent is ready for the next one to use it.

`hs` starts first so the `/health` endpoint is reachable immediately — the pod passes liveness checks even before the informer has synced. `/ready` returns 503 until `kord` calls `hs.SetReady()` after all CRD workers are started.

**Shutdown order is reverse.** `kord` shuts down first — workers stop, in-flight reconciles complete. Then informers stop. Then queues drain. Then the event recorder. Then kube. Then the health server (last, so it can report unhealthy during shutdown).

### Step 8 — Orkestra

```go
o := ork.NewOrkestra(kfg.Cluster().DefaultResync, kfg.Ork().LogLevel)
o.Register(komponents)
```

`Orkestra` owns the lifecycle. `o.Start(ctx)` starts komponents in order. `o.Shutdown()` stops them in reverse. Signal handling (SIGTERM, SIGINT) triggers `Shutdown`.

---

## Data flow through konstructOrkestra

```
konfig (env vars, flags)
  │
  ├──► Katalog ──────────────────────── CRDEntry[]
  │      │                                │
  │      ├── ConversionRegistry           │
  │      └── AdmissionRegistry            │
  │                                       │
  ├──► Scheme ◄────────────────── typed CRDs only
  │
  ├──► Kubeclient ──────────────────────► REST config, dynamic client, clientset
  │                                               │
  ├──► ClientProvider ◄───────────── typed CRDs ──┘
  │                                      │
  ├──► InformerFactory ◄─────────────────┘
  │      │                               │
  │      │  ForListerWatcher()    For()  │
  │      ▼                               ▼
  │   dynamic informer              typed informer
  │      │                               │
  │      └───────── events ──────────────┘
  │                    │
  │                    ▼
  ├──► QueueRegistry ──────────────── per-CRD workqueue
  │                    │
  │                    ▼
  ├──► KordinatorRegistry ─────────── (gvk → informer + factory)
  │
  ├──► CRDHealthMap ─────────────── (gvk → *CRDHealth)
  │         │                                │
  │         ▼                                ▼
  ├──► HealthServer routes          worker RecordSuccess/Failure
  │
  └──► DependencyKordinator ─────── starts workers in dependency order
              │
              └──► factory() → GenericReconciler or Constructor
```

---

## What each returned field is used for

```go
type orkestraKfg struct {
    konfig     *konfig.Konfig                 // read by CLI for status output
    katalog    *katalog.Katalog               // read by ork validate, ork status
    komp       *[]domain.Komponent            // owned by Orkestra, managed lifecycle
    event      *event.Event                   // used by reconcilers for CR events
    kube       *kubeclient.Kubeclient         // used by reconcilers for API calls
    kord *kordinator.DependencyKordinator // used by ork status for live state
    orkestra   *ork.Orkestra                  // started by cmd/run.go
}
```

`orkestraKfg` is returned to the CLI command. `cmd/run.go` calls `o.Start(ctx)` on it and then blocks until the context is cancelled (SIGTERM).
