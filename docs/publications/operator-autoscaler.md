# Operator Autoscaling: Runtime-Native Concurrency Control for OperatorBoxes

*Orkestra Project — April 2026*

---

## Abstract

Kubernetes workload autoscaling — HPA, KEDA, VPA — scales Pods in response to
metrics. Operators that process those Pods are assumed to be statically sized:
a fixed worker count declared at deployment time, unchanged until someone edits
a Deployment manifest. This assumption is wrong in practice. Operator load is
not constant. CRD event volume follows business rhythms — batch jobs at
business-hour start, deployments at CI/CD pipeline peak, compliance scans at
defined maintenance windows. A worker count appropriate for overnight baseline
is insufficient for morning peak, and a worker count appropriate for morning
peak wastes resources overnight.

Orkestra resolves this through runtime-native autoscaling declared inside the
operatorbox. Because each operatorbox owns its worker pool, its queue, and its
metrics in isolation, scaling one operatorbox has no effect on any other. The
declared CRD configuration is the permanent baseline. The autoscale block
declares conditions under which temporary overrides apply. When conditions are
no longer met, the baseline is restored automatically. No external autoscaler.
No revert block. No hidden state.

---

## 1. Why operator autoscaling is structurally different

Scaling a Pod workload means adding instances. Each instance is stateless with
respect to the others — requests are distributed across them and any instance
can handle any request. The scaling unit is the Pod, and the coordination
mechanism is a load balancer or service mesh.

Scaling an operator is structurally different. An operator is not stateless
with respect to its queue. The queue is the source of truth for what work
remains. Multiple operator instances coordinate through leader election — only
the leader processes events, followers maintain warm caches and wait. Adding
operator Pods does not increase processing throughput; only the leader processes
work, and only the leader's worker pool size matters.

The correct scaling unit for an operator is therefore the worker pool within
the single active instance. More workers means more goroutines pulling from
the queue concurrently. This is an intra-process scaling operation, not an
inter-Pod scaling operation.

The second difference is the queue depth. Kubernetes workqueue implementations
have no native backpressure beyond rate limiting. Under sustained high load,
the queue grows unboundedly until memory is exhausted or the operator is
restarted. Controlling queue depth is a runtime concern that the autoscaler
addresses as a companion to worker count.

The third difference is the resync interval. The resync interval controls how
frequently Orkestra re-enqueues all known CRs for reconciliation, regardless
of whether they have changed. A shorter resync interval increases throughput
and reduces the time to detect and correct drift — at the cost of higher CPU
and API server load. During maintenance windows or compliance scan cycles,
a shorter resync interval is appropriate. During off-hours, a longer interval
reduces unnecessary load.

All three parameters are declared in the operatorbox and all three are under
the autoscaler's control.

---

## 2. The baseline principle

The autoscaler's most important architectural property is that it has no
persistent state of its own. The CRD's declared workers, queueDepth, and
resync are the baseline. The autoscaler applies overrides when conditions are
met and restores the baseline when they are not.

This eliminates the configuration drift problem that afflicts traditional
autoscalers. With HPA, the current replica count is stored in the HPA resource
itself and can diverge from the Deployment's declared `replicas` field. With
KEDA, the scaling configuration is separate from the workload configuration
and must be kept synchronized. With Orkestra's autoscaler, there is no
separate scaling state. The baseline is always the Katalog. Overrides are
always temporary. A restart always begins from the declared configuration.

The consequence is that "what is this operator supposed to do?" is always
answerable by reading the Katalog, without consulting a separate resource or
querying runtime state.

---

## 3. Condition model

The autoscaler uses the same condition primitives as the rest of the Katalog —
`when:` for AND semantics and `anyOf:` for OR semantics — extended with
time-aware condition types.

**Metric conditions.** Runtime metrics are exposed under the `metrics.*`
namespace and evaluated by the condition resolver without informer or API
server involvement:

```yaml
when:
  - field: metrics.workersBusyPercent
    greaterThan: "80"
  - field: metrics.queueDepth
    greaterThan: "200"
```

**Clock conditions.** Active when the current time is after a threshold:

```yaml
anyOf:
  - time:
      after: "08:00"
      before: "17:00"
```

**Day-of-week conditions.** Active on specific days:

```yaml
anyOf:
  - dayOfWeek:
      in: ["Saturday", "Sunday"]
```

**Cron conditions with duration.** A cron expression defines when a time window
opens. The `duration:` field defines how long it remains open. Without
`duration:`, the window closes after one autoscale interval — which is almost
certainly not the intended behavior:

```yaml
anyOf:
  - cron: "0 8 * * 1-5"
    duration: 9h   # active from 08:00 to 17:00 on weekdays
```

The combined condition is: `anyOf` conditions are OR-evaluated, `when:`
conditions are AND-evaluated, and both blocks must pass when both are declared.
This matches the semantics of every other condition block in Orkestra.

---

## 4. Cooldown

Without cooldown, a queue depth that oscillates around the threshold causes the
autoscaler to apply and revert on alternating evaluation ticks. This is not a
theoretical concern — workqueue depth fluctuates on the order of seconds during
active reconcile bursts. The cooldown period defines the minimum time that
conditions must be continuously false before a revert is applied:

```yaml
autoscale:
  interval: 15s
  cooldown: 2m    # conditions must be false for 2 minutes before reverting
```

The cooldown applies only to the revert direction. Override application is
immediate when conditions become true. This asymmetry is intentional: scaling
up under load should be fast, scaling down under reduced load should be
conservative.

---

## 5. How each override is implemented

**Workers — resizable semaphore.**
The worker pool is not a fixed goroutine count. It is a weighted semaphore that
gates how many goroutines may enter `Reconcile` simultaneously. All goroutines
run continuously; the semaphore controls concurrent access.

Goroutines are over-provisioned at startup to `max(baseline.workers,
do.workers)` so that scale-up never requires spawning new goroutines at runtime.
Increasing the semaphore weight immediately allows more goroutines to proceed.
Decreasing the weight causes excess goroutines to block after their current
reconcile completes — in-flight work is never interrupted.

The implementation is a custom `ResizableSemaphore` rather than
`golang.org/x/sync/semaphore.Weighted`, which does not support weight
modification after construction.

**Queue depth — atomic limit in Enqueue.**
The per-CRD `Workqueue` holds an atomic `int32` limit. Every call to `Enqueue`
reads the limit and compares it to the current queue length. If the queue is at
or beyond the limit, the item is dropped with a warning log (GVK, key, current
depth, limit). Items already in the queue are not evicted — the limit applies
only to incoming enqueues. The limit is 0 (unlimited) by default and is
restored to 0 when the baseline is reinstated, restoring unlimited throughput.

**Resync interval — independent re-enqueue goroutine.**
Changing the informer's built-in resync period post-construction is not
supported by the Kubernetes client-go API. Orkestra implements resync override
as an independent goroutine that, while the override is active, reads all
objects from the informer's local cache and re-enqueues them at the declared
interval. The informer's own resync continues at its baseline rate in parallel;
the workqueue deduplicates keys so concurrent enqueues are safe and the only
effect is a higher reconcile frequency. When the override reverts, the goroutine
idles (interval set to 0) and the informer's resync period is again the sole
driver.

---

## 6. Properties

The autoscaler is designed around six properties that distinguish it from
external autoscaling solutions:

**Declarative.** Scaling behavior is expressed as data in the Katalog, not
as code in a controller or as configuration in a separate resource. It can
be reviewed, versioned, and audited alongside the operator behavior it modifies.

**Runtime-native.** The autoscaler reads metrics directly from the operatorbox
runtime. No Prometheus scraping. No metrics server. No external metric adapter.
The metrics are always current, always accurate, and never require a separate
data pipeline.

**Isolated.** The autoscaler for one operatorbox has no knowledge of and no
effect on any other operatorbox. A scaling event in the website operator does
not affect the pipeline operator's workers or queue.

**Reversible.** The CRD configuration is the permanent baseline. Overrides are
temporary. The autoscaler always converges toward either the override (when
conditions are met) or the baseline (when they are not). There is no state
that can drift from either.

**Predictable.** Evaluation is level-triggered on a fixed interval. The
autoscaler does not respond to events; it evaluates conditions on a tick and
applies the result. The same conditions always produce the same result. The
cooldown period is the only temporal dependency.

**Minimal.** No external controller. No separate CRD. No Helm chart for the
autoscaler itself. It is a feature of the operatorbox, declared inside it,
running inside the same process as the operator it scales.