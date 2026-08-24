# pkg/runtime/autoscaler

The autoscaler package evaluates time-based and metric-based conditions on a periodic tick and adjusts an operatorbox's worker count, queue depth, and resync interval at runtime — without stopping or restarting goroutines.

One `Autoscaler` instance runs per operatorbox that declares `autoscale:` in its `operatorBox:` block. It owns a single goroutine for the lifetime of the operatorbox and restores the declared baseline on clean shutdown.

## What lives here

| File | Role |
|------|------|
| `autoscaler.go` | `Autoscaler` — evaluation loop; condition dispatch; apply/restore logic |
| `autoscale_semaphore.go` | `ResizableSemaphore` — O(1) concurrency gate; resize without stopping goroutines |
| `autoscale_metrics.go` | `AutoMetrics` — atomic runtime counters (queue depth, P95 latency, error rate) |
| `autoscale_cross_metrics.go` | `CrossMetricsRegistry` — shared registry; enables `cross.<crd>.metrics.*` conditions |
| `autoscale_worker_info.go` | `WorkerInfo` + `BuildWorkerInfo` — serialisable worker snapshot for the CRD endpoint |

## Developer documentation

Full step-by-step documentation is in [docs/](docs/README.md).

| I want to… | Go to |
|-----------|-------|
| Understand the evaluation loop and override lifecycle | [01 — Overview](docs/01-overview.md) |
| Write `or:` / `when:` conditions | [02 — Conditions](docs/02-conditions.md) |
| Reference another CRD's metrics with `cross.<crd>.metrics.*` | [03 — Cross-Metrics](docs/03-cross-metrics.md) |
| Understand the WorkerInfo API response | [04 — Worker Info](docs/04-worker-info.md) |

For the `operatorBox.autoscale:` YAML declaration, see
[docs/runtime-manual/concepts/operator-autoscaler.md](../../docs/runtime-manual/concepts/operator-autoscaler.md).
