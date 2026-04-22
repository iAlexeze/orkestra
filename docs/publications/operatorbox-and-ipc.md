# The operatorBox: Model: Isolated Runtime Cells and Declarative IPC

*Orkestra Project — April 2026*

---

## Abstract

Kubernetes operators have been built as monolithic controllers sharing process
memory, global informers, and implicit cluster-wide caches. This model produces
correct operators in isolation but fragile systems in composition. When twenty
operators share a cluster, they share nothing explicitly and everything
accidentally. Orkestra introduces a different execution model: the
**operatorBox:**, an isolated runtime cell that gives each CRD its own informer,
queue, worker pool, health state, and reconciliation context. Operators do not
share internal state. They communicate only through **Orkestra IPC** — explicit,
declarative, read-only cross-operator observation. This paper defines the
operatorBox: model precisely, describes its properties, and demonstrates how
declarative IPC enables safe multi-operator composition without sacrificing
determinism.

---

## 1. The shared-runtime problem

Every major operator framework today runs all operators in a shared process.
Controller-runtime, the foundation of both Kubebuilder and Operator SDK, manages
multiple controllers inside a single `Manager`. The Manager holds one
`client.Client`, one scheme, one field indexer, and a shared cache. Individual
controllers register watches against this shared cache. Events are dispatched
to the appropriate `Reconcile` method.

This architecture works correctly for operators built together by the same team,
deployed together as a single binary, with full knowledge of each other's behavior.
It breaks down in any other configuration.

The failure modes are specific. A controller that enqueues items faster than its
workers can process them will cause the shared workqueue to grow unboundedly,
eventually starving other controllers of processing time. A reconciler that panics
without recovery — not uncommon in early-stage operators — crashes the entire
Manager. A controller that issues expensive List operations with broad label
selectors loads the shared cache with data that all other controllers must
also hold in memory, whether they need it or not. When two teams develop operators
independently and deploy them to the same cluster, they have no isolation
guarantees between them.

The root cause is not any framework implementation detail. It is the assumption
that sharing is efficient and isolation is expensive. For Kubernetes operators
running at typical platform scale — tens of CRDs, hundreds of thousands of
resources, multiple teams — this assumption does not hold.

---

## 2. The operatorBox:

An operatorBox: is Orkestra's fundamental execution unit. One CRD entry in the
Katalog produces one operatorBox. The operatorBox: owns exclusively:

**Its own informer.** A `SharedIndexInformer` watching exactly the resources
declared by this CRD entry. For a `Pipeline` CRD, the informer watches
`pipelines.platform.io/v1alpha1`. For a `ManagedDatabase` CRD, the informer
watches `manageddatabases.platform.io/v1alpha1`. Neither informer knows the
other exists.

**Its own event queue.** A bounded workqueue receiving events from its informer.
Queue depth, rate limiting, and max queue size are configured per CRD. A
`Pipeline` operatorBox: with high throughput requirements can be configured
with a larger queue and more workers without affecting the `ManagedDatabase`
operatorBox: at all.

**Its own worker pool.** A fixed number of goroutines pulling from the queue
and calling into the reconcile loop. Workers is a CRD-level configuration.
Ten workers for high-volume CRDs, two for low-volume ones.

**Its own reconciler instance.** A `GenericReconciler[T]` constructed
specifically for this CRD. It holds the CRD entry configuration, the
`ReconcilerConfig` with all declared behaviors, the provider registry access,
and the katalog registry reference for cross-CRD informer lookup.

**Its own health state.** A `CRDHealth` instance tracking consecutive successes,
consecutive failures, error rate, and degraded threshold. Health transitions
for one CRD do not affect the health state of any other CRD. The Control Center
shows health per operatorBox.

**Its own metrics.** Reconcile duration, error rate, queue depth, worker
utilization — all labeled by GVK, all scoped to this operatorBox.

**Its own lifecycle.** operatorBox:es start in topological dependency order.
If a `DatabaseBackedApp` CRD declares `dependsOn: managed-database: healthy`,
its workers do not start until `managed-database` has reconciled at least one
resource successfully. This is startup sequencing without coordination code.

---

## 3. What operatorBox:es do not share

The isolation properties are equally important to state precisely. operatorBox:es
do not share:

**Informer caches.** Each operatorBox: maintains its own in-memory cache of its
own resources. There is no global cache. There is no coordination required to
read from the cache of another operatorBox: — and no accidental reading of data
from another operatorBox:'s cache.

**Workqueues.** Queue pressure in one operatorBox: does not affect processing
latency in another. The 13,100 queued items observed in production during a
high-load test in the `website` operatorBox: did not affect the `pipeline`
operatorBox:'s latency at all. This is a structural property, not an operational
tuning outcome.

**Reconciler state.** No reconciler holds references to another reconciler's
internal state. The Go type system enforces this — each `GenericReconciler[T]`
is parameterized by its own domain type and holds its own configuration.

**Panic domains.** Each reconcile invocation is wrapped in `safeReconcile`, which
recovers from panics and records them as reconcile failures. A panic in one
operatorBox: increments that operatorBox:'s failure counter and triggers a requeue.
It does not crash the process. It does not affect any other operatorBox.

The isolation is within a single binary. operatorBox:es share an OS process and
runtime memory allocator. The isolation is at the data structure level — informer
caches, queues, reconcilers, and health state are not shared. This is comparable
to goroutine isolation: stronger than a monolithic function, weaker than an OS
process. For the correctness properties that matter for operator composition —
queue independence, cache independence, health isolation, panic containment — the
operatorBox: model provides the necessary guarantees.

---

## 4. Orkestra IPC: explicit cross-operator communication

Isolating operatorBox:es creates a secondary problem: operators that depend on
each other's state can no longer read it directly. A `DatabaseBackedApp`
operatorBox: needs to know whether the `ManagedDatabase` CR for its database is
Ready before creating its Deployment. In a shared-cache model, it calls
`client.Get` against the shared cache. In the operatorBox: model, there is no
shared cache to call against.

Orkestra solves this with explicit IPC: the `cross:` declaration.

```yaml
spec:
  crds:
    database-backed-app:
      operatorBox:
        cross:
          - crd: managed-database
            selector:
              name: "{{ .metadata.name }}-db"
              namespace: "{{ .metadata.namespace }}"
            as: db
        onReconcile:
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              when:
                - field: cross.db.status.phase
                  equals: "Ready"
```

The `cross:` declaration is explicit and visible. It names the source CRD
(`crd: managed-database`), the selector that identifies the specific CR
instance, and the name under which the result is available in the template
context (`as: db`). Nothing about this communication is implicit. A reader of
the Katalog can immediately see which external CRD this operatorBox: depends on
and what field it uses to gate its behavior.

The read is zero-API-calls for same-binary operatorBox:es. The `cross:` mechanism
resolves through the `KatalogRegistry`, which holds a reference to every
operatorBox:'s informer. Reading `managed-database`'s informer cache from the
`database-backed-app` reconciler is an in-memory map lookup. The API server is
not involved. This is what makes the model practical: cross-CRD observation is
as cheap as same-CRD observation.

For cross-binary or cross-cluster IPC, the `source.endpoint` field falls back
to Orkestra's own CR detail endpoint — the same endpoint the Control Center
uses. The HTTP path is authenticated and cached.

---

## 5. IPC semantics

The cross-operator communication model has specific semantics that distinguish
it from the informal coupling patterns in traditional operators.

**Read-only.** An operatorBox: can observe another operatorBox:'s resource state.
It cannot modify it. This eliminates an entire class of multi-operator bugs where
two operators attempt to reconcile the same resource field in opposite directions.

**Declared at load time.** The `cross:` dependencies are declared in the Katalog
and validated at startup. If a `cross:` declaration references a CRD kind not
present in the Katalog, Orkestra logs a warning and returns an empty result —
the dependent operatorBox: is not blocked, it simply proceeds with
`cross.db.found = "false"`.

**Level-triggered.** Cross-CRD reads happen on every reconcile of the depending
CRD. If the `managed-database` CR's status changes, the `database-backed-app`
operatorBox: observes the change on its next resync cycle — no push notification
is required. The reconcile loop is the notification mechanism.

**Not a subscription.** An operatorBox: does not subscribe to events from another
operatorBox. It reads state at reconcile time. This means there is no event
ordering dependency between operatorBox:es. The `database-backed-app` operatorBox:
will simply retry until the `managed-database` CR reaches the expected state.

---

## 6. Dependency graphs at startup

The `dependsOn` block extends the IPC model to startup sequencing.

```yaml
database-backed-app:
  dependsOn:
    managed-database: healthy
```

This declares that the `database-backed-app` operatorBox: should not start
processing until the `managed-database` operatorBox: has reached `healthy` state
— at least one successful reconcile with zero consecutive failures. Orkestra
resolves the full dependency graph at startup and starts operatorBox:es in
topological order, waiting at each node until its declared condition is met.

The dependency graph is validated at Katalog load time. Cycles are detected
and produce a fatal error with a clear message. This is the operatorBox: model
applied to time: just as cross: provides spatial isolation with explicit reads,
dependsOn provides temporal isolation with explicit sequencing.

---

## 7. Comparison to existing communication patterns

Traditional operators communicate across CRD boundaries in three patterns, none
of which are explicit.

**Direct Get calls.** A reconciler calls `client.Get` with the name of a resource
from a different CRD. This works but creates an invisible dependency. No part of
the operator's declaration indicates this dependency exists. Code review is the
only mechanism for discovering it. When the target CRD changes its status fields,
the depending operator breaks silently until the breakage is observed in production.

**Shared label-selector watches.** An operator registers a watch on a different
CRD's resources by label. Events from those resources enqueue items in this
operator's queue. This couples the operators at the event level, not just the
state level. Bugs in the event production of one operator affect the processing
latency of another.

**Status conventions.** Operators document informal conventions — "my operator
sets `status.ready = true` when healthy; other operators can read this." These
conventions exist in READMEs, not in machine-checkable declarations. They are
not visible in the Katalog, not validated at startup, and not surfaced in the
Control Center.

Orkestra IPC replaces all three patterns with one mechanism: `cross:`. The
dependency is declared, validated, observable in the Control Center, and
implemented through an architecture that provides the same zero-cost reads
as a direct cache lookup.

---

## 8. Multi-operator composition

The operatorBox: model makes multi-operator composition predictable for the first
time. A platform team can define a Katalog with a dependency graph:

```
network → namespace → database → application
```

Each arrow is a `dependsOn` relationship. Each node is an operatorBox. The
platform team can add a new operatorBox: to the Katalog — declaring its
dependencies and its IPC relationships — without modifying any existing
operatorBox. The new operatorBox: does not share memory with any existing
operatorBox. It cannot accidentally read stale data from a different CRD's
informer. It cannot cause another operatorBox: to process events it didn't intend
to process.

In production, Orkestra has operated thirteen operatorBox:es simultaneously across
three Katalogs, processing 24,060 reconciles with a 0.0% error rate. The
operatorBox:es managing different CRDs have different worker counts, different
queue depths, and different health profiles. None of their operational
characteristics affect each other. This is the correct baseline behavior for
a system built on the operatorBox: model.

---

## 9. Implications

The operatorBox: model has implications beyond the correctness properties already
described.

**Operator marketplaces become safe.** When operators are isolated by design,
installing a third-party operator from a registry does not create a risk that
it will interfere with existing operators. The worst it can do is fail to
reconcile its own resources.

**Multi-team development becomes tractable.** Two teams can develop operatorBox:es
independently, declare their IPC relationships explicitly, and deploy them to the
same Katalog without coordination. The explicit `cross:` declarations serve as the
interface contract between teams.

**Observability becomes per-operator.** The Control Center shows one health panel
per operatorBox. Degradation in one CRD is visible as degradation in exactly that
CRD, not as degradation in "the operator." Root cause analysis narrows to a
single operatorBox.

**Lifecycle becomes per-operator.** An operatorBox: can be disabled without
affecting others. It can be restarted by reapplying its Katalog entry. Its
workers can be scaled independently. These operations do not require rebuilding
or redeploying the entire operator binary.

---

## 10. Conclusion

The operatorBox: model reconstitutes the operator pattern on a different
architectural foundation. Rather than multiple controllers sharing a runtime,
each CRD gets an isolated runtime cell: its own informer, queue, workers,
health state, and reconciler. Communication between operatorBox:es is explicit,
declarative, read-only, and zero-cost for same-binary reads. Startup sequencing
is declared through a validated dependency graph.

This is not an incremental improvement to the shared-runtime model. It is a
different model, with different correctness properties and different composition
guarantees. The traditional operator model treats CRDs as API types managed by
a shared controller. Orkestra treats each CRD as its own operator — isolated,
self-contained, and explicitly connected to its dependencies.

Operators are no longer controllers in a shared binary. They are processes in
a shared runtime. The distinction is architectural, and it is what makes safe
multi-operator composition possible.
