# Trust Model & Failure Guarantees

Orkestra is a runtime that manages live infrastructure.  
Trust comes from **predictability**, **isolation**, and **transparent behavior** — not from promises.

This document explains why Orkestra is safe to run in production and how it behaves when things go wrong.

---

## 1. Runtime Isolation Model

Orkestra is designed so that failures are **contained**, **recoverable**, and **observable**.

### 1.1 Per‑CRD Isolation

Each CRD has:

- its own workqueue  
- its own worker pool  
- its own reconcile loop  
- its own health state  
- its own metrics  

A failure in one CRD **cannot** affect others.

### 1.2 Per‑CR Isolation

Each reconcile is wrapped in `safeReconcile`, which:

- recovers from panics  
- records metrics  
- updates CRD health  
- prevents worker crashes  

Actual implementation enforces this:

```go
func (c *DependencyKontroller) processNextItem(ctx context.Context) bool {
    item, shutdown := c.queue.Get()
    if shutdown { return false }
    defer c.queue.Done(item)

    reconciler := c.registry.GetReconciler(item.GVK)
    if err := c.safeReconcile(ctx, reconciler, item); err != nil {
        c.queue.AddRateLimited(item)
        return true
    }
    c.queue.Forget(item)
    return true
}
```

**Guarantee:**  
A panic in one CR does not crash the worker.  
A panic in one CRD does not affect other CRDs.

---

## 2. Idempotent, Safe Operations

Every registry implementation (`Create`, `Update`, `Delete`) is designed to be safely re-run:

- `Create` never errors on “already exists”
- `Delete` never errors on “not found”
- `Update` performs drift correction on explicit fields only

**Guarantee:**  
Reconciles can be retried indefinitely without corrupting state.

---

## 3. Transparent Observability

Orkestra exposes:

- `/health` and `/ready`  
- `/metrics` (queue depth, reconcile duration, worker count, errors)  
- `/katalog` (CRD health, config, dependency graph)  

You can always see what Orkestra is doing.

**Guarantee:**  
Failures are visible, not silent.

---

## 4. Declarative Behavior

All operator behavior is declared in:

- **Katalogs** (per‑CRD behavior)
- **Komposers** (multi‑source composition)
- **Registry entries** (resource implementations)

There is no hidden code path.

**Guarantee:**  
What you declare is what runs.

---

## 5. Failure Modes & Guarantees

Orkestra is designed with explicit, predictable failure modes.

---

### 5.1 If Orkestra is down

- Existing resources remain as-is  
- No new reconciles occur  
- Kubernetes continues normally  
- Orkestra resumes from current state on restart  

**Guarantee:**  
Orkestra never blocks the API server or normal cluster operation.

---

### 5.2 If a reconcile fails

- Error recorded in metrics  
- Error recorded in CRD health  
- Item requeued with rate-limited backoff  
- No partial state assumed  

**Guarantee:**  
A failed reconcile does not corrupt state and can be retried safely.

---

### 5.3 If a reconcile panics

- `safeReconcile` catches the panic  
- Worker continues  
- CR is retried  
- Other CRDs are unaffected  

**Guarantee:**  
One CR cannot crash the runtime. One CRD cannot crash others.

---

### 5.4 If a Katalog is invalid

- `ork validate` fails  
- Runtime refuses to start  

**Guarantee:**  
Invalid operator definitions never reach the cluster.

---

### 5.5 If a CR is malformed

- Templates may resolve to empty/invalid values  
- Registry operations fail with clear errors  
- CR status reflects the failure  
- Reconcile is retried  

**Guarantee:**  
Bad CRs fail loudly and observably.

---

### 5.6 If dependencies are missing

- Dependent CRDs wait  
- Healthy CRDs continue  
- No global stall  

**Guarantee:**  
One broken CRD cannot block the entire system.

---

### 5.7 If registry operations fail

- All operations are idempotent  
- Failures are surfaced  
- Retries are safe  

**Guarantee:**  
Registry failures are isolated and recoverable.

---

### 5.8 If the queue backs up

- Each CRD has its own queue  
- Each CRD has its own workers  
- No cross‑CRD starvation  

**Guarantee:**  
Load is isolated per CRD.

---

### 5.9 If Orkestra is upgraded

- No operator rebuilds  
- No controller redeployments  
- No CRD changes required  
- Runtime resumes from current state  

**Guarantee:**  
Upgrades do not break existing operators.

---

## 6. Summary Table

| Scenario | Guarantee |
|---------|-----------|
| Orkestra down | Cluster continues normally |
| Reconcile error | Safe retry, no corruption |
| Reconcile panic | Isolated, worker continues |
| Invalid Katalog | Rejected before runtime |
| Malformed CR | Visible failure, safe retry |
| Missing dependencies | No global stall |
| Registry failure | Idempotent, recoverable |
| Queue overload | CRD‑level isolation |
| Upgrade | No operator rebuilds needed |

---

## Final Note

This combined model gives Orkestra the same reliability expectations as:

- Kubernetes controllers  
- Crossplane providers  
- ArgoCD controllers  
- Operator SDK controllers  

But with **zero code**, **zero typed APIs**, and **zero controller boilerplate**.

- **Next:** [Production Metrics](./metrics-analysis.md)