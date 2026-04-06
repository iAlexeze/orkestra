---
title: "The Reconcile Contract"
weight: 50
description: "This document describes exactly what Orkestra guarantees about how and when"
---

This document describes exactly what Orkestra guarantees about how and when
reconciliation happens. Understanding these guarantees prevents an entire class
of operator bugs — hooks that assume ordering, status updates that assume
single delivery, and cleanup logic that assumes a clean execution environment.

---

## At-least-once delivery

Every change to a CR will be reconciled at least once. It will not necessarily
be reconciled exactly once, and it will not be reconciled immediately.

**The idempotency requirement:**

Every reconcile function must produce the same outcome when called multiple
times with the same input. `orkdeploy.Create` is idempotent by design — it
checks for existence before creating.

A hook that creates an external database user must check whether the user already
exists before attempting creation.

---

## No ordering between CRs

Orkestra makes no guarantees about the order in which CRs of the same type
are reconciled.

If ten `Website` CRs are created simultaneously, they are enqueued and processed
by the worker pool in an unspecified order.

---

## Eventual consistency, not immediate consistency

When a CR is updated, the cluster converges toward the declared state — it does
not snap to it immediately.

**Resync is the background corrector:**

The resync interval (default 15 seconds, configurable per CRD) triggers a
reconcile for every CR of that type even without a change event.

---

## What `reconcile: true` actually guarantees

`reconcile: true` on a template resource means: on every reconcile cycle, apply
the desired state for this resource, regardless of its current state.

The guarantee is: within one resync interval after any drift, the resource will
be corrected.

---

## Finalizer guarantees

Orkestra adds a finalizer to every CR it manages before any reconcile logic runs.
The finalizer guarantees that the CR will not be deleted by Kubernetes until
Orkestra removes it.

---

## The safeReconcile boundary

Every reconcile call is wrapped in `safeReconcile`, which catches panics and
returns them as errors.

**What this guarantees:** A panic in a reconcile function does not crash the
process. It is caught, logged with the full stack trace, and returned to the
workqueue as an error.

**Other CRDs are unaffected.** A panic in the `Website` reconciler does not
affect the `Database` reconciler or any other CRD.

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
