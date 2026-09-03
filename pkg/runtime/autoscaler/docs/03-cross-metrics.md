# 03 — Cross-Metrics

## The problem

Sometimes the right signal for scaling one CRD comes from another CRD's runtime state. A service that writes to a managed database should slow down when the database operator's queue is overwhelmed. Cross-metrics make this possible without coupling the two operator binaries.

## Field syntax

```
cross.<crd>.metrics.<field>
```

- `<crd>` is the lowercase Katalog map key — the same name used in `cross:` declarations.
- `<field>` is any metric exposed by that operatorbox (`queueDepth`, `workersBusyPercent`, etc.).

Example:

```yaml
operatorBox:
  autoscale:
    conditions:
      when:
        - field: cross.managed-database.metrics.queueDepth
          greaterThan: "500"
    do:
      workers: 2
```

## Resolution path

```
cross.managed-database.metrics.queueDepth
  │
  ├── IsCrossMetricField()       → true (starts with "cross.", contains ".metrics.")
  │
  ├── ResolveCrossMetric(GlobalCrossMetricsRegistry, field, cond.Source)
  │       strips "cross."
  │       splits on ".metrics."  → crd="managed-database", metricName="queueDepth"
  │
  │   Path 1 — same binary:
  │       GlobalCrossMetricsRegistry.Get("managed-database")  → *AutoMetrics
  │       AutoMetrics.Get("metrics.queueDepth")                → "342"
  │
  │   Path 2 — different binary (Source.Endpoint set):
  │       GET http://database-operator:8080/katalog/managed-database
  │       parse response["metrics"]["queueDepth"]              → "342"
  │
  └── condition evaluator compares "342" > "500" → false
```

## Registration

`GlobalCrossMetricsRegistry` is the process-wide singleton. `DependencyKordinator` registers each CRD's `AutoMetrics` at startup, keyed by `entry.CRD.Name` (the lowercase Katalog map key):

```go
GlobalCrossMetricsRegistry.Register(entry.CRD.Name, reconciler.GetAutoMetrics())
```

This happens once, before any autoscaler ticks. Registration is safe for concurrent reads — `CrossMetricsRegistry` uses `sync.Map` internally.

## Constraints

- **Read-only.** A CRD can observe another's metrics but cannot write them. Writes go through `AutoMetrics.RecordReconcile` and `AutoMetrics.SetQueueDepth`, both called by the observed CRD's own worker loop.
- **Unknown CRD returns "".** `ResolveCrossMetric` returns `""` when the CRD is not found by either resolution path. The condition evaluator treats `""` as false (conservative — does not trigger override).

## Cross-binary: source.endpoint

Cross-metric conditions follow the same two-path resolution as `readCross` in the template engine:

| Path | When used | Hops |
|------|-----------|------|
| `GlobalCrossMetricsRegistry` | Same binary — CRD is registered at startup | Zero |
| `source.endpoint` HTTP call | Different binary — CRD not in local registry | One |

For cross-binary observation, add a `source:` block to the condition:

```yaml
operatorBox:
  autoscale:
    conditions:
      when:
        - field: cross.managed-database.metrics.queueDepth
          greaterThan: "500"
          source:
            endpoint: "http://database-operator:8080/katalog/managed-database"
            token: "$DATABASE_OPERATOR_TOKEN"   # optional
```

The endpoint is the remote operator's `/katalog/{crd}` URL — the same shape all Orkestra operators expose. The remote operator serves this from its own `AutoMetrics` (populated from its informer cache), so the chain is: one HTTP hop to the remote operator, then zero API-server calls on the remote side.

The response is parsed for its `"metrics"` object, which contains the same fields as `AutoMetrics.AsMap()`:

```json
{
  "metrics": {
    "queueDepth": 342,
    "workersBusyPercent": 73.5,
    "workersIdlePercent": 26.5,
    "reconcileDurationP95Ms": 47,
    "errorRatePercent": 0.2
  }
}
```

This `"metrics"` key is included in every `/katalog/{crd}` response when `autoscale:` is declared on that CRD (populated via `CRDHealth.GetAutoMetrics()`). No separate endpoint is needed.

Numeric values returned by `fetchCrossMetricHTTP` are formatted to strip trailing zeros (e.g. `73.5`, not `73.5000`). The condition evaluator parses them as floats, so the formatting does not affect comparison correctness.

---

**Next →** [04 — Worker Info](04-worker-info.md)
