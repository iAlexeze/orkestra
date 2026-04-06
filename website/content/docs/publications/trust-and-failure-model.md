---
title: "Trust and Failure Model"
weight: 50
description: "This document describes exactly what happens when Orkestra fails, degrades, or is deliberately stopped. It covers every ..."
---

This document describes exactly what happens when Orkestra fails, degrades, or is deliberately stopped. It covers every failure mode — process crash, panic, leader loss, API server disconnect, node failure, graceful shutdown — and what each means for the CRs Orkestra manages.

The answer is strong. The failure model is designed to never corrupt CR state, never orphan child resources, and never block cluster operations. Understanding it is essential for production operations and for evaluating Orkestra's reliability for your use case.

---

## The foundational guarantee

Orkestra's failure model rests on a single foundational guarantee:

**Any operation that Orkestra does not complete will be completed on the next reconcile.**

Kubernetes provides the infrastructure for this guarantee: CRs are stored in etcd, watch events are delivered reliably, and the informer pattern ensures no events are missed across restarts. Orkestra's reconciler is level-triggered — it always works from the current desired state, not from a log of events. A reconcile that is interrupted half-way through leaves the cluster in a partially-applied state, not a corrupted one. The next reconcile corrects it.

This guarantee holds for every failure mode described below.

---

## Leader election (konductor election)

Orkestra uses Kubernetes leader election — the same mechanism used by `kube-controller-manager` — to ensure that only one Orkestra instance actively reconciles at any time.

### How it works

Leader election uses a Kubernetes `Lease` object (in `coordination.k8s.io/v1`):

```
Lease: orkestra-leader
  holder: orkestra-pod-a
  renewTime: <timestamp>
  leaseDuration: 15s
  renewDeadline: 10s
  retryPeriod: 2s
```

!!! tip "Leader configuration"
    These values are all configurable using the following environment variables:

    LEADER_ELECTION_NAMESPACE=  # Default: default
    LEASE_DURATION=             # Seconds; total lease duration
    RENEW_DEADLINE=             # Seconds; must be < LEASE_DURATION
    RETRY_PERIOD=               # Seconds; retry interval for leader election


The active leader renews the lease every 10 seconds. If renewal fails for 15 seconds (leaseDuration), the lease expires and any other instance can acquire it.

### What followers do while not leading

!!! important "Followers are not idle"
    This is the most important thing to understand about the failure model.
    Follower instances are **not** idle while waiting for the lease.

    Every Orkestra instance — leader and followers — runs its informers and
    populates its local caches continuously. When a follower wins the lease,
    it has a warm cache and starts reconciling in seconds, not minutes.

The informers run in all instances. All caches are warm. The workqueues are dormant in followers — events are added to the queue but workers do not dequeue. When leadership transfers, the new leader's workers start draining a queue that already contains all pending events.

### Failover timeline

```
t=0:  Leader pod crashes (OOM, node failure, SIGKILL)
t=0:  Lease stops being renewed
t=15: Lease expires (leaseDuration)
t=15: Follower acquires lease immediately (it was waiting)
t=15: Follower workers start dequeuing
t=16: First reconcile completes in the new leader
```

Worst-case failover time: **leaseDuration** (default 15 seconds). In practice, if a follower is already running with a warm cache, it is closer to 16-17 seconds from crash to first reconcile in the new leader.

During this 15-second window, reconciliation is paused. CRs created or updated during this window are queued and will be processed when the new leader starts. No CR changes are lost — etcd stores them and the informer delivers them to the new leader's queue.

!!! note "Admission webhooks during failover"
    If ENABLE_ADMISSION_WEBHOOK=true, the /validate and /mutate endpoints are served
    by all running Orkestra pods (the HTTPS server runs in all instances, not
    just the leader). Admission interception continues during leader failover
    because the HTTPS server does not participate in leader election.

    The ValidatingWebhookConfiguration points to the Service, which load-balances
    across all healthy pods. As long as one Orkestra pod is running, admission
    webhooks work.

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

The panic does not crash the process. It does not affect other CRDs. The affected CR is retried after a backoff delay.

### What gets logged

```json
{
  "level": "error",
  "crd": "demo.orkestra.io/v1alpha1, Kind=Website",
  "cr": "my-site",
  "message": "reconciler panic recovered: runtime error: index out of range [5] with length 3",
  "stacktrace": "goroutine 42 [running]:\nruntime/debug.Stack()\n\t/usr/local/go/src/runtime/debug/stack.go:24 +0x5b\n..."
}
```

The full stack trace is logged. The CR name and CRD are logged. You can identify exactly which CR triggered the panic and which line of code caused it.

### What happens to the CR

The CR is left in whatever state it was in when the panic occurred. If the panic happened after the Deployment was created but before the Service was created, the Deployment exists and the Service does not. On the next reconcile cycle, the Service will be created. The state is partial, not corrupted.

!!! note "safeReconcile is not optional"
    safeReconcile wraps every reconcile call — built-in GenericReconciler and
    custom constructors. It cannot be disabled. A Go panic in your hook code
    or custom reconciler will always be caught and the process will always
    continue running.

---

## API server disconnect

If the Kubernetes API server becomes unreachable, Orkestra's behavior depends on the operation:

**Informer watches:** The informer library handles reconnection automatically with exponential backoff. The local cache remains valid during the disconnect — events are buffered by the API server and delivered when the connection is restored. Orkestra will not miss events that occurred during the disconnect.

**Reconcile writes:** Any reconcile that attempts to create or update a Kubernetes resource will fail with a connection error. The error is logged, the item is requeued with backoff, and the reconcile retries when the API server is reachable again.

**Leader election renewal:** If the API server is unreachable for longer than `renewDeadline` (default 10 seconds), the current leader cannot renew its lease. After `leaseDuration` (default 15 seconds), the lease expires. If no other instance can reach the API server either, the lease simply remains expired and no instance leads. Reconciliation pauses until connectivity is restored.

There is no split-brain risk — the lease is stored in etcd, and etcd requires quorum. If the API server is unreachable, no instance can claim leadership.

---

## Node failure

If the node running the active Orkestra leader fails:

1. The lease stops being renewed at `t=0` (node failure)
2. At `t=15` (leaseDuration), the lease expires
3. A follower on a healthy node acquires the lease
4. The follower's warm cache contains all CR states
5. Reconciliation resumes

The failed node's pod will be rescheduled by Kubernetes (if Deployment replicas > 1). When it restarts, it joins as a follower with a fresh informer cache that warms within seconds.

**During failover**, CRs are not deleted or modified by Orkestra. Kubernetes objects are persistent in etcd — they do not disappear because the operator is unavailable.

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
  │  No new items are dequeued after ShutDown() is called
  │
  ▼
Workers exit when current reconcile completes
  │  Timeout is configurable using "SHUTDOWN_TIMEOUT" and "SHUTDOWN_GRACE_PERIOD"
  │
  │
  ▼
Informers stop watching
  │
  ▼
HTTPS server drains open connections (30s timeout)
  │
  ▼
HTTP server shuts down
  │
  ▼
Process exits 0
```

### Graceful shutdown timeline

Orkestra performs a coordinated, configurable shutdown when it receives SIGTERM.  
Two settings control how long the system waits for in‑flight work to finish:

- **`SHUTDOWN_TIMEOUT`** — how long each CRD’s workers are allowed to drain ongoing reconciles before Orkestra moves on.  
- **`SHUTDOWN_GRACE_PERIOD`** — the overall upper bound for the entire shutdown sequence; once this expires, Orkestra exits immediately.

`SHUTDOWN_GRACE_PERIOD` should always be larger than `SHUTDOWN_TIMEOUT`.  
Users can tune both values based on the expected duration of their reconcile or cleanup logic.

---

### Finalizer safety during shutdown

If a CR is being deleted (DeletionTimestamp set) and the reconcile is interrupted by shutdown before finalizers are removed, the CR will be stuck in terminating state until the next Orkestra instance starts and processes it.

This is a safe state — the CR is not deleted, its child resources are not deleted, and the cluster is consistent. The next reconcile will run the `onDelete` templates or hooks and then remove the finalizers.

---

## CRD health states

Each CRD has one of four health states, reported by `/katalog/{crd}/health` and `ork status`:

| State | Meaning | `/katalog/{crd}/health` |
|---|---|---|
| `starting` | Informer not yet synced, workers not yet running | 503 |
| `healthy` | All workers running, recent reconcile succeeded | 200 |
| `degraded` | `consecutiveFails >= degradeThreshold` | 503 |
| `disabled` | `enabled: false` in Katalog | 200 (excluded from runtime) |

**Degraded** does not stop reconciliation — workers continue running. It is a signal that something is systematically wrong. The CRD recovers to healthy when a reconcile succeeds (resets `consecutiveFails` to zero).

<!-- **Critical CRDs:** A CRD marked `critical: true` that becomes degraded transitions the entire Orkestra health state to degraded. This causes the `/health` probe to return 503, which may trigger pod restart depending on the liveness probe configuration. -->

---

## Admission webhook failure model

If `ENABLE_ADMISSION_WEBHOOK=true` and the Orkestra HTTPS server is unreachable when the API server calls `/validate` or `/mutate`:

The behavior depends on `FailurePolicy` in the webhook configuration:

| FailurePolicy | Behavior when Orkestra unreachable |
|---|---|
| `Ignore` (default) | API server allows the operation. Object is stored. Reconcile-time validation catches violations on the next cycle. |
| `Fail` | API server rejects the operation with `503 Service Unavailable`. `kubectl apply` returns an error. |

Orkestra uses `FailurePolicy: Ignore` by default. This means:
- A brief Orkestra outage does not prevent users from deploying CRs
- CRs deployed during an outage are validated at reconcile time when Orkestra restarts
- Users experience slightly degraded validation (later feedback, not no feedback)

!!! warning "Choosing FailurePolicy: Fail"
    Setting `FailurePolicy: Fail` means Orkestra's availability directly
    gates all CR deployments. If Orkestra is unavailable for any reason
    (restart, update, outage), no CRs can be created or updated.

    Only set `Fail` when you require strict synchronous enforcement and
    have high-availability Orkestra deployments (multiple replicas, PodDisruptionBudget).

---

## Mutation webhook failure model

Mutation webhooks follow a stricter non-blocking rule. If a mutation rule fails — template evaluation error, patch construction error, any error — the webhook returns `allowed: true` without a patch. The object is stored without mutation.

This is intentional: **mutation must never block.** A bug in a mutation rule expression should not prevent a CR from being created. The reconciler will apply defaults at reconcile time if the admission-time mutation failed.

If you observe `admission/mutate: error applying rules — allowing without mutation` in logs, the CR was stored without defaults being applied. The reconciler will correct this on the next cycle.

---

## The kube-controller-manager analogy

The Orkestra failure model is identical to `kube-controller-manager` — the process that manages Deployments, ReplicaSets, Jobs, and dozens of other controllers. `kube-controller-manager`:

- Uses Kubernetes leader election
- Runs multiple isolated controllers in one process
- Wraps each reconcile to prevent panics from crashing the process
- Uses informer-based level-triggered reconciliation
- Provides no stronger delivery guarantees than "at least once"

Kubernetes clusters trust `kube-controller-manager` with cluster-wide control plane management. The failure model that makes this trustworthy is the same model Orkestra uses for custom resources.

The operational question is not "can this process fail?" — all processes fail. The question is "what happens when it does?" For both `kube-controller-manager` and Orkestra, the answer is: failover within the lease duration, no data corruption, no orphaned resources, and full correctness restoration on the next reconcile.

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