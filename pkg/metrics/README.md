# pkg/metrics

`metrics` registers all Prometheus metrics for the Orkestra runtime. All metrics are registered at package init via `promauto` — importing the package is sufficient to make them available at `/metrics`.

Each file owns one domain of metrics. Recording functions are the only public API; the underlying `prometheus.Counter` / `Gauge` / `Histogram` variables are unexported.

## Metric domains

| File | Metric prefix | What it tracks |
|------|--------------|----------------|
| `autoscale.go` | `orkestra_autoscale_*` | Worker count overrides and baseline restores per CRD |
| `deletion_protection.go` | `orkestra_deletion_protection_*` | Webhook admission decisions (allowed/denied) |
| `external.go` | `orkestra_external_call_*` | External HTTP API calls made during reconciliation |
| `docker.go` | `orkestra_docker_*` | Docker build/push operations triggered by reconcilers |
| `git.go` | `orkestra_git_*` | Git operations (clone, pull, push) triggered by reconcilers |

## Recording pattern

Each file exposes named `Record*` or `Set*` functions — callers never reference the underlying metric vars:

```go
// After an external API call:
metrics.RecordExternalCall(crd, name, url, duration.Seconds(), errStr, statusCode)

// When the autoscaler applies an override:
metrics.RecordAutoscaleOverride(crd, workerCount)

// When the autoscaler restores the baseline:
metrics.RecordAutoscaleRestore(crd, baselineWorkers)
```

## Labels

Most metrics carry a `crd` label (the CRD name as registered in the Katalog) so that Grafana panels can filter per operator. The `external.go` and `docker.go` metrics additionally carry `name` (the resource name) and `url` / `image` for per-call granularity.

## Adding new metrics

Add a new file (e.g. `queue.go`) that:
1. Declares metric vars with `promauto.New*` at package level
2. Exports `Record*` / `Set*` / `Observe*` helpers
3. Does not export the underlying `prometheus.*` vars

The `promauto` registration happens at init automatically — no registration call needed.
