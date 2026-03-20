# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- Runtime template interpretation for dynamic CRDs — `ork generate runtime`
  no longer required for zero-code operators
- `ork init` scaffolds a complete project and runs `go mod tidy` automatically

---

## [v1.0.0] - 2026-03-19

### Added
- **Zero-code operators** — manage CRD lifecycle from a Katalog YAML file alone.
  No Go reconcilers, no controller boilerplate.
- **OrkestraRegistry** — built-in resource implementations with owner references,
  idempotency, and drift correction: `Deployment`, `Service`, `Secret`,
  `ConfigMap`, `ServiceAccount`, `Job`, `CronJob`
- **Katalog sources** — compose CRD definitions from multiple origins:
  - Local file paths
  - Remote URLs
  - Environment variable references (`$VAR`)
  - Helm charts (local directory, remote repository, git-based)
  - Inline `spec.crds` with override semantics
- **Merger** — deduplicates across sources, catches duplicates with clear
  attribution, recursive source resolution
- **`reconcile: true`** shorthand — declare drift correction alongside onCreate
  without repeating the resource declaration under onReconcile
- **Dependency graph** — `dependsOn` with topological startup ordering and
  cycle detection. Dependents block until dependencies signal readiness.
  Missing CRDs retry in background without blocking healthy CRDs.
- **Health API** — every operator exposes built-in endpoints:
  - `GET /health` — liveness probe with uptime
  - `GET /ready` — readiness probe
  - `GET /katalog` — all CRDs with health, config, dependency graph
  - `GET /katalog/{crd}` — per-CRD config and reconcile stats
  - `GET /katalog/{crd}/health` — 200 healthy / 503 degraded
- **Prometheus metrics** — reconcile total, duration histogram, queue depth,
  worker count, resource count, CRD activation latency
- **`ork` CLI**:
  - `ork run` — start the operator runtime
  - `ork validate` — validate a Katalog (with clear per-field errors)
  - `ork template` — preview merged, validated Katalog (JSON, YAML, graph, verbose)
  - `ork generate runtime` — generate runtime wiring for typed CRDs
  - `ork init` — scaffold a new operator project
  - `ork version` — version, commit, and build date
- **Leader election** — HA deployments with multiple replicas
- **Dynamic informer** — unstructured CRDs use the dynamic client directly,
  bypassing scheme conversion errors
- **Examples**:
  - Website — hello world operator (Deployment + Service)
  - Platform Namespace — secrets, configmaps, serviceaccounts
  - Meta Katalog — composition from files, Helm, and inline overrides

### Notes
- Go mode (`BuildKatalogFromGo`) is not included in v1. Dynamic YAML-driven
  operators cover all demonstrated use cases. Go mode is planned for v2.
- `ork generate runtime` is required in v1 for dynamic CRDs. Runtime template
  interpretation (eliminating this step) is planned for v1.1.