# KI-001 — Gateway webhook stats show zeros with multiple replicas

**Status:** Open — pre-v1  
**Affects:** Gateway `/katalog` endpoint — deletion protection, conversion, and admission stats

---

## What you see

You run `curl /katalog` on the gateway and the counters are all zero — even though you can clearly see deletion protection is working (Helm refuses to uninstall, `kubectl delete` is denied) and conversion is working (objects are being created and reconciled across API versions).

---

## Why it happens

Orkestra's gateway runs with two replicas by default. When the Kubernetes API server needs to call a webhook — whether to block a deletion or convert a CR between versions — it picks whichever gateway pod is available. The chosen pod records the event in its own memory.

When you then query `/katalog` (via a port-forward or the control center), you land on one specific pod. If the webhook traffic went to the other pod, that pod's counters are zero and that's what you see.

This is a consequence of how Orkestra is structured: the runtime and gateway are separate processes. The gateway must run in the cluster with TLS because the API server requires it. Multiple replicas are the right production default. But per-pod in-memory stats don't survive across replicas.

The actual protection and conversion are unaffected — those work correctly regardless of which pod handles the request.

---

## Workaround

Prometheus metrics are the reliable source. The `metrics.RecordDeletionProtectionBlocked` and `metrics.RecordConversion` calls emit correct Prometheus counters in each pod. A Prometheus setup that scrapes all gateway pods and aggregates with `sum()` will show the true totals.

```promql
sum(increase(orkestra_deletion_protection_blocked_total[5m])) by (resource)
sum(orkestra_conversion_requests_total) by (kind, source_version, target_version)
```

---

## Resolution plan

The control center should discover all gateway pods behind the `gatewayEndpoint` and aggregate their `/katalog` stats before presenting them — rather than reading from a single endpoint. This does not require the control center to have direct cluster access; the gateway itself can expose a fleet-aggregated view.

Scheduled for resolution before v1.
