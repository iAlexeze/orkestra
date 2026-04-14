# Kordinator — Developer Documentation

`kordinator` is the orchestration heart of Orkestra. These docs cover its internals progressively — start at the registry and work forward to understand how every part fits together.

## Documents

| File | Covers |
|---|---|
| [01-registry.md](01-registry.md) | `ResourceKatalog` — the per-GVK registry that holds every informer and reconciler factory |
| [02-health.md](02-health.md) | `CRDHealth` — atomic health counters, worker state tracking, dependency status, aggregate operator health |
| [03-startup.md](03-startup.md) | `DependencyKordinator` startup sequence — topology, dependency channels, non-blocking activation |
| [04-self-healing.md](04-self-healing.md) | The retry loop and its four phases — missing CRDs, runtime deletion, reappearance, deferred activation |
| [05-workers.md](05-workers.md) | The worker loop, `processItemForGVK`, queue drain, and shutdown semantics |
| [06-handlers.md](06-handlers.md) | The three runtime introspection HTTP handlers that power `ork status` and the Control Center |
