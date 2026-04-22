# Fault Tolerance

Every operatorbox in Orkestra runs inside a panic boundary. A panic anywhere
inside an operatorbox — user constructor, hook, template renderer, cross-CRD
read, or Orkestra's own internal code — is caught, logged with a full stack
trace, emitted as a Warning event on the affected CR, and returned to the
workqueue as an error. The operatorbox retries with exponential backoff. The
process does not crash. Other operatorboxes are unaffected.

This is not a feature layered on top of the execution model. It is a direct
consequence of it. Each CRD has its own worker pool. Each reconcile executes
inside `safeReconcile`. The panic domain is the operatorbox boundary.

---

## What happens when a panic occurs

```
1. safeReconcile catches the panic
2. Full stack trace logged at ERROR with CRD and CR name
3. Warning event emitted on the affected CR
4. CR requeued with exponential backoff
5. CRD marked degraded in health state
6. Katalog marked degraded
7. All other operatorboxes continue reconciling normally
8. Runtime stays green
```

Nothing else is affected. No other CRD loses its workers. No queue is
abandoned. No cache goes stale. The control plane does not restart.

---

## Failure containment table

| Failure | Contained in | Effect | Everything else |
|---|---|---|---|
| Panic in constructor | OperatorBox | CR fails, CRD degrades | Unaffected |
| Panic in hook | OperatorBox | CR fails, CRD degrades | Unaffected |
| Panic in template renderer | OperatorBox | CR fails, CRD degrades | Unaffected |
| Panic in cross-CRD read | OperatorBox | CR fails, CRD degrades | Unaffected |
| Panic in Orkestra internal code | OperatorBox | CR fails, CRD degrades | Unaffected |
| Consecutive failures | OperatorBox | CRD degraded, retrying | Unaffected |

The operatorbox is the fault boundary. What is inside it stays inside it.

---

## Observability during failure

The Control Center surfaces degradation without any action from the operator author:

- The affected CRD panel shows **Degraded**
- The consecutive failure count is visible: `2 consecutive failures`
- The last error is shown verbatim: `reconciler panic: assignment to entry in nil map`
- The Katalog panel reflects the degraded state of any operatorbox it produced
- The runtime panel stays **Green** — the Kordinator, informer factory, and
  admission layer are unaffected

The failure is visible, actionable, and contained. No digging through pod logs
to understand which CRD is failing. No guessing whether other CRDs are affected.

---

## Validated in production

During development, a real bug in Orkestra's internal service builder triggered
`assignment to entry in nil map` inside the `MultiRegionApp` operatorbox. This
was not user code. It was a panic inside Orkestra's own reconcile path.

The result:

- `MultiRegionApp` operatorbox degraded and began retrying with backoff
- `Website` operatorbox continued reconciling with zero interruption
- The runtime remained fully operational
- The error appeared immediately in the Control Center
- No process restart. No cross-CRD contamination. No data loss.

This validated the operatorbox model under real, unsimulated failure conditions.
The isolation held.

---

## Why traditional controllers fail differently

In a standard `controller-runtime` Manager, all controllers share one process
and one goroutine group. A panic in any controller that is not explicitly
recovered crashes the Manager. All CRDs stop reconciling. All queues are
abandoned. All caches become stale. Recovery requires a pod restart, which
means a cache cold-start and a burst of reconciles across every CRD in the
cluster.

Orkestra's model does not have this failure mode. There is no shared panic
domain between operatorboxes. A panic in one is a local event — caught,
recorded, and retried — not a global one.

This is the same resilience model used by Erlang supervision trees and Akka
actors: isolate failures at the process boundary, recover locally, and keep
the rest of the system running. Orkestra applies that model to Kubernetes
operators.