# Changelog — Operator Autoscaler

## [Unreleased] — feature/operator-autoscaler

### Added

- **`operatorBox.autoscale:` YAML field** — operators declare autoscale behaviour
  inside their operatorBox block. Parsed into `AutoscaleSpec`; nil when omitted.

- **Semaphore-gated concurrency in `GenericReconciler`** — `Reconcile` now
  acquires a `ResizableSemaphore` before processing and releases after. The
  autoscaler resizes the semaphore without stopping goroutines or interrupting
  in-flight reconciles.

- **Goroutine over-provisioning in `startCRDWorkers`** — when `do.workers` >
  `baseline.workers`, the kordinator starts `do.workers` goroutines at startup.
  Scale-up is instant (no new goroutine spawn at runtime); scale-down converges
  naturally as goroutines complete and find the semaphore full.

- **Live queue depth propagation** — worker loop calls `ReportQueueDepth` after
  every item so `metrics.queueDepth` conditions read real-time values.

- **Autoscaler goroutine lifecycle** — `RunAutoscaler` is launched in a goroutine
  tied to the CRD context. On shutdown it restores baseline before exiting.

- **`AutoscalerRunner` and `QueueDepthReporter` optional interfaces** — defined
  locally in kordinator to avoid the reconciler ↔ kordinator import cycle.
  Duck-typed at runtime via type assertions.

### Files changed

| File | Change |
|------|--------|
| `pkg/types/types.go` | Add `Autoscale *AutoscaleSpec` to `OperatorBoxConfig` |
| `pkg/reconciler/generic.go` | Add `workerSem`, `autoMetrics`, `autoscaler`; split `Reconcile` into gate + core |
| `pkg/reconciler/generic_autoscale.go` | New — `AutoscaleTarget`, `AutoscalerRunner`, `QueueDepthReporter` impl |
| `pkg/kordinator/dependency_kordinator.go` | Goroutine over-provisioning; autoscaler goroutine start |
| `pkg/kordinator/worker.go` | Queue depth reporting after each processed item |

### Initial autoscaler files (prior commit e1349e7)

| File | Description |
|------|-------------|
| `pkg/types/autoscale.go` | `AutoscaleSpec`, conditions, action, baseline, state types |
| `pkg/reconciler/autoscale_semaphore.go` | `ResizableSemaphore` — O(1) runtime resize |
| `pkg/reconciler/autoscale_metrics.go` | `AutoMetrics` — atomic counters, P95 rolling window, `Get(field)` |
| `pkg/reconciler/autoscaler.go` | `Autoscaler.Run`, `evaluate`, `conditionsMet`, cron/time/metric evaluators |
| `pkg/metrics/autoscale.go` | Prometheus counters for override/restore events |
| `docs/publications/operator-autoscaler.md` | Design rationale and feature overview |
| `docs/runtime-manual/concepts/operator-autoscaler.md` | User-facing introduction |
| `docs/runtime-manual/concepts/operator-autoscaler/__index.md` | User-facing main reference |
