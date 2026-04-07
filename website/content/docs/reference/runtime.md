---
title: "Runtime"
weight: 50
description: "The Orkestra Runtime is the engine that executes Katalogs and Komposers."
---

The Orkestra Runtime is the engine that executes Katalogs and Komposers.  
It handles CRD activation, dependency ordering, reconciliation, drift correction, metrics, and health endpoints.

This page explains how the runtime behaves at startup, during reconcile, and during shutdown.

---

# Startup Flow

1. Load the Katalog or Komposer  
2. Build the dependency graph  
3. Start CRDs in topological order  
4. For each CRD:
   - Create informers  
   - Create workqueue  
   - Start workers  
   - Expose health endpoints  
   - Emit activation metrics  

!!! note
    Missing CRDs do not block startup — Orkestra activates them when they appear.

---

# Reconciliation Flow

For each CR event:

1. Fetch the CR  
2. Convert to the internal version (highest available)  
3. Run the reconciler:
   - Templates  
   - Hooks  
   - Custom constructors  
4. Apply Registry operations  
5. Write status  
6. Requeue if needed  

---

# Drift Correction

If a template has:

```yaml
reconcile: true
```

Then on every reconcile:

- The resource is compared to the template  
- Differences are patched  
- Manual changes are corrected  

!!! caution
    Drift correction is intentional — Orkestra enforces the declared state.

---

# Shutdown Flow

1. Stop workers  
2. Stop informers  
3. Shutdown CRDs in **reverse dependency order**  
4. Emit shutdown metrics  

---

# Health Endpoints

Every operator exposes:

```
GET /health
GET /ready
GET /metrics
GET /katalog
GET /katalog/{crd}
GET /katalog/{crd}/health
```

!!! tip
    These endpoints are identical for every operator — no custom wiring required.

---

# Metrics

The runtime emits:

- `controller_reconcile_total`  
- `controller_reconcile_duration_seconds`  
- `controller_queue_depth`  
- `controller_workers_active`  
- `controller_resource_count`  
- `controller_crd_activation_total`  
- `controller_crd_activation_latency_seconds`  
