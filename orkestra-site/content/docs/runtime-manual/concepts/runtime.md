---
title: "Runtime"
weight: 144
---

# **Orkestra Runtime**

The Orkestra Runtime is the execution engine that powers every operator built with Orkestra.  
It handles CRD activation, dependency ordering, reconciliation, drift correction, metrics, health endpoints, and lifecycle management.

A Katalog defines *what* the operator should do.  
The Runtime defines *how* it runs.

---

## **Runtime Responsibilities**

The Runtime provides:

- CRD activation and dependency ordering  
- Informer setup and cache synchronization  
- Workqueue creation and worker management  
- Reconciliation orchestration  
- Drift correction for declarative templates  
- Hook and constructor execution  
- Status updates  
- Finalizer handling  
- Metrics and health endpoints  

All operators — dynamic or typed — share the same Runtime.

---

# **Startup Flow**

When you run:

```bash
ork run --katalog <path>
```

the Runtime performs the following steps:

1. **Load and merge all Katalogs / Komposers**  
2. **Validate** the merged result  
3. **Build the CRD dependency graph**  
4. **Activate CRDs in topological order**  
5. For each CRD:
   - Create informers  
   - Sync caches  
   - Create workqueue  
   - Start workers  
   - Expose CRD‑specific health endpoints  
   - Emit activation metrics  

CRDs with no dependencies start immediately.  
CRDs with dependencies wait until their parents are active.

---

## **CRD Activation**

A CRD is considered *active* when:

- its informer cache has synced  
- its workers have started  
- its health endpoint returns `200`  

Activation metrics:

- `controller_crd_activation_total{crd, result}`
- `controller_crd_activation_latency_seconds{crd}`

These help diagnose slow startup or dependency bottlenecks.

---

# **Reconciliation Flow**

For each event (create, update, delete, resync):

1. Fetch the CR from the informer cache  
2. Convert it to the **internal version** (highest declared version)  
3. Run the reconciler:
   - Declarative templates (`onCreate`, `onReconcile`, `onDelete`)
   - Go hooks (if declared)
   - Custom constructors (if declared)
4. Apply Registry operations  
5. Write status  
6. Requeue if needed  

Reconciliation is **idempotent** — running it twice produces the same result.

---

## **Drift Correction**

If a template has:

```yaml
reconcile: true
```

then on every reconcile:

- the live resource is compared to the template  
- differences are patched  
- manual changes are corrected  

This ensures declarative consistency.

If `reconcile: false`, the resource is created once and never updated.

---

# **Deletion Flow**

When a CR is deleted:

1. Finalizers run (global + CRD‑specific)  
2. `onDelete` templates execute  
3. Hooks may run cleanup logic  
4. Finalizers are removed  
5. The CR is deleted  

Finalizers ensure cleanup is safe and deterministic.

---

# **Health Endpoints**

Every operator exposes:

```
GET /health
GET /ready
GET /metrics
GET /katalog
GET /katalog/{crd}
GET /katalog/{crd}/health
```

These endpoints are identical for all operators — no custom wiring required.

---

# **Metrics**

The Runtime emits high‑signal, per‑CRD metrics:

- `controller_reconcile_total`
- `controller_reconcile_duration_seconds`
- `controller_queue_depth`
- `controller_workers_active`
- `controller_resource_count`
- `controller_crd_activation_total`
- `controller_crd_activation_latency_seconds`

See the **Metrics** concept page for full details.

---

# **Shutdown Flow**

When the operator stops:

1. Workers stop accepting new items  
2. Informers shut down  
3. CRDs shut down in **reverse dependency order**  
4. Shutdown metrics are emitted  

This ensures a clean, predictable shutdown.

---

# **Summary**

The Orkestra Runtime provides:

- deterministic CRD activation  
- dependency‑aware startup  
- consistent reconciliation  
- declarative drift correction  
- unified metrics and health endpoints  
- safe finalizer handling  
- predictable shutdown  

It is the foundation that makes Orkestra operators reliable, observable, and easy to operate.

---

## Related Documentation
- [Runtime Reference](../../reference/runtime.md)