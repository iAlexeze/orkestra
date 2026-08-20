# sentinel — event-time gate values

Sentinels are booleans computed at informer `UpdateFunc` time by comparing `oldObj` and `newObj`. They are carried through the queue so that `enqueueGate` and `reconcileGate` can both ask "did the generation change?" without `oldObj` being available at dequeue time.

> [!IMPORTANT]
> This package is the canonical home for sentinel names. It imports only stdlib and `k8s.io/apimachinery` so it can be imported by `pkg/types` (for YAML constants) and `pkg/runtime/informer` (for `Compute`) without creating an import cycle. Do not add dependencies outside those two — move computation to the caller instead.

> [!TIP]
> **A sentinel is not a Note.** Notes carry arbitrary values into the reconcile context and are available everywhere in templates — `onCreate`, `onReconcile`, `status.fields`, `normalize`, rules. A sentinel answers a single yes/no question about what changed between two versions of the object, and is only available in gate conditions. Use a Note when you need a value inside reconciliation; use a sentinel when you need to decide whether reconciliation should run at all. → [`pkg/note`](../../note/README.md)

---

## YAML usage

Declare which sentinels to compute on `preReconcile.sentinels`. They are then available as gate conditions on `enqueueGate` and `reconcileGate`.

```yaml
operatorBox:
  preReconcile:
    sentinels: [generationChanged, labelsChanged]
    enqueueGate:
      sentinels: [generationChanged]   # skip the queue unless spec changed
```

On secondary watches, `enqueueGate.sentinels` works the same way:

```yaml
operatorBox:
  watch:
    - apiVersion: apps/v1
      kind: Deployment
      enqueueGate:
        sentinels: [generationChanged]
```

---

## Built-in sentinels

| Name | True when |
|------|-----------|
| `generationChanged` | `old.generation != new.generation` — spec change |
| `labelsChanged` | label map differs between old and new |
| `annotationsChanged` | annotation map differs between old and new |
| `deletionStarted` | `DeletionTimestamp` transitions from nil to non-nil |
| `finalizersChanged` | finalizer list differs between old and new |

---

## Documents

| File | What it covers |
|------|----------------|
| [01-sentinel-names.md](docs/01-sentinel-names.md) | Each sentinel in detail — what it tests, when to use it, common patterns |
| [02-compute.md](docs/02-compute.md) | How `Compute` runs at event time, what `QueueItem.SentinelMap` carries, and how gates read it |
| [03-adding-a-sentinel.md](docs/03-adding-a-sentinel.md) | Step-by-step: constant, registration, computation, tests, documentation |

Read [01-sentinel-names.md](docs/01-sentinel-names.md) when choosing which sentinels to declare for a gate. Read [02-compute.md](docs/02-compute.md) when tracing how a sentinel value flows from the informer event to a gate evaluation. Read [03-adding-a-sentinel.md](docs/03-adding-a-sentinel.md) when extending the sentinel set.
