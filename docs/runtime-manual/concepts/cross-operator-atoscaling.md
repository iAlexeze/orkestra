# Cross‑Operator Metrics Internals  
*How Orkestra exposes live runtime metrics across operators.*

Cross‑operator metrics allow one operator to read the **live runtime metrics** of another operator through Orkestra’s cross‑operator IPC layer.  
This enables cross‑operator autoscaling, distributed backpressure, and pipeline‑wide coordination.

This document explains the internal data flow:

```
AutoMetrics → GenericReconciler → readCross → Resolver → Autoscaler
```

…and how metrics remain:

- **in‑memory**
- **atomic**
- **zero‑API‑call**
- **safe across binaries and clusters**

---

## 1. Where Metrics Come From: `AutoMetrics`

Every operatorBox: maintains its own `AutoMetrics` instance:

- updated continuously by the worker pool  
- atomic, lock‑free  
- read by the autoscaler and cross‑operator IPC  

`AutoMetrics` tracks:

- queue depth  
- busy/idle worker percentage  
- reconcile P95 duration  
- error rate  
- total reconciles  

These values are updated on:

- every enqueue  
- every dequeue  
- every reconcile start  
- every reconcile completion  
- every reconcile error  

### Internal structure (simplified)

```go
type AutoMetrics struct {
    queueDepth atomic.Int64
    reconcileErrors atomic.Int64
    reconcileTotal atomic.Int64
    workerSem *ResizableSemaphore
    p95 *rollingP95
}
```

### Exported as a map

```go
func (m *AutoMetrics) AsMap() map[string]interface{} {
    return map[string]interface{}{
        "queueDepth": m.queueDepth.Load(),
        "workersBusyPercent": m.WorkersBusyPercent(),
        "workersIdlePercent": m.WorkersIdlePercent(),
        "reconcileDurationP95Ms": m.P95Ms(),
        "errorRatePercent": m.ErrorRatePercent(),
    }
}
```

This map is what gets injected into cross‑operator IPC.

---

## 2. How Metrics Enter Cross‑Operator IPC

Cross‑operator IPC is handled by:

```go
func (r *GenericReconciler[T]) readCross(...)
```

This function resolves each `cross:` declaration and returns a map injected into the template resolver.

### The key insight

The `GenericReconciler` **already holds**:

```go
autoMetrics *AutoMetrics
```

So adding metrics to cross‑operator IPC is trivial:

```go
if r.autoMetrics != nil {
    result[as].(map[string]interface{})["metrics"] = r.autoMetrics.AsMap()
}
```

This attaches the live metrics of the **current operator** to the cross‑operator result map.

---

## 3. Informer Path (Same Binary)

When the referenced operator is in the same binary:

```
readCross → ReadCrossFromInformer → attach metrics → return
```

### Informer path characteristics

- zero API calls  
- zero network calls  
- pure in‑memory lookup  
- fastest possible path  

### Returned structure

```json
{
  "found": "true",
  "name": "db",
  "namespace": "default",
  "spec": {...},
  "status": {...},
  "labels": {...},
  "metrics": {
    "queueDepth": 42,
    "workersBusyPercent": 67,
    "errorRatePercent": 1,
    "reconcileDurationP95Ms": 12
  }
}
```

This is what the autoscaler sees.

---

## 4. HTTP Fallback Path (Cross‑Binary / Cross‑Cluster)

If the referenced operator is in another binary or cluster:

```
readCross → fetchCrossViaHTTP → attach metrics → return
```

### HTTP endpoint

Every Orkestra instance exposes:

```
/katalog/{crd}/cr/{namespace}/{name}
```

This returns:

- spec  
- status  
- labels  
- found flag  
- metrics  

### Why this is safe

- HTTP calls are short‑lived (5s timeout)  
- Errors return `"found": "false"`  
- Autoscaler treats missing metrics as false conditions  
- No operator ever blocks on another  

---

## 5. Resolver Injection

Once `readCross` returns the map, it is injected into the template resolver:

```
resolver.WithCross(result)
```

This makes fields available as:

```
cross.<alias>.metrics.queueDepth
cross.<alias>.metrics.workersBusyPercent
cross.<alias>.metrics.errorRatePercent
```

The autoscaler’s condition engine reads these fields exactly like local metrics.

---

## 6. Autoscaler Evaluation

On every autoscaler tick:

1. Resolver resolves all `cross.*` fields  
2. Autoscaler reads the resolved values  
3. Conditions are evaluated  
4. Overrides apply or baseline restores  

### Example evaluation

```yaml
- field: cross.db.metrics.queueDepth
  greaterThan: "500"
```

If:

```
cross.db.metrics.queueDepth = 742
```

→ condition is true.

If the referenced operator is not found:

```
cross.db.metrics = nil
```

→ condition is false.

---

## 7. Safety Guarantees

Cross‑operator metrics are designed to be safe:

### No deadlocks  
Operators never wait on each other.  
Metrics are read‑only.

### No cycles  
Even if A references B and B references A, both reads are independent.

### No flapping  
Cooldown applies normally.

### No dropped items  
Queue depth changes never drop items.

### No goroutine leaks  
Worker semaphore resizing is safe.

---

## 8. Performance Characteristics

### Same‑binary  
- O(1) lookup  
- zero allocations  
- zero network  
- zero API calls  

### Cross‑binary  
- one HTTP GET  
- 64KB response limit  
- 5s timeout  
- safe fallback to `"found": "false"`  

### Autoscaler  
- O(1) evaluation  
- lock‑free reads  
- no heap allocations  

This is why cross‑operator autoscaling feels instantaneous.

---

## 9. Summary

Cross‑operator metrics flow through Orkestra like this:

```
AutoMetrics (live signals)
        ↓
GenericReconciler.readCross
        ↓
Informer or HTTP fallback
        ↓
Resolver.WithCross
        ↓
Autoscaler condition engine
        ↓
Scaling decisions
```

This design gives Orkestra:

- distributed load awareness  
- pipeline‑wide coordination  
- upstream/downstream feedback loops  
- cross‑cluster autoscaling  
- zero‑API‑call performance  

It is one of the most powerful capabilities in the entire runtime.