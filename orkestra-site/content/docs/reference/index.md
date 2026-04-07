---
title: "Index"
weight: 89
---

# Reference

This section provides detailed technical reference for the Orkestra runtime, its API, configuration schemas, and metrics.

---

## Runtime API

The Orkestra operator exposes a small, stable HTTP API for inspecting CRDs, their reconcilers, and their health status.

| Endpoint | Description |
|----------|-------------|
| `GET /katalog` | List all CRDs with basic health and configuration |
| `GET /katalog/{crd}` | Detailed config and runtime state for a single CRD |
| `GET /katalog/{crd}/health` | Health status (200 healthy / 503 degraded) |
| `GET /health` | Liveness probe |
| `GET /ready` | Readiness probe (200 when all reconcilers started) |
| `GET /metrics` | Prometheus metrics endpoint |

See the [Katalog API Reference](./katalog-api.md) for full details.

---

## Runtime Behavior

The Orkestra Runtime is the engine that executes Katalogs and Komposers.

| Phase | Behavior |
|-------|----------|
| **Startup** | Load Katalog, build dependency graph, start CRDs in topological order. Missing CRDs do not block startup. |
| **Reconciliation** | Fetch CR, convert to original version, run templates/hooks/constructors, apply registry operations, update status. |
| **Drift Correction** | Resources with `reconcile: true` are corrected on every reconcile. |
| **Shutdown** | Stop workers, stop informers, shut down CRDs in reverse dependency order. |

See the [Runtime Reference](./runtime.md) for full details.

---

## Metrics

Orkestra exposes Prometheus metrics for every CRD and the conversion webhook.

| Metric | Type | Description |
|--------|------|-------------|
| `controller_reconcile_total` | Counter | Reconcile count by result |
| `controller_reconcile_duration_seconds` | Histogram | Reconcile latency |
| `controller_queue_depth` | Gauge | Workqueue depth |
| `controller_workers_active` | Gauge | Active worker count |
| `controller_resource_count` | Gauge | Live CR count from cache |
| `controller_crd_activation_total` | Counter | CRD activation attempts |
| `controller_crd_activation_latency_seconds` | Histogram | Time from startup to activation |
| `orkestra_conversion_requests_total` | Counter | Conversion requests by direction |
| `orkestra_conversion_duration_seconds` | Histogram | Conversion latency |

All metrics include a `crd` label for per‑CRD granularity. See the [Metrics Reference](./metrics.md) for full details.

---

## Schemas

| Document | Description |
|----------|-------------|
| [Katalog Schema](./katalog-schema.md) | Complete field reference for Katalog YAML |
| [Komposer Schema](./komposer-schema.md) | Complete field reference for Komposer YAML |
| [Registry Schema](./registry-schema.md) | Artifact format for OCI‑published patterns |

---

## Related Documents

- [Katalog API Reference](./katalog-api.md) — detailed endpoint documentation
- [Metrics Reference](./metrics.md) — all metrics with labels and examples
- [Runtime Reference](./runtime.md) — startup, reconciliation, and shutdown flow
- [Katalog and Komposer Reference](./katalog-komposer-reference.md) — complete schema reference for both

---

**All reference documents assume familiarity with the core concepts. If you are new to Orkestra, start with the [Guides](../getting-started/index.md) or [Concepts](../runtime-manual/concepts/katalog.md).** 🎼