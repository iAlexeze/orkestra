# Async Reconciliation Without Async Code

*Orkestra Project — April 2026*

---

## Abstract

Kubernetes reconcilers are synchronous by contract: receive an event, reconcile,
return. Real systems are not. A Deployment takes time to become ready. An
external database takes time to accept connections. A dependent CRD takes time
to reach a stable state. Traditional operator authors resolve this mismatch by
hand — polling loops, custom state machines, phase annotations, manual requeue
logic. Every operator implements these patterns independently, inconsistently,
and with varying correctness. Orkestra resolves the mismatch at the architecture
level. The `when:` gate on `onReconcile` is a declarative skip instruction: if
the condition is false, the block does not execute and the CR is requeued. No
sleeping. No blocking. No custom state machine. The reconciler returns
immediately, and the Kubernetes watch mechanism calls it again when something
changes. This paper describes how the mechanism works, why it is correct, and
what it replaces.

---

## 1. The reconciliation contract and its constraints

The Kubernetes reconciliation contract is intentionally minimal. A reconciler
receives a key — typically `namespace/name` — looks up the current state of the
object, compares it to desired state, and takes corrective action. It then
returns. The contract makes no provision for waiting, blocking, or long-running
operations. If a reconciler blocks for seconds, it occupies a worker goroutine
that could be processing other items from the queue. If it blocks for minutes,
it causes visible latency for every CR in the same worker pool.

This constraint is the right design for the general case. Most reconcile
operations — updating a Deployment's image, patching a Service's selector,
writing a ConfigMap — complete in milliseconds. The constraint becomes a problem
when the desired end state requires an intermediate state to stabilize first.

A website operator that creates a Deployment and then immediately creates an
Ingress pointing to a Service that points to that Deployment is correct in the
declarative sense — all three resources reflect desired state. But the system
is not useful until the Deployment's pods are running and the Service has
endpoints. If the operator creates all three in one reconcile cycle, the Ingress
will resolve to a Service with no backends. This is not an error the operator
can prevent by writing Go differently. It is a property of distributed systems:
desired state and actual state converge over time, not instantly.

---

## 2. What traditional operators do

The traditional solution is a manually implemented state machine. The reconciler
checks a phase annotation on the CR, executes the action appropriate for that
phase, advances the phase, and returns. On the next reconcile, it reads the
phase and continues.

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    var website v1.Website
    if err := r.Get(ctx, req.NamespacedName, &website); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    switch website.Status.Phase {
    case "":
        return r.reconcileCreating(ctx, &website)
    case "Creating":
        return r.reconcileWaitingForDeployment(ctx, &website)
    case "DeploymentReady":
        return r.reconcileCreatingIngress(ctx, &website)
    case "Ready":
        return r.reconcileRunning(ctx, &website)
    }
    return ctrl.Result{}, nil
}
```

This approach has four structural problems.

**Phase state is external to the reconciler.** The phase is stored in
`status.phase`, which means it persists to etcd, is readable by users, and
must be kept consistent with the actual resource state. A bug that advances
the phase without creating the resource, or creates the resource without
advancing the phase, corrupts the state machine in ways that are difficult to
detect and recover from.

**Requeue logic is manual.** `reconcileWaitingForDeployment` must return
`ctrl.Result{RequeueAfter: 5 * time.Second}` to ensure the reconciler is
called again before the informer watch fires. This interval is a guess — too
short causes unnecessary API server load, too long causes unnecessary latency.

**The state machine is not the operator logic.** The operator's actual business
logic — create a Deployment with this image and this replica count, create a
Service with this selector — is buried inside the phase handlers. The state
machine is scaffolding that surrounds it. This scaffolding must be tested,
maintained, and understood before any change to the operator logic can be made
safely.

**Idempotency must be proven for each phase.** Each phase handler must be safe
to call multiple times. The informer watch may fire more than once between
reconcile calls, causing the same phase to execute repeatedly. Every resource
creation must check for existence before creating, every status update must be
a patch not a replace, every phase transition must be a conditional write.

---

## 3. The mechanism: declarative phase gating

Orkestra's `when:` gate eliminates the state machine by making phase conditions
explicit and external to the reconcile logic.

```yaml
operatorBox:
  default: true

  onCreate:
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        reconcile: true

  onReconcile:
    when:
      - field: children.deployment.status.readyReplicas
        equals: "{{ .spec.replicas }}"
    services:
      - name: "{{ .metadata.name }}-svc"
        port: 80
        targetPort: 8080
        reconcile: true
    ingresses:
      - name: "{{ .metadata.name }}-ingress"
        host: "{{ .spec.host }}"
        serviceName: "{{ .metadata.name }}-svc"
        reconcile: true
```

On the first reconcile, `onCreate` runs and creates the Deployment. The
`onReconcile` block is evaluated: `children.deployment.status.readyReplicas`
does not yet equal `spec.replicas`, so the block is skipped. The reconciler
returns. No error. No explicit requeue.

The Deployment's informer watch fires when the pod count changes. The CR is
requeued. On the next reconcile, the condition is evaluated again. If the
Deployment is still not ready, the block is skipped again. When the Deployment
reaches the desired replica count, the condition passes and the Service and
Ingress are created.

This is async reconciliation. The operator does not wait, sleep, or block. It
declares a condition and lets the runtime handle the timing.

---

## 4. Children observation: the data that makes it possible

The `when:` gate references `children.deployment.status.readyReplicas`. This
field is not in the CR's spec. It is the live status of a child resource —
the Deployment that Orkestra created for this CR in a previous reconcile cycle.

Orkestra populates the `.children.*` namespace in the resolver before every
reconcile. Child resources are identified by the `orkestra-owner` label that
Orkestra sets on every resource it creates. The children lookup reads from the
API server's watch cache using `ResourceVersion: "0"` — no etcd round-trip,
no added latency.

The result is that every template expression in the Katalog has access to the
live state of every child resource without any Go code:

```
{{ readyReplicas .children.deployment }}                       — number of ready pods
{{ desiredReplicas .children.deployment }}                     — total desired pods
{{ serviceLoadBalancerIP .children.service }}                  — LB ingress IP
{{ serviceLoadBalancerHost .children.service }}                — LB ingress hostname
{{ jobSucceeded .children.job }}                               — true when job completed
{{ get .children.cronjob "status" "lastScheduleTime" }}        — last run timestamp
```

The same data is available in status field expressions, cross-CRD observation,
and autoscale conditions. One mechanism, many uses.

---

## 5. Requeue without explicit requeue logic

A natural question: if the `when:` gate skips the block, when does the reconciler
run again?

Two mechanisms handle this without any explicit requeue call.

**Watch events.** The informer watching the CR fires an event whenever any field
of the CR changes. The Deployment that Orkestra created has `orkestra-owner` as
a label and the CR as an owner reference. When the Deployment's `status.readyReplicas`
changes — which happens as pods become Ready — the Deployment's watch event fires.
Orkestra's informer factory routes this to the owning CR's queue. The CR is
requeued without any polling.

**Resync.** Every operatorbox has a configured `resync:` interval — typically
30s to 120s. At each resync, every known CR is re-enqueued regardless of whether
it has changed. This is the backstop: even if a watch event is missed or delayed,
the CR reconciles within one resync interval.

The combination means the operator never needs to specify `ctrl.Result{RequeueAfter: N}`.
The watch mechanism handles event-driven requeue. The resync handles everything else.

---

## 6. Multi-phase ordering without a state machine

The phase ordering in the traditional approach — create Deployment, wait for
readiness, create Service — is expressed in Orkestra as two resource groups
with a condition gate between them. No phase annotation. No switch statement.
No persistence of intermediate state.

The phase order falls out of the structure naturally:

```
onCreate:    runs once when the CR is created
onReconcile: runs on every reconcile, gated by when:
```

Resources declared in `onCreate` are created on the first reconcile cycle.
Resources declared in `onReconcile` are created on the cycle when the `when:`
condition first passes — and re-applied on every subsequent cycle where the
condition remains true, correcting any drift.

The Deployment is created first because it is in `onCreate`. The Service and
Ingress are created later because they are in `onReconcile` behind a `when:`
gate that waits for the Deployment. This is phase ordering expressed as
declaration, not as a state machine.

---

## 7. Idempotency by construction

Every resource group in Orkestra uses create-or-update semantics with
`reconcile: true`. If the resource already exists, it is updated to match the
template. If it does not exist, it is created. This is unconditionally
idempotent — the same reconcile call produces the same result regardless of
how many times it runs.

The `when:` gate does not need to be idempotent because it does not write state.
It reads `.children.*` and either executes the block or skips it. A gate that
evaluates to true on ten consecutive reconciles executes the same block ten
times, and each execution produces the same result because the block uses
create-or-update semantics.

This eliminates the idempotency burden that traditional operator authors carry
on every phase handler. The resource application is idempotent by construction.
The gate is stateless by construction.

---

## 8. Production validation

The async reconciliation model has been validated in production with 13,220
resources under management across three Katalogs. The pattern of create-in-onCreate,
wait-in-when, apply-in-onReconcile operates correctly under continuous load
with a 0.0% reconcile error rate.

The five-stage pipeline Katalog — one of the three in production — uses this
pattern across five ordered resource groups. The first group creates
infrastructure. Each subsequent group is gated by a `when:` condition that
checks the readiness of the preceding group's output. The entire pipeline,
which replaced over 400 lines of Go code, is expressed in approximately
60 lines of YAML. It runs without errors, without manual intervention, and
without any state machine beyond the declarative gates.

---

## 9. Conclusion

Async reconciliation in Orkestra is not a feature. It is the natural behavior
of the `when:` gate combined with the `.children.*` observation mechanism and
the Kubernetes watch infrastructure.

An operator author who writes `onCreate` and `onReconcile` with a `when:` gate
between them has written an async, multi-phase, event-driven operator workflow
without writing any async code. The reconciler returns immediately on every
call. The runtime handles timing. The watch mechanism handles notification.
The resync handles backstop requeue.

The four problems of traditional async operators — manual polling, hand-crafted
state machines, idempotency proofs, and requeue logic — do not exist in the
operatorbox model. They are replaced by two declarations and one condition.