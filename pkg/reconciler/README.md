# pkg/reconciler

The reconciler package is the execution engine of every Orkestra operator. It takes a CR event from the Kubernetes informer and turns it into a sequence of API calls — creating, updating, or deleting child resources according to the Katalog's declarative configuration.

## What lives here

| File | Role |
|------|------|
| `generic.go` | `GenericReconciler[T]` — the single reconciler used by all CRDs |
| `generic_autoscale.go` | `AutoscaleTarget`, `AutoscalerRunner`, `ResyncLoopStarter`, `QueueInjector`, `QueueDepthReporter` implementations |
| `autoscaler.go` | `Autoscaler` — condition evaluation loop; applies and restores overrides |
| `autoscale_semaphore.go` | `ResizableSemaphore` — O(1) runtime concurrency resize without stopping goroutines |
| `autoscale_metrics.go` | `AutoMetrics` — atomic runtime metrics (queue depth, P95 latency, error rate) read by the autoscaler |
| `run_template_reconcile.go` | Declarative pipeline: normalize → resolver → onCreate → onReconcile → providers |
| `normalize.go` | `applyNormalize` — in-memory spec normalization before mutation/validation |
| `run_*.go` | Per-resource-type runners (deployments, services, secrets, cronjobs, …) |
| `run_foreach.go` | `forEach:` expansion — expands declarations into N copies before runners see them |
| `conditions.go` | `EvaluateWhen` wrappers and helpers |
| `children.go` | Reads owned child resources into the resolver's `.children.*` data |
| `run_namespace_guard.go` | `CheckNamespace` — allowed/restricted namespace enforcement |

## Developer documentation

Full step-by-step documentation is in [docs/](docs/README.md).

| I want to… | Go to |
|-----------|-------|
| Understand the full reconcile pipeline | [01 — Architecture](docs/01-architecture.md) |
| Understand what every `run_*.go` must implement | [02 — The run_*.go Contract](docs/02-run-pattern.md) |
| Understand the registry layer | [03 — Registry Layer](docs/03-registry-layer.md) |
| Debug condition / activeNames issues | [04 — Conditions](docs/04-conditions.md) |
| Understand `forEach:` expansion | [05 — forEach](docs/05-foreach.md) |
| Add `normalize:` to a Katalog | [06 — normalize](docs/06-normalize.md) |
| Add a new resource type end-to-end | [07 — Adding a Resource](docs/07-adding-a-resource.md) |

For the autoscaler design and the `operatorBox.autoscale:` declaration, see
[docs/runtime-manual/concepts/operator-autoscaler.md](../../docs/runtime-manual/concepts/operator-autoscaler.md).
