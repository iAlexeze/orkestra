# **Orkestra Metrics**
**High‑signal, per‑CRD observability for declarative operators**

Orkestra exposes a focused set of Prometheus metrics designed specifically for declarative operators.  
Unlike generic Kubernetes metrics, Orkestra metrics are:

- **per‑CRD**
- **runtime‑aware**
- **reconcile‑centric**
- **activation‑aware**

These metrics give platform engineers deep visibility into:

- reconcile performance  
- queue pressure  
- CRD activation behavior  
- informer resource counts  
- worker utilization  

All metrics are automatically exposed at:

```
/metrics
```

via the built‑in health server.

!!! note
    No instrumentation is required.  
    Every operator built with Orkestra exposes these metrics automatically.

---

# **Metric Categories**

## **1. Reconciliation Metrics**

### `controller_reconcile_total{crd, result}`  
Counts all reconcile attempts.

- `crd`: CRD name  
- `result`: `success` or `failure`

Use this to:

- measure operator workload  
- detect failure spikes  
- validate reconcile throughput  

---

### `controller_reconcile_duration_seconds{crd}`  
Histogram of reconcile latency.

Use this to:

- detect slow reconcilers  
- identify API bottlenecks  
- tune worker counts  

!!! tip
    High latency + high queue depth usually indicates insufficient workers.

---

## **2. Resource & Queue Metrics**

### `controller_resource_count{crd}`  
Gauge of how many CRs exist for a CRD.

Use this to:

- detect runaway CR creation  
- validate informer correctness  
- monitor system scale  

---

### `controller_queue_depth{crd}`  
Gauge of the current work queue depth.

Use this to:

- detect backpressure  
- tune worker counts  
- identify reconcilers falling behind  

---

### `controller_workers_active{crd}`  
Gauge of active workers per CRD.

Use this to:

- understand worker utilization  
- detect worker starvation  
- tune concurrency  

!!! caution
    If `workers_active` is consistently equal to the configured worker count,  
    your operator is likely saturated.

---

## **3. CRD Activation Metrics**

### `controller_crd_activation_latency_seconds{crd}`  
Histogram of time from operator start → CRD active.

Use this to:

- diagnose slow startup  
- validate informer sync performance  
- tune readiness gates  

---

### `controller_crd_activation_total{crd, result}`  
Counts CRD activation attempts.

Use this to:

- detect CRDs that repeatedly fail to initialize  
- validate startup reliability  

!!! note
    Activation metrics are especially useful in multi‑CRD operators with dependency ordering.

---

# **Why These Metrics Matter**

Orkestra’s metrics are designed to answer the questions platform teams actually ask:

- *“Is this CRD healthy?”*  
- *“Why is this reconciler slow?”*  
- *“Are we falling behind on events?”*  
- *“Which CRDs are the busiest?”*  
- *“How long does startup take?”*  
- *“Is drift correction happening too often?”*  

Traditional operator frameworks require custom instrumentation.  
Orkestra gives you all of this **out of the box**, with zero Go code.

!!! tip
    These metrics make Orkestra operators not just easier to build — but easier to operate, debug, and scale.

---

# **Summary**

Orkestra metrics provide:

- high‑signal operational insight  
- per‑CRD granularity  
- reconcile‑centric observability  
- startup and activation diagnostics  
- queue and worker visibility  

They are a core part of Orkestra’s philosophy:  
**operators should be observable by default.**

---

## Related Documentation

- **Concept:** [Runtime](../runtime-manual/concepts/runtime.md)
- **Reference:** [Runtime Schema](../reference/runtime.md)
- **Next Use Case:** [Registry‑Powered Operators](./registry-schema.md)
