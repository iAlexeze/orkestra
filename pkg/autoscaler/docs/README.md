# Autoscaler — Developer Documentation

This directory explains how the `pkg/autoscaler` package works and how to write effective autoscale declarations.

## Documents

| File | What it covers |
|------|----------------|
| [01-overview.md](01-overview.md) | The evaluation loop, override lifecycle, and baseline restore |
| [02-conditions.md](02-conditions.md) | `anyOf:` (time/cron/day) and `when:` (metric) condition evaluation |
| [03-cross-metrics.md](03-cross-metrics.md) | Observing another CRD's runtime metrics via `cross.<crd>.metrics.*` |
| [04-worker-info.md](04-worker-info.md) | `WorkerInfo` — the CRD endpoint's worker snapshot and what each field means |

Read them in order the first time. For a quick reference when writing a Katalog autoscale block, jump straight to [02-conditions.md](02-conditions.md).
