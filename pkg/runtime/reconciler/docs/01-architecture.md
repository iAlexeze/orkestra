# 01 — Reconciler Architecture

## The big picture

Every CR (Custom Resource) that Orkestra manages goes through one shared reconcile pipeline. The pipeline is implemented in `generic.go` and `run_template_reconcile.go`. Resource-specific logic lives in isolated `run_*.go` files.

```text
Kubernetes event (Add/Update/Delete)
         │
         ▼
GenericReconciler.Reconcile()          generic.go
   │
   ├── workerSem.Acquire()             concurrency gate (ResizableSemaphore)
   │     blocks if autoscaler has reduced effective concurrency below goroutine count
   │
   └── reconcileCore()
          │
          ├── Context enrichment (logger, requestID, CRD name)
          ├── Cache lookup — informer store, no API call
          ├── Deletion routing → handleDeletion()
          ├── Finalizer / managed-label / annotation patching
          │
          └── reconcileImpl()
                 │
                 ├── Phase 1 — Rollback gate
                 │     isRollbackActive() → true:
                 │       runRollback()  — applies onRollback templates with .previous.*
                 │       return         — normal reconcile is blocked until spec changes
                 │
                 ├── Phase 2 — Reconcile-time mutation (if mutation rules declared)
                 ├── Phase 3 — Reconcile-time validation (if validation rules declared)
                 │
                 ├── Phase 4 — dispatch:
                 │     ├── Go hook (hooks.OnReconcile)        — user-provided, typed
                 │     ├── Declarative templates              — runTemplateReconcile()
                 │     └── No-op                              — finalizers + events only
                 │
                 ├── Phase 5 — Rollback trigger check (on dispatch error)
                 │     history.record(); shouldRollback() → true:
                 │       markRollbackActive()  — writes RollbackGenerationAnnotation
                 │
                 └── Phase 6 — Spec snapshot (on dispatch success)
                       snapshotSpec()  — writes PreviousSpecAnnotation (gzip+base64)
                       clearFailureHistory()

   workerSem.Release() + autoMetrics.RecordReconcile(duration, failed)
```

The semaphore capacity starts at `baseline.workers`. When `autoscale:` is
declared, the autoscaler resizes it at runtime without stopping goroutines.
For non-autoscaled CRDs the semaphore is always uncontested.

## runTemplateReconcile — execution order

`run_template_reconcile.go` is the declarative path. It builds a resolver, enriches it with external data, then dispatches to resource-type runners.

```text
Step 1   NewResolver(obj)
            .spec.*, .status.*, .metadata.*
            .children.* (owned child resources)

Step 2   readCross(r.rc.Cross)
            .cross.<kind>.spec.*, .status.*
            Read from sibling informer caches — zero API calls.
            Falls back to HTTP endpoint for cross-binary CRDs.

Step 3   runGit()
            .git.commit, .git.changed, .git.path
            Only if git: is declared.

Step 4   runExternal()
            .external.<n>.status, .body
            HTTP calls to external services.
            Only if external: is declared.

Step 5   runDocker()
            Docker build/push.
            Only if docker: is declared.

Step 6   runResourceGroup(onCreate, update=false)
            All resource types in the onCreate: block.

Step 7   runResourceGroup(onReconcile, update=true)
            All resource types in the onReconcile: block.

Step 8   runProviders()
            aws:, mongodb:, ... — external infrastructure providers.
```

Each step can reference data produced by all earlier steps. The resolver is passed by value between steps — it is immutable.

## runResourceGroup — per-type dispatch

`runResourceGroup` calls each resource runner with the same signature:

```text
expandForEach*(resolver, t.<Resource>)   →   []TypedTemplateSource
runXxx(ctx, kube, resolver, owner, srcs, update, guard)
```

The `expandForEach*` call happens **before** the runner is invoked. Every runner receives an already-expanded slice and does not need to know about `forEach`. The field can be a list (`.item` = element) or a map (`.item` = key, `.value` = map value).

```text
runResourceGroup()
   │
   ├── expandForEachNamespaces           → runNamespaces
   ├── expandForEachSecrets              → runSecrets
   ├── expandForEachConfigMaps           → runConfigMaps
   ├── expandForEachServiceAccounts      → runServiceAccounts
   ├── expandForEachRoles                → runRoles          ← scoped to a namespace
   ├── expandForEachRoleBindings         → runRoleBindings   ← binds Role to ServiceAccount
   ├── expandForEachReplicaSets          → runReplicaSets
   ├── expandForEachDeployments          → runDeployments
   ├── expandForEachServices             → runServices
   ├── expandForEachJobs                 → runJobs
   ├── expandForEachCronJobs             → runCronJobs
   ├── expandForEachStatefulSets         → runStatefulSets
   ├── expandForEachPVs                  → runPVs
   ├── expandForEachPVCs                 → runPVCs
   ├── expandForEachIngresses            → runIngresses
   ├── expandForEachHPAs                 → runHPAs
   └── expandForEachPDBs                 → runPDBs
```

Ordering matters for RBAC: ServiceAccounts come before Roles, and Roles before RoleBindings — a binding cannot reference a role that does not yet exist. The ordering in `runResourceGroup` guarantees this without coordination logic in the runners.

## The update flag

The same runner handles both `onCreate` and `onReconcile`. The `update` flag distinguishes them:

| `update` | Called from | Semantics |
|----------|-------------|-----------|
| `false`  | `onCreate`  | Idempotent create — skip if exists |
| `true`   | `onReconcile` | Drift correction — patch if changed |

A source with `reconcile: true` under `onCreate` means "create it and keep it in sync without a separate `onReconcile` declaration". The runner handles this by calling `Update` after a successful `Create` when `src.Reconcile` is true.

## The namespace guard

Before any API call the runner checks whether the target namespace is allowed. The guard is a closure created in `runResourceGroup`:

```go
guard := r.namespaceGuardFunc(ctx, obj)
```

Inside each runner, call it early — after namespace resolution, before any API call:

```go
if guard != nil && !guard(ctx, owner, ns) {
    continue // CheckNamespace already logged the reason
}
```

The guard is nil when the CRD has no `restrictedNamespaces` / `allowedNamespaces` config. Always do a nil check.

## Deletion path

On CR deletion, `handleDeletion` runs. It dispatches to:
- `hooks.OnDelete` — Go hook, if registered.
- `runTemplateOnDelete` — interprets the `onDelete:` block.

`runTemplateOnDelete` branches on `t.Ordered`:

- **`ordered: false` (default)** — Kubernetes owner references handle cascade deletion automatically. Only `jobs:` (fire-and-forget cleanup) and provider teardown run explicitly.
- **`ordered: true`** — Sequential deletion with completion gates. Implemented in `run_delete_ordered.go`:
  - Stages are taken from `Groups []HookTemplates` when declared; otherwise the flat resource fields are treated as a single implicit group.
  - Each stage: submit all deletes with `PropagationPolicy: Foreground`, then poll the API server until every resource returns 404.
  - `Timeout *Duration` (default `5m`) caps each stage. Exceeding it returns an error that blocks finalizer removal.
  - Polling uses `DynamicClient` (not the informer) so the answer is authoritative even before the watch stream catches up.

## cross_util.go — rawToMap

`cross_util.go` provides two package-level helpers used by the cross-CRD reading path:

- **`rawToMap(raw interface{})`** — converts any informer cache object to `map[string]interface{}`. Fast path for `*unstructured.Unstructured` (returns `u.Object` directly, zero allocation); JSON round-trip for typed objects.
- **`metaField(objMap, field)`** — safe nil-checked string extraction from `objMap["metadata"]`.

These replaced the `raw.(*unstructured.Unstructured)` type assertions in `ReadCrossFromInformer` and `ReadCrossFromInformerByLabel`. The same `rawToMap` pattern is duplicated in the kordinator package (`cr_handlers.go`) to avoid a cross-package dependency.

## Status patching

After reconcile (success or failure), `patchStatusWithChildren` is called unconditionally. It reads owned child resources from the informer cache and writes a `Ready` condition. This is automatic — runners do not need to touch status.

For a full explanation of the rollback subsystem — trigger evaluation, spec snapshotting, `onRollback` templates, and annotation lifecycle — see [08 — Rollback](08-rollback.md).

---

**Next →** [02 — The run_*.go Function Contract](02-run-pattern.md)
