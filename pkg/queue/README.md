# pkg/queue

`queue` is the buffering layer between informer events and the reconciler. Every
CRD gets its own `Workqueue`. The kordinator reads from these queues and calls
the reconciler for each item. The queue package has no knowledge of what is
being reconciled — it holds keys and routes them.

## What lives here

| File | Role |
|------|------|
| `queue.go` | `Workqueue` — per-CRD rate-limiting queue with atomic depth limit |
| `registry.go` | `QueueRegistry` — maps GVK strings to their `Workqueue`; started as an Orkestra component |

## Core concepts

**One queue per CRD.** Isolation means a CRD with large CRs or slow external
calls cannot delay reconciles for other CRDs. Workers for a CRD read only from
that CRD's queue.

**Rate-limiting.** The underlying `workqueue.TypedRateLimitingInterface` applies
exponential backoff on re-queued items after reconcile failures. The first
failure retries immediately; subsequent failures back off up to a maximum.
`Forget(item)` resets the backoff counter on success.

**Depth limit.** `Workqueue.Enqueue` enforces an atomic `int32` depth limit.
When the limit is non-zero and the queue is at or beyond it, new items are
dropped with a warning log. This is the mechanism by which the autoscaler's
`do.queueDepth` override takes effect. 0 means unlimited (the default).

**Deduplication.** The workqueue deduplicates by key — if the same key is
enqueued multiple times while a worker holds it, only one copy queues behind.
This makes concurrent resync sources (informer + autoscaler resync goroutine)
safe.

## Developer documentation

| I want to understand… | Go to |
|---|---|
| The Workqueue type and how Enqueue enforces depth limits | [01 — Workqueue](docs/01-workqueue.md) |
| The QueueRegistry and how queues are registered and looked up | [02 — Registry](docs/02-registry.md) |

## Key types at a glance

| Type | File | Role |
|---|---|---|
| `Workqueue` | `queue.go` | Per-CRD queue: Enqueue, Depth, SetMaxDepth |
| `QueueRegistry` | `registry.go` | GVK → Workqueue map; Orkestra component lifecycle |
| `QueueItem` | `queue.go` | `{Key, GVK}` — the unit of work passed between informer and worker |
