# GenericReconciler

`pkg/reconciler.GenericReconciler[T]` is the per-CRD reconcile loop. Every CRD managed by Orkestra gets exactly one `GenericReconciler` instance. It handles the complete CR lifecycle: creation, drift correction, finalizer management, deletion, events, metrics, and health reporting.

The `GenericReconciler` is generic over `T domain.Object` — it works with both `*unstructured.Unstructured` (dynamic mode) and concrete typed objects (typed mode). The same code path handles both.

---

## Reconcile dispatch order

On every reconcile event, the `GenericReconciler` executes in this order:

```
1. Ensure managed label    — idempotent, sets orkestra.konductor.io/managed=true
2. Finalizer add           — on new CRs, adds declared finalizers
3. Deletion check          — if DeletionTimestamp is set, run onDelete + remove finalizers
4. MutateFirst (optional)  — apply defaults before validation (if mutateFirst: true)
5. Validation              — evaluate validation rules (deny/warn)
6. Mutation (default)      — apply defaults after validation
7. Reconcile implementation:
     a. Go hooks (OnReconcile)         — priority 1
     b. Declarative templates          — priority 2 (runTemplateReconcile)
     c. No-op                          — finalizers/events/metrics only
8. Emit Normal event + update metrics
```

If any step returns an error, the reconcile is requeued with exponential backoff. The error is logged and a Warning event is emitted on the CR.

---

## Finalizer lifecycle

Finalizers are the mechanism Kubernetes uses to block CR deletion until cleanup is complete. `GenericReconciler` manages finalizers automatically.

**On create:** The first reconcile adds all declared finalizers (Katalog-level + CRD-level) to the CR via a patch. The patch uses `Update` semantics — if it fails with a conflict, the workqueue retries.

**On delete:** When `DeletionTimestamp` is set on the CR, the reconciler:
1. Runs the `onDelete` template block (if declared) — this creates Jobs or other cleanup resources
2. Waits for Jobs to complete (owner reference `blockOwnerDeletion: true` handles this)
3. Removes finalizers one by one via patch
4. When all finalizers are removed, Kubernetes proceeds with deletion

!!! warning "Disabled CRDs with live finalizers"
    If you disable a CRD in the Katalog (`enabled: false`) while CRs with Orkestra
    finalizers exist, those CRs will be stuck in terminating state. The reconciler
    is no longer running to remove the finalizers. Always delete all CRs before
    disabling a CRD entry.

---

## The safeReconcile wrapper

Every worker goroutine calls `safeReconcile` instead of calling the reconciler directly:

```go
func safeReconcile(ctx context.Context, reconciler func(ctx context.Context, key string) error, key string) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("reconciler panic: %v\n%s", r, debug.Stack())
        }
    }()
    return reconciler(ctx, key)
}
```

A panic in the reconcile function — a nil pointer dereference, an out-of-bounds slice access — is caught here, converted to an error, logged, and returned to the workqueue. The workqueue retries with backoff. Other CRDs are completely unaffected.

This is the per-CRD failure domain. A crash in the `Website` reconciler does not crash the `Database` reconciler.

---

## Template reconciliation

When `reconciler.default: true` and templates are declared, `runTemplateReconcile` is called:

```go
func (r *GenericReconciler[T]) runTemplateReconcile(ctx context.Context, obj T) error
```

This function:
1. Builds a `Resolver` from the live CR object
2. Determines which lifecycle phase this is (create vs reconcile vs delete)
3. For each resource type in the template (deployments, services, etc.):
   a. Evaluates `when` conditions — skips the resource if any condition fails
   b. Resolves template expressions for all fields
   c. Calls the corresponding OrkestraRegistry function (Create or Update)
4. Returns any error from the registry calls

The `Resolver` is built once per reconcile and shared across all template evaluations. It evaluates `{{ .spec.image }}` against the live CR's unstructured map.

---

## Metrics emitted per reconcile

Every reconcile cycle emits:

```
controller_reconcile_total{crd, result="success|error"}          +1
controller_reconcile_duration_seconds{crd}                       duration
controller_workers_active{crd}                                   updated (gauge)
controller_queue_depth{crd}                                      updated (gauge)
controller_resource_count{crd}                                   updated (gauge)
```

When providers are declared, each provider call also emits:

```
orkestra_provider_reconcile_total{crd, provider, kind, result}   +1
orkestra_provider_delete_total{crd, provider, kind, result}      +1
orkestra_provider_reconcile_duration_seconds{crd, provider, kind}  duration
```

`result` is `"success"` or `"failure"`. The `provider` label is the block name (e.g. `"aws"`, `"mongodb"`). The `kind` label is the resource kind within the block (e.g. `"s3"`, `"database"`).

`controller_resource_count` is the number of CRs currently managed — updated from the informer cache count. `controller_workers_active` is updated at the start and end of each reconcile.

---

## Health state tracking

`GenericReconciler` updates `CRDHealth` on every reconcile:

- `consecutiveFails` increments on error, resets to zero on success
- When `consecutiveFails >= degradeThreshold`, the CRD transitions to degraded
<!-- - A degraded CRD with `critical: true` marks the entire Orkestra instance as degraded -->
- Health state is exposed via `/katalog/{crd}/health` and `ork status`

---

## The managed label

On every reconcile, before any other logic, the reconciler ensures the CR has:

```
Labels:
  orkestra.konductor.io/managed: "true"

Annotations:
  orkestra.konductor.io/managed-by: <operator-name>
  orkestra.konductor.io/managed-since: <timestamp>
```

This label is the foundation of `ork reconcile all` — it uses `metav1.ListOptions{LabelSelector: "orkestra.konductor.io/managed=true"}` to scope its operations to exactly what this operator instance manages.

The patch is idempotent — if the label is already set, no API call is made.

---

## Typed vs dynamic mode

**Dynamic mode** (`apiTypes.location` not set):
- Objects are `*unstructured.Unstructured`
- Template expressions work against `u.Object` map
- All declarative features work
- No code generation required

**Typed mode** (`apiTypes.location` set):
- Objects are decoded into concrete Go types by the REST client
- Go hooks can do `obj.(*Website)` for type-safe field access
- The Resolver falls back to metadata-only context for typed objects
- `ork generate runtime` is required before `ork run`

The `GenericReconciler` itself does not distinguish — it calls the same functions. The distinction surfaces in the `Resolver` (unstructured gets full spec access) and in hook type assertions.

---

## Constructor vs default reconciler

When `reconciler.default: false` and a `constructor` is declared, `GenericReconciler` is not used. The custom reconciler returned by the constructor receives raw reconcile events and is responsible for its own entire lifecycle — finalizers, events, metrics, everything.

---

## Provider stats

When a CRD declares provider blocks, `GenericReconciler` accepts a `providerStats providerStatsRecorder` parameter and records the outcome of each provider call:

```go
// After provider.Reconcile succeeds
providerStats.RecordSuccess(block.Name)

// After provider.Reconcile fails
providerStats.RecordFailure(block.Name)

// After provider.Delete succeeds / fails
providerStats.RecordDeleteSuccess(block.Name)
providerStats.RecordDeleteFailure(block.Name)
```

The `providerStatsRecorder` interface is satisfied by `*health.ProviderStats`. The parameter is nil-safe — if not wired, recording is silently skipped. The stats are used by `BuildCRDInfoHandler` to populate per-provider error rates in the `/katalog/{crd}` response without querying Prometheus.

See [Constructor Guide](./constructor.md) for when this is appropriate.
