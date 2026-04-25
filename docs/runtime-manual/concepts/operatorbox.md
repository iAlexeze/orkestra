# operatorBox:

An **operatorBox:** is the fundamental execution unit in Orkestra. Each CRD entry
in a Katalog produces one operatorBox. The operatorBox: is what makes Orkestra a
runtime rather than a framework: it owns the complete lifecycle of one CRD, in
isolation from every other CRD in the same process.

---

## What an operatorBox: owns

Every operatorBox: owns exclusively:

**Informer** — A `SharedIndexInformer` watching exactly the resources declared
by this CRD entry. The in-memory cache holds all CR instances. Reads from this
cache are zero-cost.

**Event queue** — A bounded workqueue receiving events from the informer.
Queue depth and rate limiting are configured per CRD via `workers:` and queue
settings.

**Worker pool** — A fixed number of goroutines pulling from the queue. Configured
via `workers:` in the Katalog. Two workers for low-volume CRDs, ten or more for
high-throughput ones.

**Reconciler** — A `GenericReconciler[T]` instance holding the full CRD entry
configuration: normalize rules, mutation rules, validation rules, template
declarations, provider registry access, and cross-CRD informer references.

**Health state** — A `CRDHealth` instance tracking success rate, consecutive
failures, and degraded threshold. Visible in the Control Center per operatorBox.

**Metrics** — Reconcile duration, error rate, queue depth, worker utilization —
all labeled by GVK and scoped to this operatorBox.

**Lifecycle** — Start and stop independently. Start in dependency order. Drain
queue on shutdown.

---

## Declaring an operatorBox:

In the Katalog, the `operatorBox:` block declares the complete behavior of one
CRD's isolated runtime:

```yaml
spec:
  crds:
    pipeline:
      apiTypes:
        group: platform.io
        version: v1alpha1
        kind: Pipeline
        plural: pipelines

      workers: 10
      resync: 30s
      dependsOn:
        database: healthy

      normalize:
        spec:
          schedule: "{{ ... }}"

      mutation:
        rules: [...]

      validation:
        rules: [...]

      operatorBox:
        default: true          # use Orkestra's GenericReconciler
        cross: [...]           # IPC declarations
        onCreate: [...]        # resource creation on first reconcile
        onReconcile: [...]     # drift correction on every reconcile
        onDelete: [...]        # cleanup before finalizer removal
        status: [...]          # status field declarations
        providers: [...]       # external infra (AWS, MongoDB, etc.)
```

The `default: true` flag tells Orkestra to use the declarative reconciler. No
Go code is required. Set `default: false` and provide a `constructor:` when
you need a typed Go reconciler.

---

## Reconcile pipeline

When a queue item is dequeued, the operatorBox: reconcile pipeline runs in this
order:

```
informer cache → DeepCopy → normalize → mutation → validation
    → OnReconcile hook (Go) or runTemplateReconcile (declarative)
        → cross-CRD observation  (.cross.*)
        → external HTTP calls    (.external.*)
        → forEach expansion      (list field: .item=element | map field: .item=key, .value=value)
        → onCreate resource groups
        → onReconcile resource groups
        → provider dispatch
    → patchStatusWithChildren
```

Each step receives the output of the previous step. Normalize produces the
canonical spec. Mutation applies defaults. Validation enforces constraints.
Template rendering operates on the normalized, mutated, validated spec.

---

## Isolation guarantees

operatorBox:es within the same binary do not share:

- Informer caches
- Workqueues
- Reconciler instances
- Health state
- Panic domains (each reconcile is wrapped in `safeReconcile`)

A panic in one operatorBox: is recorded as a reconcile failure and triggers a
requeue. It does not affect any other operatorBox. Queue pressure in one
operatorBox: does not affect processing latency in another.

The isolation is within a single OS process. operatorBox:es share the Go
runtime scheduler and memory allocator. They do not share any Orkestra-level
data structures.

---

## Orkestra IPC — cross-operatorBox: communication

An operatorBox: can observe another operatorBox:'s CR state through the `cross:`
declaration. This is explicit, read-only, and zero-cost for same-binary
operatorBox:es.

```yaml
operatorBox:
  cross:
    - crd: managed-database
      selector:
        name: "{{ .metadata.name }}-db"
      as: db
  onReconcile:
    deployments:
      - name: "{{ .metadata.name }}"
        when:
          - field: "{{ phase .cross.db }}"
            equals: "Ready"
```

The `cross:` declaration resolves through the `KatalogRegistry`, which holds
a reference to every operatorBox:'s informer. Reading another operatorBox:'s
state is an in-memory map lookup — the API server is not involved.

For cross-binary or cross-cluster observation, declare `source.endpoint` in
the `cross:` entry to fall back to an HTTP call.

---

## Startup sequencing

operatorBox:es start in the topological order defined by their `dependsOn`
declarations. The dependency graph is validated at Katalog load time — cycles
are fatal.

```yaml
database-backed-app:
  dependsOn:
    managed-database: healthy   # wait until managed-database has reconciled successfully
```

Supported conditions: `started` (workers running), `healthy` (running with zero
consecutive failures). The `DependencyKordinator` resolves the graph and starts
each operatorBox: when its declared condition is met.

---

## Relationship to the Katalog

A Katalog is a collection of operatorBox: declarations. The Katalog is the schema;
the operatorBox: is the runtime instance of that schema. One Katalog can declare
many operatorBox:es. Multiple Katalogs can run in one Orkestra instance. Each
operatorBox: is isolated regardless of which Katalog declared it.

The Control Center shows one panel per operatorBox: health state, reconcile
rate, queue depth, worker utilization, and the last error.
