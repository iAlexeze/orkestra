# `pkg/resources` — Kubernetes Resource Library

The `pkg/resources` package is the standard library of Kubernetes resource operations used by Orkestra's `GenericReconciler`. It provides idempotent, owner-reference-aware Create, Update, Delete, and Resolve functions for every resource type the reconciler can manage.

This package is not for direct end-user consumption. It is the layer the reconciler dispatches to when it processes declarative templates.

---

## Structure

```
pkg/resources/
├── deployments/
├── services/
├── secrets/
├── configmaps/
├── statefulsets/
├── jobs/
├── cronjobs/
├── ingresses/
├── pvcs/
├── pvs/
├── pods/
├── replicasets/
├── roles/
├── rolebindings/
├── serviceaccounts/
├── namespaces/
├── hpas/
├── pdbs/
├── customresources/
├── common/          ← shared utilities: probes, security, volumes, rolling update
└── template/        ← Resolver: evaluates Go templates against live CR fields
```

Each resource subdirectory exports four functions:

```go
Create(ctx, kube, owner, spec) error   // idempotent — no-op if exists
Update(ctx, kube, owner, spec) error   // drift correction — creates if missing
Delete(ctx, kube, owner, spec) error   // idempotent — no-op if absent
Resolve(src, ownerName) ResolvedSpec   // applies defaults and system labels
```

---

## The contract

- **Create** — sets owner references and system labels (`managed-by: orkestra`, `orkestra-owner: <cr-name>`). No-op if the resource already exists.
- **Update** — corrects drift. Creates the resource if it does not exist. Updates it if fields have changed.
- **Delete** — no-op if the resource does not exist.
- **Resolve** — takes a template source (with potentially unresolved field values) and returns a fully resolved spec.

All functions receive a `kubeclient.KubeClient` and a `domain.Object` (the owner CR). They never assume the owner's concrete type.

---

## Adding a new resource type

1. Create `pkg/resources/<kind>/` with `<kind>.go` and `types.go`.
2. Add the template source struct to `pkg/types/types_hook_templates.go`.
3. Add a resolver method in `pkg/resources/template/resolver.go`.
4. Add the runner in `pkg/reconciler/run_<kind>.go`.
5. Call the runner from `pkg/reconciler/generic.go`.

See `pkg/resources/deployments/` for a complete example.

---

**See also:** [Reconciler docs](../reconciler/docs/README.md) · [Contributing](./CONTRIBUTING.md)
