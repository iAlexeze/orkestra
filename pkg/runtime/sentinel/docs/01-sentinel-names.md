# Sentinel Names

Each sentinel is a named boolean computed at `UpdateFunc` time. The result is a string `"true"` or `"false"` stored in `QueueItem.SentinelMap` so it survives the queue.

---

## `generationChanged`

```go
old.GetGeneration() != new.GetGeneration()
```

The Kubernetes API server increments `metadata.generation` whenever the object's `spec` changes (for resources that track this). A `metadata.labels` or `metadata.annotations` change does not increment it.

**Use for**: skipping reconciliation when only non-spec fields changed — annotations written by the reconciler itself, status updates, or label patches that do not affect the desired state.

```yaml
preReconcile:
  sentinels: [generationChanged]
  enqueueGate:
    sentinels: [generationChanged]
```

**Note**: not all resources increment `generation`. Built-in types like `ConfigMap` and `Secret` do not. For those resources, `generationChanged` is always `"false"` on update events.

---

## `labelsChanged`

```go
!reflect.DeepEqual(old.GetLabels(), new.GetLabels())
```

True when the label map differs — any key added, removed, or changed in value.

**Use for**: re-running label-driven logic when a user re-labels a CR mid-lifecycle, or when a controller patches labels on a child resource you watch.

---

## `annotationsChanged`

```go
!reflect.DeepEqual(old.GetAnnotations(), new.GetAnnotations())
```

True when the annotation map differs.

**Use for**: annotations used as side-channel signals — e.g. a deployment tool writing a `deploy-timestamp` annotation that should trigger a reconcile.

Be careful: if your reconciler writes annotations back to the CR, every reconcile produces an update event. Without an additional guard (such as gating on `generationChanged` as well), this can produce a reconcile loop.

---

## `deletionStarted`

```go
old.GetDeletionTimestamp() == nil && new.GetDeletionTimestamp() != nil
```

True exactly once: the first event after a `kubectl delete` reaches the API server and `DeletionTimestamp` is set. By the time the object is dequeued for reconciliation, `DeletionTimestamp` is already non-nil — but this sentinel captures the transition at event time.

**Use for**: immediate enqueue of deletion logic without waiting for the normal reconcile cycle, or as a gate to short-circuit expensive reconcile work when the object is already terminating.

---

## `finalizersChanged`

```go
!reflect.DeepEqual(old.GetFinalizers(), new.GetFinalizers())
```

True when the finalizer list differs — any finalizer added or removed.

**Use for**: detecting when a finalizer has been removed externally (e.g. by a user force-removing it) so the reconciler can react before the object disappears.

---

## Combining sentinels in a gate

`enqueueGate.sentinels` is a list. The gate fires when **any** listed sentinel is `"true"`:

```yaml
enqueueGate:
  sentinels: [generationChanged, labelsChanged]
```

To require both, use a template expression in `enqueueGate.when` instead — sentinels are available as template variables in that context.

→ [02-compute.md](02-compute.md) — how sentinels flow from the informer event to the gate
