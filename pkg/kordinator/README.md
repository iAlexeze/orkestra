# Dependency Kordinator

The `DependencyKordinator` is the orchestration engine that manages the lifecycle of Custom Resource Definitions (CRDs) in an Orkestra operator. It ensures CRDs start in the correct topological order, shut down gracefully in reverse order, and self‑heal when CRDs are added or removed from the cluster at runtime.

## Features

- **Topological startup** – CRDs start only after their declared dependencies are satisfied.
- **Graceful shutdown** – CRDs are stopped in reverse dependency order, allowing dependents to drain first.
- **Self‑healing** – Monitors the cluster continuously and activates CRDs that appear after startup.
- **Runtime deletion handling** – Deactivates workers when a CRD is deleted, preventing error logs.
- **Non‑blocking startup** – The main loop does not stall on dependencies that take time (e.g., waiting for `healthy`). Deferred CRDs are activated later by a background retry loop.
- **Health‑aware dependencies** – Supports both `started` and `healthy` dependency conditions.

## How It Works

### Dependency Graph

At startup, the `Katalog` is parsed into a directed acyclic graph (DAG) where:

- **Nodes** represent CRDs.
- **Edges** represent `dependsOn` relationships.

Kahn’s algorithm produces a deterministic topological order. CRDs that share the same dependencies are sorted alphabetically for predictability.

### Startup Sequence

1. The main `Kordinate` loop iterates through the topological order.
2. For each CRD, it checks whether all dependencies are currently satisfied using a **non‑blocking** channel read.
3. If dependencies are ready and the CRD exists in the cluster, workers are started and the `started` channel is closed.
4. If dependencies are **not** ready, or the CRD is missing, the loop skips the CRD and continues.
5. A background `retryMissingCRDs` goroutine periodically re‑evaluates skipped CRDs and activates them once dependencies become satisfied.

### Retry Loop

The retry loop runs indefinitely and handles three categories:

1. **CRDs missing at startup** – Monitors the `missing` map and activates CRDs when they appear in the cluster.
2. **Runtime deletions** – Detects when an active CRD is removed and deactivates its workers.
3. **Deferred activation** – Scans CRDs that were skipped because dependencies were not ready. When conditions become satisfied, it activates them.

### Dependency Conditions

| Condition | Channel Closed When                     | Typical Use Case                               |
|-----------|-----------------------------------------|------------------------------------------------|
| `started` | Workers have been launched               | “The CRD’s controller is running.”             |
| `healthy` | First successful reconciliation completes | “The CRD has processed at least one resource.” |

- The `started` channel is **never closed during deactivation** so that dependents can continue running (in a degraded state) rather than blocking.
- `healthy` channels are closed by a separate health‑checker goroutine.

### Shutdown

When the leader election lease is lost, the `Kordinate` context is cancelled:

1. CRDs are shut down in **reverse topological order**.
2. Each CRD’s worker pool is drained (with a configurable timeout).
3. Informers and queues are stopped.

## Key Methods

| Method                 | Description                                                                                  |
|------------------------|----------------------------------------------------------------------------------------------|
| `Kordinate(ctx)`       | Main entry point. Starts CRDs, blocks until leadership ends, then shuts down.                 |
| `dependenciesReady()`  | Non‑blocking check that all `started` / `healthy` channels for a CRD are closed.              |
| `activateCRD()`        | Starts informer, launches workers, closes `startedCh`, updates health.                         |
| `deactivateCRD()`      | Stops workers and marks CRD as degraded without closing dependency channels.                  |
| `retryMissingCRDs()`   | Background loop that handles missing, deleted, and deferred CRDs.                              |
| `startCRDWorkers()`    | Launches a worker pool for a single CRD.                                                      |
| `stopCRDWorkers()`     | Cancels worker context and waits for the pool to drain.                                       |

## Health Integration

The `DependencyKordinator` maintains:

- Per‑CRD health via `crdHealthMap` (started state, worker counts, queue depth, error rates).
- Aggregate Katalog health via `orkHealth` (ready / degraded).
- Channels for `anyOnline` and `allOnline` that signal overall operator status.

## Configuration

- `defaultWorkers` – Number of workers per CRD if not specified in the Katalog.
- `drainTimeout` – Maximum time to wait for workers to finish during shutdown.
- `PostStartRetryInterval` – Frequency of the retry loop (with exponential backoff for missing CRDs).

## Usage Example

```go
kord := NewDependencyKordinator(
    kubeClient,
    informerFactory,
    resourceKatalog,
    eventBus,
    healthService,
    queueRegistry,
    defaultQueue,
    crdHealthMap,
    orkHealth,
    5,                     // defaultWorkers
    depGraph,
    30*time.Second,        // drainTimeout
)

// Blocks until context is cancelled (e.g., leader election lost)
kord.Kordinate(ctx)
```

## Logs

The kordinator emits structured logs at startup, during activation, and on shutdown:

```
INFO  starting component=orkestra dependency kordinator
INFO  startup order order="website → namespace-guard → service-sentinel → pod-watcher"
INFO  starting CRD crd=website
INFO  workers started crd=website gvk="demo.orkestra.io/v1alpha1, Kind=Website" workers=3
INFO  dependencies not ready — deferring activation crd=pod-watcher
INFO  started crds_online=3
...
INFO  dependencies now satisfied, activating crd=pod-watcher
INFO  CRD pod-watcher activated
```

This makes it easy to trace the startup sequence and understand why certain CRDs are deferred.
