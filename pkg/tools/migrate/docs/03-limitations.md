# Limitations

`ork migrate` is a mechanical starting point. It handles the deterministic rewrites and flags the rest.

## toclient mode

### r.Get, r.Create, r.Patch via embedded client in sub-methods

`ork migrate --mode toclient` preserves your struct and all call sites. If your reconciler uses `client.Client` via embedding (promoted methods on the struct), the struct itself and sub-method calls remain unchanged and compile without modification.

If you later want to migrate those calls to Orkestra's `kubeclient.Interface`, do it by hand or run `--mode native`.

### r.Status().Update()

`toclient` mode leaves `Status().Update()` in place — it compiles and works via `client.Client`. The TODO flag is not applied in this mode.

---

## native mode

### r.Get, r.Create, r.Patch inside sub-methods

Calls on the embedded client in sub-methods (e.g. `reconcileDeployment`, `reconcileService`) are not rewritten. After changing the struct, update each call:

```go
// Before
r.Get(ctx, client.ObjectKey{...}, existing)
r.Create(ctx, desired)
r.Patch(ctx, existing, patch)

// After — kubeclient
r.kube.Get(ctx, namespace, name, existing)
r.kube.Create(ctx, desired)
r.kube.Patch(ctx, existing, patch)

// After — Orkestra managed resources
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

---

## Out of scope in both modes

- **kubebuilder RBAC markers** — `// +kubebuilder:rbac:groups=...` are left as-is. They have no effect in Orkestra — declare resources in the Katalog's `resources:` list and generate RBAC via `ork generate rbac`.
- **main.go, scheme registration, manager setup** — these are separate files; the tool only touches the file you pass it. Delete them manually.
- **Webhooks** — `SetupWebhookWithManager` and admission handlers are not touched.
- **Multi-file operators** — the tool processes one file at a time. Run it on each reconciler file separately.
- **Finalizer logic** — the tool does not analyse `DeletionTimestamp` handling or finalizer add/remove patterns.

---

Back: [README](../README.md)
