# OperatorBox

_CRDs in. Operators out._

You write a CRD declaration. Orkestra wraps it in an operatorBox and runs it as a fully isolated operator — its own informer, its own queue, its own worker pool, its own health state. No Go code required unless you want it.

Think of it the way you think of containers. A container takes an image and runs it in isolation from everything else on the host. An operatorBox takes a CRD declaration and runs it in isolation from every other CRD in the same process. Panics don't escape. Queue pressure doesn't spill. Health state is tracked independently.

The operatorBox is what makes Orkestra a **runtime** rather than a framework.

---

## What an operatorBox owns

Every operatorBox owns exclusively:

**Informer** — A `SharedIndexInformer` watching exactly the resources declared by this CRD entry. The in-memory cache holds all CR instances. Reads are zero-cost.

**Event queue** — A bounded workqueue receiving events from the informer. Queue depth and rate limiting are configured per CRD via `workers:` and queue settings.

**Worker pool** — A fixed number of goroutines pulling from the queue. Configured via `workers:` in the Katalog.

**Reconciler** — A `GenericReconciler[T]` holding the full CRD entry configuration: normalize rules, mutation rules, validation rules, template declarations, provider registry access, and cross-CRD informer references.

**Health state** — A `CRDHealth` instance tracking success rate, consecutive failures, and degraded threshold. Visible in the Control Center per operatorBox.

**Metrics** — Reconcile duration, error rate, queue depth, worker utilization — all labeled by GVK and scoped to this operatorBox.

**Lifecycle** — Starts and stops independently. Starts in dependency order. Drains queue on shutdown.

---

## Declaring an operatorBox

In the Katalog, the `operatorBox:` block declares the complete behavior of one CRD's isolated runtime:

```yaml
spec:
  crds:
    pipeline:
      crdFile: my-pipeline.yaml
      workers: 10
      resync: 30s
      dependsOn:
        database: healthy

      normalize:
        spec:
          schedule: "{{ ... }}"

      operatorBox:
        default: true          # use Orkestra's GenericReconciler
        cross: [...]           # IPC declarations
        onCreate: [...]        # resource creation on first reconcile
        onReconcile: [...]     # drift correction on every reconcile
        onDelete: [...]        # cleanup before finalizer removal
        status: [...]          # status field declarations
        providers: [...]       # external infra (AWS, MongoDB, etc.)
```

`default: true` tells Orkestra to use the declarative reconciler — no Go code required. Set `default: false` with a `constructor:` when you need a typed Go reconciler.

---

## Where to go next

- [Reconcile Pipeline](reconcile-pipeline/) — the ordered steps from queue to status patch
- [Isolation and IPC](isolation/) — how isolation is enforced and how operatorBoxes communicate
- [Startup Sequencing](startup-sequencing/) — dependency order and the Katalog relationship
