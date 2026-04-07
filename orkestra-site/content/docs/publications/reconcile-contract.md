---
title: "Reconcile Contract"
weight: 70
---

# The Reconcile Contract

This document describes exactly what Orkestra guarantees about how and when
reconciliation happens. Understanding these guarantees prevents an entire class
of operator bugs — hooks that assume ordering, status updates that assume
single delivery, and cleanup logic that assumes a clean execution environment.

---

## At-least-once delivery

Every change to a CR will be reconciled at least once. It will not necessarily
be reconciled exactly once, and it will not be reconciled immediately.

**What this means in practice:**

A CR updated three times in rapid succession produces a minimum of one reconcile
and a maximum of three. The workqueue deduplicates: while a reconcile for
`my-website` is in flight, additional change events for `my-website` are merged
into one pending item. After the current reconcile completes, one more reconcile
runs to process all changes that arrived while the first was in progress.

In the presence of retries — a reconcile that returns an error is requeued with
exponential backoff — the same CR may be reconciled many times. Your hook or
template reconciler will run multiple times for the same object state. This is
not a bug. It is the expected behaviour of a level-triggered control loop.

**The idempotency requirement:**

Every reconcile function must produce the same outcome when called multiple
times with the same input. `orkdeploy.Create` is idempotent by design — it
checks for existence before creating. `orkdeploy.Update` applies desired state
regardless of current state — inherently idempotent. Go hooks must be written
with the same property.

A hook that creates an external database user must check whether the user already
exists before attempting creation. A hook that sends a notification must track
whether it has already been sent — in the CR's status or via an idempotency key
in the external system. A hook that calls a billing API must be safe to call
twice with the same arguments.

:::warning [Non-idempotent hooks produce subtle production bugs]
    A hook that charges a credit card on reconcile will charge twice if the
    reconcile is retried due to a transient network error. A hook that creates
    a DNS record will fail on the second reconcile if the record already exists
    and the error is not handled. These bugs are invisible in development and
    appear only under production load or network instability.
:::
---

## No ordering between CRs

Orkestra makes no guarantees about the order in which CRs of the same type
are reconciled.

If ten `Website` CRs are created simultaneously, they are enqueued and processed
by the worker pool in an unspecified order. Worker A may process `site-1` while
Worker B processes `site-7`. There is no guarantee that `site-1` is reconciled
before `site-2`.

**What ordering is available:**

`dependsOn` provides ordering between CRD types at startup — `application` does
not start reconciling until `namespace` is ready. But this is startup ordering,
not per-CR ordering. Once both CRDs are running, `application` CRs and `namespace`
CRs are reconciled independently and concurrently.

If your reconcile logic requires that resource A exists before resource B, declare
that dependency in the CR's spec and check it in the hook:

```go
OnReconcile: func(ctx context.Context, obj *apiv1.Application) error {
    // Check that the Namespace this Application depends on exists
    _, err := kube.DynamicClient().Resource(namespaceGVR).Get(ctx, obj.Spec.Namespace, ...)
    if err != nil {
        // Namespace not ready — return an error to requeue
        return fmt.Errorf("waiting for namespace %q: %w", obj.Spec.Namespace, err)
    }
    // Proceed
}
```

The requeue with backoff is the coordination mechanism between dependent CRs.

---

## Eventual consistency, not immediate consistency

When a CR is updated, the cluster converges toward the declared state — it does
not snap to it immediately.

The reconcile cycle runs when an event arrives from the informer. Between events,
child resources exist in whatever state the last reconcile left them. If a Deployment
is manually scaled to zero between reconcile cycles, it stays at zero until the
next reconcile triggers. With `reconcile: true` on the Deployment template, this
is corrected at the next cycle. Without it, it stays at zero until the CR is
updated.

**Resync is the background corrector:**

The resync interval (default 15 seconds, configurable per CRD) triggers a
reconcile for every CR of that type even without a change event. This is how
drift is detected and corrected for resources declared with `reconcile: true`.

After `kubectl edit` changes a Deployment manually, the maximum time before
correction is one resync interval. Not immediate — eventual.

**The implication for status:**

Status fields written by Layer 2 declarations are eventually consistent with
the spec fields they reference. `status.observedReplicas: "{{ .spec.replicas }}"`
reflects the spec at the time of the last successful reconcile — not necessarily
the current spec. If the user updates `spec.replicas` at 10:00 and the next
reconcile runs at 10:00:15, `status.observedReplicas` is stale for 15 seconds.
This is correct and expected.

---

## What `reconcile: true` actually guarantees

`reconcile: true` on a template resource means: on every reconcile cycle, apply
the desired state for this resource, regardless of its current state.

It does not mean: the resource is always exactly as declared at all times.

The guarantee is: within one resync interval after any drift, the resource will
be corrected. The window between drift and correction is bounded by `resync`.
The default is 15 seconds. For production operators where drift must be corrected
quickly, set `resync: 5s` or lower — at the cost of higher API server load.

---

## Finalizer guarantees

Orkestra adds a finalizer to every CR it manages before any reconcile logic runs.
The finalizer guarantees that the CR will not be deleted by Kubernetes until
Orkestra removes it.

**What finalizers guarantee:**
- `kubectl delete website my-site` sets `DeletionTimestamp` and the CR enters
  a terminating state. Orkestra sees the timestamp, runs `onDelete` templates
  or hooks, and then removes its finalizer.
- After all finalizers are removed, Kubernetes proceeds with deletion.
- The CR is not deleted while Orkestra's finalizer is present.

**What finalizers do not guarantee:**
- The CR cannot be updated while in a terminating state. Updates are rejected
  by the API server.
- If Orkestra is unavailable when the CR is deleted, the CR stays in terminating
  state until Orkestra resumes. This is the safe state — not deleted, not cleaned
  up. The next Orkestra instance finishes the cleanup.
- Finalizers do not prevent the CRD itself from being deleted. Deleting the CRD
  while CRs exist force-deletes the CRs without running finalizers.

{{< callout type="warning" title="Always delete CRs before deleting the CRD" >}}
`kubectl delete crd websites.demo.orkestra.io` while `Website` CRs exist
will force-delete all CRs, bypassing finalizers and `onDelete` logic.
The child resources (Deployments, Services) will remain as orphans — no
owner to garbage-collect them.
{{< /callout >}}

---

## The safeReconcile boundary

Every reconcile call is wrapped in `safeReconcile`, which catches panics and
returns them as errors:

```go
defer func() {
    if r := recover(); r != nil {
        err = fmt.Errorf("reconciler panic: %v\n%s", r, debug.Stack())
    }
}()
```

**What this guarantees:** A panic in a reconcile function — nil pointer
dereference, slice out of bounds, type assertion failure — does not crash the
process. It is caught, logged with the full stack trace, and returned to the
workqueue as an error. The CR is requeued with backoff.

**What it does not guarantee:** The reconcile that panicked may have partially
applied state. A Deployment may have been created before the panic; the Service
may not have been. The next reconcile corrects this — `Create` is idempotent and
`Update` applies desired state. The partial state is transitional, not permanent.

**Other CRDs are unaffected.** A panic in the `Website` reconciler does not
affect the `Database` reconciler or any other CRD. Each CRD has its own worker
goroutines. `safeReconcile` wraps each worker call independently.

---

## Summary

| Guarantee | What it means |
|---|---|
| At-least-once delivery | Every change is reconciled at least once. Multiple reconciles for the same state are expected and safe. |
| No per-CR ordering | CRs of the same type reconcile in unspecified order. Hooks must not assume ordering between CRs. |
| Eventual consistency | Drift is corrected within one resync interval, not immediately. |
| `reconcile: true` window | Drift corrected within `resync` (default 15s), not instantaneously. |
| Finalizer protection | CRs are not deleted until Orkestra completes cleanup and removes its finalizer. |
| Panic isolation | A panic in one CRD's reconciler does not affect other CRDs. |
| Idempotency requirement | Every reconcile function must be safe to call multiple times with the same input. |
