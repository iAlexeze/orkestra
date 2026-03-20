# Orkestra Architecture

Orkestra is a declarative operator runtime. You declare CRDs in a Katalog.
Orkestra builds the operator around them — clients, informers, reconcilers,
workers, dependency ordering, health API, metrics, and leader election.

You write a Katalog. Orkestra manages everything else.

---

## Core design principles

**CRDs are data.** Every CRD is a declaration in a Katalog or Komposer YAML
file. The runtime reads the declaration and constructs all the machinery at
startup. Nothing is hardcoded.

**Dependency-aware lifecycle.** CRDs declare what they depend on. Orkestra
builds a DAG, validates it, and starts CRDs in topological order. Shutdown
runs in reverse. Dependents never start before their dependencies are ready.

**Two CRD styles, one model.** A CRD is either dynamic (default) or typed.
Dynamic CRDs use the dynamic client and `*unstructured.Unstructured` —
no compiled Go types needed, no scheme registration, no code generation.
Typed CRDs have a compiled Go type at `apiTypes.location` — the generator
registers the type and scheme at build time. Both styles run through the
same reconciler, the same health API, the same metrics.

**Three reconciler paths, one framework.** 
- Every CRD starts with `reconciler.default: true` — GenericReconciler handles the full lifecycle.

```yaml
reconciler:
  default: true             # Default behaviour
```

- Add `reconciler.hooks` for Go hooks when you need type-safe access or
external API calls. 

```yaml
reconciler:
  default: true
  hooks:
    location: github.com/my-org/orkestra/pkg/reconciler/hooks
    function: WebsiteHooks
    alias: websitehooks         # Optional
```

- Set `reconciler.default: false` with a `constructor` when you need full control of the reconcile loop. 

```yaml
reconciler:
  default: false
  constructor: 
    location: github.com/my-org/orkestra/pkg/reconciler
    function: NewWebsiteReconciler
    alias: websitereconciler    # Optional
```
In all three cases Orkestra owns the informer, workqueue, metrics, health, and leader election.

**Zero-code by default.** For dynamic CRDs with `onCreate`/`onReconcile`/
`onDelete` template declarations, no code generation is needed. The
GenericReconciler interprets templates directly at runtime. `ork run` is
the only command required.

---

## High-level flow

```
Katalog or Komposer YAML
        │
        ▼
Merger — resolves sources, deduplicates, validates
        │
        ▼
konstructOrkestra — wires all runtime components
  ├── NewSchemeRegistry   — registers typed CRD schemes
  ├── NewKubeclient       — REST client, dynamic client, clientset
  ├── ClientProvider      — CRD REST client factories (typed CRDs only)
  ├── SharedInformerFactory — per-CRD informers with per-CRD resync
  ├── KontrollerRegistry  — maps GVK → informer + reconciler factory
  ├── HealthServer        — routes registered before start
  └── DependencyKontroller — topological startup + worker dispatch
        │
        ▼
Orkestra — starts all komponents in registration order
        │
        ▼
KonductorElection — leader election as post-start hook
  └── kontroller.RunOrDie() runs only on the elected leader
```

See full architecture view [here](../docs/full-architecture-view.md).


---

## Startup sequence

`Konduct` is the entry point. It calls `konstructOrkestra` to wire
everything, starts all komponents via `orkestra.Start()`, then starts
leader election as a post-start hook.

```go
func Konduct(kfg *konfig.Konfig, m *merger.Merger, ctx context.Context) {
    startup := konstructOrkestra(kfg, m, ctx)

    go func() {
        startup.orkestra.Start(ctx)
    }()

    ko := konductor.NewKonductorElection(
        startup.kube,
        startup.event,
        func(ctx context.Context) { startup.kontroller.RunOrDie(ctx) },
        func(konductor string) { printBanner(startup, konductor) },
        konductor.Options{...},
    )

    startup.orkestra.AddPostStartHook(ko, func(ctx context.Context) {
        ko.Start(ctx)
    })

    startup.orkestra.Wait()
}
```

Komponent startup order (sequential, each must succeed before the next starts):

```
1. HealthServer        — routes registered before start, first up last down
2. Kubeclient          — REST config, dynamic client, clientset
3. EventRecorder       — depends on kubeclient
4. QueueRegistry       — per-CRD isolated queues
5. DefaultWorkqueue    — shared fallback queue
6. SharedInformerFactory — starts all informers, closes ready channel
7. DependencyKontroller — topological CRD startup + event dispatch
```

Shutdown runs in reverse order. The KonductorElection lease is released before
the kontroller shuts down.

---

## konstructOrkestra

`konstructOrkestra` [wires](../cmd/internal/konstruct_orkestra.go) the entire runtime without starting anything
(except kubeclient, which is started early as downstream services need
the REST config). All reconciler factories are closures — called only after
`orkestra.Start()` guarantees all komponents are live.

**Step by step:**

1. Load and validate the Katalog from the merger
2. Build the scheme registry — typed CRDs register `AddToScheme`
3. Create HealthServer, Kubeclient, EventRecorder, queues
4. Register REST client constructors via ClientProvider (typed CRDs only — dynamic CRDs skip this)
5. Build SharedInformerFactory
6. For each enabled CRD — create informer (typed or dynamic path), register reconciler factory in KontrollerRegistry
7. Build per-CRD health map
8. Register all health and Katalog API routes on the HealthServer
9. Create DependencyKontroller with the dependency graph
10. Return the `orkestraKfg` struct — Orkestra owns all komponents

---

## Dynamic vs typed CRD path

**Dynamic CRD** — `apiTypes.location` not set:

```
NewDynamicListerWatcher  →  ForListerWatcher  →  SharedIndexInformer
                                                 (unstructured.Unstructured)
```

The dynamic client bypasses scheme decoding entirely. The informer stores
`*unstructured.Unstructured` objects. The resolver has full access to all
spec fields at runtime via the object map.

**Typed CRD** — `apiTypes.location` set:

```
ClientProvider.Register  →  infFactory.For  →  SharedIndexInformer
                                               (compiled Go type)
```

The REST client decodes API server responses into compiled Go structs.
The reconciler has type-safe access via the concrete type. `ork generate runtime` command is required after this to register your types with Orkestra. Then `ork run`.

Both paths produce a `cache.SharedIndexInformer`. The KontrollerRegistry,
DependencyKontroller, and GenericReconciler are identical after this point.

---

## The Katalog

`pkg/katalog` is the runtime representation of the merged, validated
Katalog. It holds `[]CRDEntry` — the post-validation set of enabled CRDs
with all defaults applied, GVKs set, and modes resolved.

**Key methods:**

```go
kat.Enabled()              // []CRDEntry — only enabled CRDs
kat.All()                  // []CRDEntry — all CRDs including disabled
NewDependencyGraph(kat)    // builds the DAG
NewSchemeRegistry(kat)     // builds the runtime scheme
```

---

## Dependency graph

CRDs declare dependencies in the Katalog:

```yaml
- name: application
  dependsOn:
    - project
    - managednamespace
```

The DependencyKontroller computes topological order from the DAG.
CRDs start in dependency order — `application` only starts after both
`project` and `managednamespace` signal readiness.

Missing CRDs at startup — if a dependency is declared but its CRD is not
yet installed on the cluster — are handled by `retryMissingCRDs`. The
kontroller retries in the background without blocking healthy CRDs.
When the CRD appears, it starts automatically and signals dependents.

---

## GenericReconciler — three reconcile paths

Priority order — first match wins:

**1. Go hooks** — `r.hooks.OnReconcile != nil`

Registered via `HookFactory()` at startup. Full type-safe access to the CR.
Requires `ork generate runtime` to register the hook function.

**2. Declarative templates** — `r.rc.OnCreate != nil || r.rc.OnReconcile != nil`

`runTemplateReconcile()` interprets `onCreate`/`onReconcile`/`onDelete`
blocks directly at runtime. Calls OrkestraRegistry functions with resolved
template values. No code generation needed.

**3. No-op** — neither hooks nor templates declared

Finalizers, events, and metrics still handled. Useful for CRDs that only
need lifecycle tracking.

---

## KonductorElection — leader election

All pods run informers (warm caches). Only the elected leader runs workers.
On leadership loss or graceful shutdown the lease is released immediately,
a follower takes over, and its already-warm cache means zero startup delay.

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
```

The banner prints only on the elected leader. Followers are silent.

---

## Observability

Every Orkestra operator exposes these endpoints automatically:

| Endpoint | Purpose |
|---|---|
| `GET /health` | Liveness — 200 always when running |
| `GET /ready` | Readiness — 200 after all komponents ready, 503 before |
| `GET /metrics` | Prometheus metrics |
| `GET /katalog` | All CRDs — health, config, dependency graph |
| `GET /katalog/{crd}` | Single CRD — config, reconcile stats |
| `GET /katalog/{crd}/health` | 200 healthy / 503 degraded |

**Prometheus metrics — all per-CRD:**

| Metric | Type |
|---|---|
| `controller_resource_count` | Gauge — live CR count from informer cache |
| `controller_reconcile_total` | Counter — success/error per CRD |
| `controller_reconcile_duration_seconds` | Histogram — reconcile latency |
| `controller_queue_depth` | Gauge — current queue backlog |
| `controller_workers_active` | Gauge — active worker count |
|`controller_crd_activation_latency_seconds` | Histogram — CRD activation latency _(for missing CRDs)_ |
|`controller_crd_activation_total` | Counter — CRD activation count _(for missing CRDs)_ |


All metrics use the full GVK string as the `crd` label.

---

## Graceful shutdown

On SIGTERM or context cancellation, Orkestra shuts komponents down in
reverse startup order:

```
7. DependencyKontroller — stops workers, drains queues
6. SharedInformerFactory — stops informers
5. DefaultWorkqueue
4. QueueRegistry
3. EventRecorder
2. Kubeclient
1. HealthServer — last to stop so health probes stay live during shutdown
```

No partial reconciliations. No double processing. In-flight reconciles
complete before workers stop.