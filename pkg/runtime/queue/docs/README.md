# Queue — Developer Documentation

The queue package is the buffering layer between informer events and the reconciler. Each CRD gets its own `Workqueue` — a rate-limited, deduplicating queue backed by `client-go`'s `TypedRateLimitingQueue`. The kordinator enqueues CR keys when the informer fires; reconciler workers drain them. A `QueueRegistry` manages the full set of per-CRD queues and implements the `domain.Komponent` lifecycle so the supervisor can start and shut them down cleanly.

| # | Topic |
|---|-------|
| [01](01-workqueue.md) | Workqueue — structure, Enqueue, depth limit |
| [02](02-registry.md) | QueueRegistry — registration, lookup, lifecycle |
