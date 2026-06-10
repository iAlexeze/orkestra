# Reconciler — Developer Documentation

This directory explains how the `pkg/reconciler` package works and how to add new resource types (e.g. `run_ingress.go`).

## Documents

| File | What it covers |
|------|----------------|
| [01-architecture.md](01-architecture.md) | The full reconcile pipeline from CR event to Kubernetes API call |
| [02-run-pattern.md](02-run-pattern.md) | The `run_*.go` function contract — what every resource runner must do |
| [03-registry-layer.md](03-registry-layer.md) | How `pkg/resources/<kind>` packages work |
| [04-conditions.md](04-conditions.md) | `when:` / `anyOf:` evaluation, operators, and the `activeNames` guard |
| [05-foreach.md](05-foreach.md) | How `forEach:` expansion works and what it requires from a runner |
| [06-normalize.md](06-normalize.md) | The `normalize:` phase — collapsing multiple input shapes before reconcile |
| [07-adding-a-resource.md](07-adding-a-resource.md) | Step-by-step guide — using `run_ingress.go` as the worked example |
| [08-rollback.md](08-rollback.md) | Declarative rollback — spec snapshotting, trigger evaluation, `onRollback` templates |
| [09-ptr-hooks.md](09-ptr-hooks.md) | Why `PTR` not `T`, the `ObjectHooks` adapter, and why the two-type-parameter form was not used |

Read them in order the first time. For a quick reference when writing a new runner, jump straight to [07-adding-a-resource.md](07-adding-a-resource.md) and use the checklist at the bottom.
