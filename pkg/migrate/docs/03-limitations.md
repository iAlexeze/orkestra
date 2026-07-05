# Limitations

`ork migrate` is a mechanical starting point. It handles the deterministic rewrites and flags the rest. These are the cases that require manual attention.

## Not automatically rewritten

### Embedded client.Client

The standard controller-runtime struct pattern:

```go
type WebAppReconciler struct {
    client.Client         // embedded
    Scheme *runtime.Scheme
}
```

The tool does not rewrite the struct. After migration, replace it with Orkestra's interfaces:

```go
type WebAppReconciler struct {
    informer cache.SharedIndexInformer
    kube     kubeclient.KubeClient
    ev       event.Recorder
}
```

And update the constructor function signature:

```go
func NewWebAppReconciler(
    kube kubeclient.KubeClient,
    informer cache.SharedIndexInformer,
    ev event.Recorder,
) domain.Reconciler {
    return &WebAppReconciler{kube: kube, informer: informer, ev: ev}
}
```

### r.Get, r.Create, r.Patch inside sub-methods

Calls on the embedded client in sub-methods (e.g. `reconcileDeployment`, `reconcileService`) are not rewritten. After changing the struct, update each call:

```go
// Before
r.Get(ctx, client.ObjectKey{...}, existing)
r.Create(ctx, desired)
r.Patch(ctx, existing, patch)

// After — manual Get/Create/Patch
r.kube.Get(ctx, namespace, name, existing)  // or use informer cache
r.kube.Create(ctx, desired)
r.kube.Patch(ctx, existing, patch)

// After — Orkestra resources (05-constructor-orkestra-resources style)
orkdeploy.Update(ctx, r.kube, owner, spec)
```

### r.Status().Update()

Flagged inline but not fully replaced — the field map cannot be inferred from the source. Replace with:

```go
r.kube.PatchStatus(ctx, webapp, apiv1.GroupVersionResource, map[string]interface{}{
    "phase":    "Running",
    "endpoint": fmt.Sprintf("%s.%s.svc.cluster.local", webapp.Name, webapp.Namespace),
    "replicas": webapp.Spec.Replicas,
})
```

### ctrl.Result{RequeueAfter: X}

The tool removes `RequeueAfter` and flags it. If you need time-based requeue, return an error — Orkestra's exponential backoff will retry. For periodic reconciliation, use an `external:` schedule or a `when:` condition.

### kubebuilder RBAC markers

Comments like `// +kubebuilder:rbac:groups=...` are left as-is. They have no effect in Orkestra — RBAC is declared in the Katalog's `resources:` list and generated via `ork generate rbac`.

### main.go, scheme registration, manager setup

These are separate files — the tool only touches the file you pass it. Delete them manually after migration.

## Out of scope by design

- Webhooks — `SetupWebhookWithManager` and admission handlers are not touched.
- Multi-file operators — the tool processes one file at a time. Run it on each reconciler file separately.
- Finalizer logic — the tool does not analyse `DeletionTimestamp` handling or finalizer add/remove patterns.
