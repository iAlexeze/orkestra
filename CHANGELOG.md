# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Built‑in Kubernetes resource support** — Orkestra can now watch, queue, reconcile, validate, mutate, and health‑check *any* Kubernetes Kind (Pods, Deployments, Secrets, Services, Jobs, StatefulSets, etc.) without requiring CRDs or typed Go structs. Dynamic mode now treats built‑ins and CRDs uniformly with full metrics, health, and queueing
- **API enrichment engine** (`pkg/katalog/enrichment.go`) — kind‑only declarations (e.g., `kind: Pod`) are automatically enriched with group, version, plural, API path, and namespaced scope. Partial declarations are rejected with actionable errors. Enrichment results are surfaced in `ork validate`
- **Declarative validation framework** — field‑based validation rules (`exists`, `equals`, `prefix`, `suffix`, `contains`, `min`, `max`, custom operators). Supports deny/warn actions, multi‑violation reporting, and emits Prometheus metrics for pass/fail and rule‑level detail
- **Validation metrics** — `controller_validation_total` and `controller_validation_rejected_total` counters added with CRD, field, and rule labels for deep governance observability
- **Live status dashboard** (`ork status -w`) — kubectl‑style watch mode with full‑screen redraw, real‑time health transitions, queue depth, worker activity, reconcile counters, error rate, and uptime
- **Enrichment reporting in `ork validate`** — CLI now prints enriched CRDs with resolved group/version/plural and scope, clearly distinguishing built‑in vs custom resources
- **Per‑CRD Prometheus metrics** — reconcile histograms, queue depth, worker activity, resource count, error rate, consecutive failures, degrade thresholds, and last error tracking
- **Improved Katalog parsing pipeline** — enrichment applied early, followed by uniqueness validation, dependency validation, GVK/GVR assignment, defaults, reconcilers, runtime objects, hooks, and reconciler mode validation

### Changed

- **Katalog pipeline ordering** — enrichment now occurs immediately after loading enabled CRDs, ensuring all downstream validation, defaults, GVK assignment, and runtime setup operate on canonical metadata
- **GenericReconciler** — integrated validation engine, deny/warn behavior, and improved health/error reporting
- **CLI status output** — redesigned table layout, clearer health icons, improved uptime formatting, and consistent CRD filtering across single‑shot and watch modes
- **Health API** — expanded per‑CRD fields (degrade threshold, consecutive fails, last error, resource count, workers active) for richer observability
- **Metrics registry** — standardized CRD label format (`group/version, Kind=…`) across all metrics for consistency and easier Grafana dashboards

### Security
