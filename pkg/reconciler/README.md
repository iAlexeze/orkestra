# pkg/reconciler

The reconciler package is the execution engine of every Orkestra operator. It takes a CR event from the Kubernetes informer and turns it into a sequence of API calls — creating, updating, or deleting child resources according to the Katalog's declarative configuration.

## What lives here

| File | Role |
|------|------|
| `generic.go` | `GenericReconciler[PTR]` — the single reconciler used by all CRDs; PTR must be a pointer to the concrete CR type (e.g. `*Database`) |
| `ptr_hooks.go` | Design note: explains the PTR naming convention, the ObjectHooks adapter, and why the two-type-parameter form was not used |
| `generic_autoscale.go` | `AutoscaleTarget`, `AutoscalerRunner`, `ResyncLoopStarter`, `QueueInjector`, `QueueDepthReporter` implementations; `SetSpawnWorker`, `SetRollbackNotifiers`, `GetAutoMetrics`, `WorkerInfo` wiring helpers |
| `rollback.go` | Rollback subsystem — spec snapshotting, trigger evaluation, `onRollback` template execution, annotation lifecycle |
| `run_template_reconcile.go` | Declarative pipeline: normalize → resolver → onCreate → onReconcile → providers; calls runners in [`pkg/runners`](../runners/README.md) |
| `normalize.go` | `applyNormalize` — in-memory spec normalization before mutation/validation |
| `run_delete_ordered.go` | Sequential staged deletion with completion gates (`onDelete.ordered: true`) |
| `run_customresource.go` | Resolves, conditions-checks, and applies Custom Resource declarations; skips gracefully when target CRD is missing |
| `conditions.go` | `EvaluateWhen` wrappers and helpers |
| `run_namespace_guard.go` | `CheckNamespace` — allowed/restricted namespace enforcement |
| `gvr.go` | GVR aliases used by `run_delete_ordered.go`; authoritative GVR definitions live in `pkg/children` |

Per-resource-type runners (deployments, services, secrets, roles, …) live in [`pkg/runners`](../runners/README.md).

Child resource reading, forEach expansion, enrichment, and the built-in kind registry live in [`pkg/children`](../children/README.md).

## Developer documentation

Full step-by-step documentation is in [docs/](docs/README.md).

| I want to… | Go to |
|-----------|-------|
| Understand the full reconcile pipeline (including rollback gate) | [01 — Architecture](docs/01-architecture.md) |
| Understand the runner contract and canonical shape | [pkg/runners — 01 Runner Contract](../runners/docs/01-runner-contract.md) |
| Quick runner reference (reconciler-scoped) | [02 — Runner Function Contract](docs/02-run-pattern.md) |
| Understand the registry layer | [03 — Registry Layer](docs/03-registry-layer.md) |
| Debug condition / activeNames issues | [04 — Conditions](docs/04-conditions.md) |
| Understand `forEach:` expansion | [05 — forEach](docs/05-foreach.md) |
| Add `normalize:` to a Katalog | [06 — normalize](docs/06-normalize.md) |
| Add a new resource type end-to-end | [07 — Adding a Resource](docs/07-adding-a-resource.md) |

For the autoscaler design and the `operatorBox.autoscale:` declaration, see
[docs/runtime-manual/concepts/operator-autoscaler.md](../../docs/runtime-manual/concepts/operator-autoscaler.md).
