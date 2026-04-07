---
title: "Trust and Failure Model"
weight: 50
description: "This document describes exactly what happens when Orkestra fails, degrades, or is deliberately stopped. It covers every ..."
---

This document describes exactly what happens when Orkestra fails, degrades, or is deliberately stopped. It covers every failure mode — process crash, panic, leader loss, API server disconnect, node failure, graceful shutdown — and what each means for the CRs Orkestra manages.

The answer is strong. The failure model is designed to never corrupt CR state, never orphan child resources, and never block cluster operations.

---

## The foundational guarantee

Orkestra's failure model rests on a single foundational guarantee:

**Any operation that Orkestra does not complete will be completed on the next reconcile.**

Kubernetes provides the infrastructure for this guarantee: CRs are stored in etcd, watch events are delivered reliably, and the informer pattern ensures no events are missed across restarts.

---

## Leader election (konductor election)

Orkestra uses Kubernetes leader election — the same mechanism used by `kube-controller-manager` — to ensure that only one Orkestra instance actively reconciles at any time.

### What followers do while not leading

Every Orkestra instance — leader and followers — runs its informers and
populates its local caches continuously. When a follower wins the lease,
it has a warm cache and starts reconciling in seconds, not minutes.

### Failover timeline

```
t=0:  Leader pod crashes (OOM, node failure, SIGKILL)
t=0:  Lease stops being renewed
t=15: Lease expires (leaseDuration)
t=15: Follower acquires lease immediately (it was waiting)
t=15: Follower workers start dequeuing
t=16: First reconcile completes in the new leader
```

Worst-case failover time: **leaseDuration** (default 15 seconds).

---

## Panic recovery (safeReconcile)

Every reconcile call is wrapped in `safeReconcile`:

```go
func safeReconcile(ctx context.Context, fn func(ctx context.Context, key string) error, key string) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("reconciler panic recovered: %v\n%s", r, debug.Stack())
        }
    }()
    return fn(ctx, key)
}
```

### What happens when a reconciler panics

1. The panic is caught by the deferred recover
2. The stack trace is logged at ERROR level with the full goroutine stack
3. A Warning Kubernetes event is emitted on the CR that caused the panic
4. The error is returned to the workqueue
5. The workqueue requeues the item with exponential backoff
6. **Other CRDs are completely unaffected** — each CRD has its own worker pool

---

## Graceful shutdown

When Orkestra receives SIGTERM (standard Kubernetes pod termination):

```
SIGTERM received
  │
  ▼
Context cancelled — propagates to all components
  │
  ▼
Workers stop accepting new queue items (queue.ShutDown())
  │  In-flight reconciles are allowed to complete
  │
  ▼
Workers exit when current reconcile completes
  │
  ▼
Informers stop watching
  │
  ▼
HTTPS server drains open connections (30s timeout)
  │
  ▼
Process exits 0
```

---

## Summary: what can go wrong and what Orkestra does

| Failure | Effect | Recovery |
|---|---|---|
| Reconciler panic | CR left in partial state | Requeued with backoff, corrected on next reconcile |
| Process crash | Reconciliation paused for up to 15s | Follower acquires lease, resumes immediately |
| Node failure | Reconciliation paused for up to 15s | Follower on healthy node acquires lease |
| API server disconnect | Reconcile writes fail, queued for retry | Automatic reconnection, retry on restore |
| Graceful shutdown | In-flight reconcile completes, then stops | New instance picks up pending events |
| Admission webhook unreachable | Objects stored without synchronous validation | Reconcile-time validation corrects violations |
| Mutation error | Object stored without defaults | Reconcile-time mutation applies defaults |
| Leader lease expired (no follower) | Reconciliation paused until a leader is elected | New instance or connectivity restore |
| CRD degraded | Reconciliation continues, health API shows degraded | Recovers when a reconcile succeeds |


---

## Related Documentation
- [Orkestra Shutdown](../reference/shutdown.md)
