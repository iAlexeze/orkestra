# Reconciler — Developer Documentation

This directory explains how the `pkg/reconciler` package works and how to add new resource types (e.g. `run_ingress.go`).

## Documents

| File | What it covers |
|------|----------------|
| [01-architecture.md](01-architecture.md) | The full reconcile pipeline from CR event to Kubernetes API call |
| [02-run-pattern.md](02-run-pattern.md) | The `run_*.go` function contract — what every resource runner must do |
| [03-registry-layer.md](03-registry-layer.md) | How `pkg/orkestra-registry/<kind>` packages work |
| [04-conditions.md](04-conditions.md) | `when:` / `anyOf:` evaluation, operators, and the `activeNames` guard |
| [05-foreach.md](05-foreach.md) | How `forEach:` expansion works and what it requires from a runner |
| [06-adding-a-resource.md](06-adding-a-resource.md) | Step-by-step guide — using `run_ingress.go` as the worked example |

Read them in order the first time. For a quick reference when writing a new runner, jump straight to [06-adding-a-resource.md](06-adding-a-resource.md) and use the checklist at the bottom.
