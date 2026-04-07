---
title: "Observability"
weight: 123
---

# Observability

Orkestra provides first‑class observability for every operator built on the platform.  
Instead of requiring custom instrumentation, Orkestra exposes a consistent set of metrics, health endpoints, activation signals, and runtime introspection APIs — all automatically generated from your Katalog.

Observability in Orkestra focuses on answering the questions platform teams actually ask:

- Is this CRD healthy?
- Are reconcilers falling behind?
- How long does startup take?
- Which CRDs are the busiest?
- Why is reconciliation slow?
- How many resources exist for each CRD?

This page describes the observability model and how to use it effectively.

---

## Metrics

Orkestra exposes a focused set of high‑signal Prometheus metrics:

- reconcile performance  
- queue depth and worker utilization  
- CRD activation latency  
- informer resource counts  
- reconcile success/failure rates  

Metrics are exposed at:

```
/metrics
```

and are available immediately when the operator starts.

For the full list of metrics, see the **Metrics** concept page.

---

## Health Endpoints

Every operator exposes a consistent set of health endpoints:

```
GET /health
GET /ready
GET /metrics
GET /katalog
GET /katalog/{crd}
GET /katalog/{crd}/health
```

These endpoints provide:

- operator‑level health  
- CRD‑level health  
- dependency graph  
- reconcile statistics  
- activation state  

They are designed to integrate cleanly with Kubernetes probes, dashboards, and alerting systems.

---

## CRD Activation Visibility

Orkestra activates CRDs in dependency order.  
A CRD becomes active when:

- its informer cache has synced  
- its workers have started  
- its health endpoint returns `200`

Activation metrics include:

- `controller_crd_activation_total`
- `controller_crd_activation_latency_seconds`

These help diagnose slow startup, dependency bottlenecks, or misconfigured CRDs.

---

## Reconcile Visibility

For each CRD, Orkestra tracks:

- total reconcile attempts  
- reconcile duration histogram  
- error rate  
- queue depth  
- active workers  
- live resource count  

This provides a complete picture of reconcile performance and system load.

The `ork status` command surfaces these metrics in a human‑readable format.

---

## Drift Correction Visibility

When a template has:

```yaml
reconcile: true
```

Orkestra enforces declarative drift correction.  
This means:

- live resources are compared to templates  
- differences are patched  
- drift is corrected automatically  

Drift correction events are visible through:

- reconcile metrics  
- `ork describe`  
- Kubernetes events  
- the `/katalog/{crd}` endpoint  

---

## Debugging and Inspection

Orkestra provides several tools for debugging:

### CLI inspection commands

- `ork get` — list CRs  
- `ork describe` — show spec, status, and events  
- `ork events` — show Kubernetes events  
- `ork reconcile` — trigger reconciliation  

These commands remove the need to remember API groups or construct `kubectl` commands manually.

### Runtime introspection

The `/katalog` endpoint exposes:

- CRD configuration  
- dependency graph  
- reconcile statistics  
- health state  

This makes it easy to understand how the operator is behaving at runtime.

---

## Summary

Orkestra provides:

- high‑signal Prometheus metrics  
- consistent health endpoints  
- CRD activation visibility  
- reconcile performance insight  
- drift correction transparency  
- CLI tools for inspection and debugging  

Observability is built into the platform — no custom instrumentation required.

---

## Related Documentation

- [Metrics](../../reference/metrics.md)
- [Runtime](./runtime.md)
- [Typed CRDs](./typed-crds.md)
- [Katalog Schema](../../reference/katalog-schema.md)
- [Komposer Schema](../../reference/komposer-schema.md)
