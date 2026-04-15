# Autoscaler Runtime Behavior
*How the Operator Autoscaler executes inside Orkestra.*

The Operator Autoscaler runs **inside Kordinator**, alongside the worker pool, queue manager, and health engine. It is a lightweight, per‑CRD controller that evaluates autoscale conditions on a fixed interval and applies overrides immediately when conditions are met.

The autoscaler is **authoritative** for three runtime parameters:

- worker count  
- queue depth  
- resync interval  

Kordinator trusts the autoscaler to manage these values throughout the operator lifecycle.

---

## 1. Lifecycle

The autoscaler is activated when:

```yaml
operatorBox:
  autoscale: { ... }
```

is present in the CRD’s Katalog entry.

At Katalog load time:

1. The CRD’s declared `workers`, `queueDepth`, and `resync` are captured as the **baseline**.
2. A per‑CRD autoscaler goroutine is started.
3. The autoscaler begins evaluating conditions every `interval:`.

If autoscaling is not declared, the CRD runs with its baseline configuration for its entire lifetime.

---

## 2. Autoscaler loop

Each CRD with autoscaling enabled runs its own loop:

```
every interval:
    evaluate conditions
    if conditions true:
        apply overrides immediately
    else if conditions false for entire cooldown:
        restore baseline
```

This loop is independent of:

- the reconcile loop  
- the worker pool  
- the queue processor  
- the health engine  

Autoscaling does not block or interfere with normal reconciliation.

---

## 3. Applying overrides

When conditions evaluate to **true**, the autoscaler applies the `do:` block:

```yaml
do:
  workers: 12
  queueDepth: 1000
  resync: 20s
```

Overrides are applied **immediately** and **atomically**:

### Workers
The worker pool uses a resizable semaphore.  
Increasing workers raises the semaphore capacity.  
Decreasing workers reduces capacity after in‑flight reconciles complete.

No goroutines are killed.  
No work is interrupted.

### Queue depth
The queue’s maximum depth is updated in place.  
If the queue is already deeper than the new limit, no items are dropped — the limit only affects *new* enqueues.

### Resync
A dedicated resync goroutine re‑enqueues all objects at the override interval.  
When the override ends, this goroutine idles and the baseline resync takes over.

---

## 4. Restoring baseline

When conditions are false for the entire `cooldown:` period, the autoscaler restores:

- baseline workers  
- baseline queue depth  
- baseline resync interval  

Restoration is also atomic and uses the same mechanisms as override application.

A restart of Orkestra always begins from the baseline, never from an override.

---

## 5. Cooldown behavior

Cooldown prevents oscillation when metrics fluctuate around a threshold.

Example:

```
queueDepth: 195 → 205 → 198 → 210 → 190
```

Without cooldown:

- autoscaler would apply and revert on alternating ticks

With cooldown:

- override applies immediately when conditions become true  
- override reverts only after conditions remain false for the entire cooldown window  

Cooldown applies **only** to the revert direction.  
Overrides are always immediate.

---

## 6. Condition evaluation

On each tick, the autoscaler evaluates:

- `anyOf:` (OR)  
- `when:` (AND)  
- metric conditions  
- clock windows  
- day‑of‑week rules  
- cron expressions  

All evaluations are:

- in‑memory  
- constant‑time  
- independent of the API server  
- independent of informers  

This makes the autoscaler extremely fast and predictable.

---

## 7. Interaction with Kordinator

Kordinator delegates three responsibilities to the autoscaler:

| Responsibility | Owner |
|---|---|
| Decide when to scale | Autoscaler |
| Apply scaling decisions | Kordinator |
| Maintain baseline | Kordinator |

Kordinator does **not** second‑guess autoscaler decisions.  
The autoscaler is the **source of truth** for runtime scaling.

---

## 8. Metrics emitted by the autoscaler

The autoscaler exposes its own metrics:

| Metric | Description |
|---|---|
| `orkestra_autoscale_override_active{crd}` | 1 when override is active |
| `orkestra_autoscale_overrides_total{crd}` | Count of override activations |
| `orkestra_autoscale_restores_total{crd}` | Count of baseline restorations |
| `orkestra_autoscale_workers_current{crd}` | Current worker count |
| `orkestra_autoscale_queue_depth_current{crd}` | Current queue depth limit |

These appear in `/metrics` and the Control Center.

---

## 9. Safety guarantees

The autoscaler guarantees:

- No reconcile is ever interrupted  
- No worker goroutine is ever killed  
- No queue item is ever evicted  
- No override persists after restart  
- No invalid metric field is accepted  
- No cron/time/dayOfWeek expression is applied without validation  

Autoscaling is safe, reversible, and deterministic.

---

## 10. Summary

The autoscaler is a **first‑class runtime controller** inside Orkestra.  
It evaluates conditions declaratively, applies overrides instantly, and restores baseline safely.

It transforms OperatorBoxes from static execution cells into **adaptive, self‑optimizing runtime units**.
