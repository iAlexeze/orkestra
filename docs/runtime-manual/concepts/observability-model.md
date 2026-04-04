# The Observability Model

Orkestra provides a unified observability interface for every CRD it manages.
This document assembles the full picture — the live API, the Prometheus metrics,
the CLI tools, and how they relate to each other — so that an SRE can monitor
an Orkestra deployment with confidence.

---

## The design principle

Every CRD managed by Orkestra contributes to the same observability surface.
There is no per-CRD monitoring setup. There is one health server, one metrics
endpoint, one CLI interface. The data is the same regardless of whether Orkestra
manages two CRDs or twenty.

The observability stack has two tiers:

**Live operational state** — the `/katalog` endpoint. Reflects the current moment:
queue depth right now, workers active right now, active warnings right now. Resets
on restart. Consumed by `ork status`, `ork describe`, and dashboards.

**Historical metrics** — the `/metrics` endpoint. Prometheus counters and histograms.
Persist across restarts (in Prometheus). Consumed by dashboards, alerts, and
long-term trend analysis.

Neither substitutes for the other. Use the live API to answer "what is happening
right now." Use Prometheus to answer "what has been happening over time."

---

## The `/katalog` endpoint

`GET /katalog` returns the aggregate state of all managed CRDs:

```json
{
  "operator": "website-operator",
  "healthy": true,
  "uptime": "4h23m",
  "crds": [
    {
      "name": "website",
      "gvk": "demo.orkestra.io/v1alpha1, Kind=Website",
      "healthy": true,
      "workers": 3,
      "workersActive": 2,
      "resourceCount": 47,
      "queueDepth": 0,
      "reconcileTotal": 8312,
      "reconcileErrors": 3,
      "conversion": {
        "total": 62,
        "failures": 0,
        "avgLatencyMs": 0.5,
        "p95LatencyMs": 1.2
      },
      "admission": {
        "validationTotal": 1204,
        "validationDenied": 9,
        "mutationApplied": 387,
        "webhooksEnabled": true
      }
    }
  ]
}
```

`GET /katalog/{crd}` returns the full detail for one CRD. `GET /katalog/{crd}/health`
returns `200` or `503` for point health checks.

**`ork status` is the `/katalog` endpoint rendered for the terminal.** There is no
separate data source. The data is the same; the format differs.

---

## The Prometheus metrics

Six metric families. Each tells a distinct operational story.

### `controller_reconcile_total{crd, result}`

The most important metric. Total reconcile cycles, by outcome.

```
controller_reconcile_total{crd="demo.orkestra.io/v1alpha1, Kind=Website",result="success"} 8309
controller_reconcile_total{crd="demo.orkestra.io/v1alpha1, Kind=Website",result="error"}   3
```

**Alert:** Error rate above 5% sustained for more than 2 minutes.

```promql
rate(controller_reconcile_total{result="error"}[5m])
  / rate(controller_reconcile_total[5m]) > 0.05
```

### `controller_reconcile_duration_seconds{crd}`

Reconcile duration histogram. p95 and p99 tell you whether reconciles are taking
longer than expected — which usually indicates API server latency, slow external
calls in hooks, or queue backlog.

**Alert:** p99 above 5 seconds.

```promql
histogram_quantile(0.99,
  rate(controller_reconcile_duration_seconds_bucket[5m])
) > 5
```

### `controller_queue_depth{crd}`

Current items waiting in the workqueue. A persistently non-zero queue depth means
workers cannot keep up with the event rate. The solution is increasing `workers`
in the Katalog.

**Alert:** Queue depth above 100 for more than 5 minutes.

```promql
controller_queue_depth > 100
```

### `controller_workers_active{crd}`

Current active workers. Compare against the configured worker count to understand
utilisation. If `workersActive` consistently equals `workers`, you are at capacity.

### `controller_admission_validation_total{crd, result, source}`

Admission and reconcile-time validation outcomes. The `source` label is the
operationally critical dimension.

```
source="admission"   — call from /validate (kubectl apply time)
source="reconcile"   — call from the reconcile loop
```

**The diagnostic alert:**

```promql
# CRs are being denied at reconcile time but NOT at admission time
# Means ENABLE_ADMISSION_WEBHOOK is not set or the webhook is not intercepting
rate(controller_admission_validation_total{result="denied",source="reconcile"}[5m]) > 0
  AND
rate(controller_admission_validation_total{result="denied",source="admission"}[5m]) == 0
```

### `controller_admission_validation_violations_total{crd, field, rule, action, source}`

Per-field violation detail. Use this to understand which rules fire most and
whether they fire at both enforcement points.

```promql
# Which fields are most often denied at admission?
topk(5, sum by(field) (
  controller_admission_validation_violations_total{action="deny",source="admission"}
))
```

### `controller_admission_mutation_total{crd, result, source}`

Mutation outcomes. High `applied` rate at `source="admission"` means users
frequently omit fields that your defaults cover — a signal that documentation
or client tooling should improve.

---

## The CLI tools

### `ork status`

```
CRD                   Workers  Queue  Health   Reconciles  Errors
website               2/3      0      healthy  8,312       3
platform-namespace    2/2      0      healthy  412         0
database              4/4      12     healthy  1,891       0
```

The `Queue` column is the most actionable at a glance. A non-zero queue with
workers all active means throughput is bounded — more workers or fewer CRs.

### `ork describe <crd> <name>`

Full CR detail: spec, status, active warnings, recent events. The first diagnostic
step when a user reports their CR is not being reconciled correctly.

```bash
ork describe website my-site
```

### `ork events <crd>`

Recent Kubernetes events for all CRs of a type. Faster than `kubectl get events`
when you want to see what the operator has been doing across multiple CRs.

### `ork top`

Per-CRD resource consumption: reconcile rate, queue depth, worker utilisation.
The terminal equivalent of a Grafana dashboard row.

---

## Building a dashboard

The `/katalog` endpoint is a structured JSON API. Any tool that can scrape JSON
can build a dashboard from it. The recommended stack:

**Grafana + Prometheus:** Scrape `/metrics` and query with PromQL. The metric names
and label conventions are stable.

**Out-of-the-box dashboards:** Planned for a future release. The API is already
stable — the dashboard is a renderer, not a new data source.

**Minimum useful dashboard:**

Four panels cover the operational questions that matter:

1. **Reconcile error rate** — `rate(controller_reconcile_total{result="error"}[5m])` per CRD
2. **Queue depth** — `controller_queue_depth` per CRD
3. **Worker utilisation** — `controller_workers_active / <configured workers>` per CRD
4. **Validation denial rate** — `rate(controller_admission_validation_total{result="denied"}[5m])` by source

These four panels answer the four operational questions: is reconciliation healthy,
is there a backlog, is throughput bounded by workers, and is admission policy firing.

---

## The status API as observability

The CR's own `/status` subresource is part of the observability model. After every
successful reconcile, Orkestra writes:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
  observedGeneration: 4
  phase: Running
  readyReplicas: "3"
```

This makes `kubectl get websites` itself informative. External tools — ArgoCD
health checks, custom controllers, monitoring scripts — can watch for `Ready=True`
on CRs without knowing anything about Orkestra's internal health model.

The CR's status is the user-facing observability layer. Prometheus is the
platform-facing observability layer. The `/katalog` API is the operator-facing
observability layer. All three are available without any configuration.


- [Metrics](../../reference/metrics.md)
- [Runtime](./runtime.md)
- [Typed CRDs](./typed-crds.md)
- [Katalog Schema](../../reference/katalog-schema.md)
- [Komposer Schema](../../reference/komposer-schema.md)
