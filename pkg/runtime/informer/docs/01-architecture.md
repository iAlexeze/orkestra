# 01 — Informer Factory Architecture

## Purpose

The informer factory sits between the Kubernetes API server and the reconciler work queues. Its job is narrow: open one watch stream per CRD, populate an in-memory cache, and route every incoming event to the correct queue.

Reconcilers never call the API server to read CRs. They read from the informer cache. This is what makes cross-CRD observation zero-API-calls and status reads zero-cost.

## Lifecycle

```
konstructRuntime()
    │
    ├── SharedInformerFactory(...)     ← constructor, nothing started yet
    │       initializes: informers map, namespaceFilters map, ready channel
    │
    ├── For each CRD in kat.Enabled():
    │       Factory.For(obj, ctx, opts)          typed informer
    │     OR
    │       Factory.ForListerWatcher(lw, ...)    dynamic informer
    │
    │   getOrCreate(gvk, lw, obj, ctx, opts):
    │       inf = cache.NewSharedIndexInformer(lw, obj, resync, indexers)
    │       inf.AddEventHandler(handleEvent)     ← all three event types
    │       crdExists(gvk) via discovery client
    │       if missing: entry.Missing = true, added to f.missing
    │       if present: inf registered, not yet started
    │
    └── orkestra.Start(ctx)
            └── Factory.Start(ctx)
                    close(f.ready)               ← unblocks ListerWatchers
                    for each entry: go inf.Run(ctx.Done())
```

`f.ready` is a channel that blocks all `List`/`Watch` calls until `Start()` is called. This prevents the informer from opening connections before the rest of the runtime is ready.

## The Factory struct

```
Factory
├── clientProvider     — builds typed REST clients per object type (deferred)
├── restConfig         — used by the discovery client for CRD existence checks
├── queueRegistry      — GVK → per-CRD Workqueue mapping
├── defaultWq          — fallback queue when no per-CRD queue is registered
├── scheme             — runtime.Scheme for GVK resolution from Go types
├── defaultResync      — resync period when opts.Resync == 0
├── informers          — map[gvkStr]*InformerEntry (all registered informers)
├── missing            — subset of informers that had no CRD at startup
├── namespaceFilters   — map[gvkStr]*NamespaceFilter (Tier 2 filter, see 02)
├── started            — atomic.Bool, set after Start()
├── mu                 — sync.RWMutex protecting informers/missing/namespaceFilters
└── ready              — chan struct{}, closed by Start()
```

## InformerEntry

Each entry in `f.informers` holds:

```
InformerEntry
├── Informer    — the cache.SharedIndexInformer (always non-nil after getOrCreate)
├── Name        — human-readable label (CRD kind)
├── Resync      — per-informer resync period
├── Missing     — true when CRD did not exist at registration time
├── GVK         — *schema.GroupVersionKind
├── Ctx/Cancel  — stored context (used for post-startup retry)
└── WasNeverStarted — set when Start() skips a missing entry
```

Missing entries are kept in `f.missing` and retried by `konstructRuntime` via `SetMissingOnStartup` / `RemoveMissing`.

## handleEvent — the hot path

Every watch event flows through `handleEvent`. The function must be fast — it runs in the informer's goroutine:

```
handleEvent(obj)
    │
    ├── <-f.ready                          block until factory started
    │
    ├── gvkFromObject(obj)
    │       scheme.ObjectKinds(runtimeObj)
    │       returns *schema.GroupVersionKind
    │       drops event and returns on error (unknown type)
    │
    ├── extractNamespace(obj)              Tier 2: see 02-namespace-filter.md
    │       unwrap DeletedFinalStateUnknown tombstone if needed
    │       meta.Accessor(rObj).GetNamespace()
    │
    ├── namespaceAllowed(gvkStr, namespace)
    │       mu.RLock → look up filter → mu.RUnlock
    │       filter.Allows(namespace)
    │       if not allowed: log debug + return  ← event dropped, queue untouched
    │
    └── queueRegistry.For(gvkStr)
            found:     wq.Enqueue(obj, gvkStr)
            not found: defaultWq.Enqueue(obj, gvkStr) + warn log
```

The GVK is resolved from the scheme rather than from `obj.GetObjectKind()` because cached objects have their `TypeMeta` stripped by the reflector. The scheme lookup is cached internally — it is not a map allocation on every event.

## WaitForCacheSync

After `Start()`, callers block on `Factory.WaitForCacheSync(ctx)` until every non-missing informer reports `HasSynced() == true`. This means the initial `List` has completed and the in-memory cache is populated.

Missing informers are excluded from the sync wait — they have no `List` to complete.

## Thread safety

| Operation | Lock used |
|-----------|-----------|
| `RegisterNamespaceFilter` (write) | `mu.Lock` |
| `namespaceAllowed` (read, hot path) | `mu.RLock` |
| `getOrCreate` | `mu.Lock` (held by `For`/`ForListerWatcher`) |
| `WaitForCacheSync` | `mu.RLock` |
| `Registered`, `Missing` | `mu.RLock` |
| `SetMissingOnStartup`, `RemoveMissing` | `mu.Lock` |

`handleEvent` takes only a read lock (for the namespace filter map lookup). Write locks are held only during registration, which happens before `Start()`.

---

**Next →** [02 — Namespace Filter](02-namespace-filter.md)
