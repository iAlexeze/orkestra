# queue

The `queue:` block controls how reconcile events accumulate and when the operatorBox is considered unhealthy. Every CRD gets its own isolated queue by default — queue pressure from one CRD cannot affect another.

```yaml
crds:
  myapp:
    operatorBox:
      reconciler:
        workers: 4
        resync: 30s
        queue:
          maxDepth: 500
          failureThreshold: 10
```

---

## Fields

| Field | Type | Default | Description |
|---|---|---|---|
| `maxDepth` | int | `100` (`QUEUE_DEPTH` env) | Maximum items the queue holds. When the limit is reached, new reconcile events are **dropped** — not queued, not retried. The resync period will re-enqueue the CR on the next tick. |
| `failureThreshold` | int | `5` (`FAILURE_THRESHOLD` env) | Consecutive reconcile failures before the operatorBox transitions to degraded. Resets to zero on the next successful reconcile. |
| `shared` | bool | `false` | Use the shared global workqueue instead of an isolated per-CRD queue. Rarely needed. |

Defaults are controlled by `QUEUE_DEPTH` and `FAILURE_THRESHOLD` environment variables in the runtime deployment — set them in `values.yaml` under `runtime.config`.

---

## `maxDepth` — understanding the drop behaviour

When the queue reaches `maxDepth`, Orkestra logs a warning and drops the incoming event:

```json
{"level":"warn","key":"default/my-app","gvk":"...","limit":100,"depth":100,"message":"enqueue: queue depth limit reached — item dropped"}
```

This is **intentional back-pressure**, not data loss. The dropped event is a reconcile trigger — the CR itself is unchanged in etcd. The next resync tick re-enqueues it automatically.

The right value for `maxDepth` depends on how many CRs the operator manages and how bursty your event pattern is. A good starting point:

- **Steady workload** (few CRs, low event rate): leave at the default `100`
- **Bursty workload** (many CRs, event spikes): increase to 500–2000
- **Autoscaled operator**: set a baseline here and let the autoscaler override it under load

### Interaction with autoscaling

When the autoscaler is active, `maxDepth` becomes the baseline. The autoscaler can raise the limit at runtime when conditions trigger (e.g., queue depth exceeds 80% of the limit) and restore it when conditions clear:

```yaml
operatorBox:
  reconciler:
    queue:
      maxDepth: 100       # baseline — what you start with
  autoscale:
    conditions:
      when:
        - field: metrics.queueDepth
          greaterThan: "80"
    do:
      workers: 8
      queueDepth: 500   # raised when conditions are met
```

**Try it:**
```bash
ork init --pack advanced
cd 12-autoscale/01-without-autoscaler   # see what happens when maxDepth is hit
ork run

cd 12-autoscale/02-based-on-own-metrics  # autoscaler raises the limit dynamically
ork run
```

---

## `failureThreshold` — when health degrades

Each reconcile failure increments a consecutive failure counter. When it reaches `failureThreshold`, the operatorBox transitions to **degraded**:

- The Control Center marks it unhealthy with the failure count and last error.
- Other CRDs with `dependsOn: <this-crd>: healthy` stop processing new CRs.
- If `rollback:` is configured, the rollback templates execute.
- The counter resets to zero on the next successful reconcile.

The default of `5` is appropriate for most operators. Increase it for operators that call external services that can be transiently unavailable — a lower threshold would cause false degraded states during brief outages:

```yaml
operatorBox:
  reconciler:
    queue:
      failureThreshold: 20   # external service can be down for a few minutes
```

Decrease it for operators managing critical infrastructure where you want immediate health signalling:

```yaml
operatorBox:
  reconciler:
    queue:
      failureThreshold: 2   # degrade fast — this CRD must be healthy
```

---

## `shared` — the shared queue

By default each CRD has its own isolated workqueue. Setting `shared: true` puts this CRD into the global shared queue. This is only useful in rare situations — for example, when a built-in Kubernetes resource (Pod, ConfigMap) is being reconciled and you want it to share the global queue rather than consuming a separate goroutine pool.

For custom CRDs, always leave `shared: false`.

---

## Global defaults

The defaults for all CRDs in the runtime are set via environment variables, configured in `values.yaml`:

```yaml
# charts/orkestra/values.yaml
runtime:
  config:
    maxDepth: 500          # QUEUE_DEPTH env — default maxDepth for all CRDs
    failureThreshold: 10   # FAILURE_THRESHOLD env — default failureThreshold for all CRDs
```

A per-CRD `queue:` declaration overrides the global default for that CRD only.
