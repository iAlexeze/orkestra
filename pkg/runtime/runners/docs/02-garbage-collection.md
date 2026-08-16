# 02 — Garbage Collection

## The Problem

Kubernetes GC does **not** cascade owner references from namespace-scoped resources (CRs) to cluster-scoped resources:

| Owner | Dependent | GC works? |
|-------|-----------|-----------|
| Namespace-scoped → | Namespace-scoped | ✅ |
| Namespace-scoped → | Cluster-scoped | ❌ |

When a CR creates cluster-scoped resources (Namespaces, ClusterRoles, ClusterRoleBindings, PersistentVolumes), they orphan on CR deletion. Explicit deletion is required.

## Two deletion paths

| Path | Mechanism | File |
|------|-----------|------|
| **CR deletion** | Template-based — resolves names from the box declaration, expands `forEach` | [`cluster_scoped_deletion.go`](../cluster_scoped_deletion.go) |
| **Surface switch** | Label-selector sweep — finds resources by `orkestra-owner=<prevOwnerKey>` | [`surface_sweep.go`](../surface_sweep.go) |

### Why two mechanisms?

Template-based deletion works at CR deletion time because the CR's spec is intact — names resolve, `forEach` expands correctly.

For surface switches the spec may have already changed before cleanup runs (e.g. `spec.regions` is cleared when routing away from a `regional` target). Template-based deletion would expand `forEach` to nothing and silently miss the orphans. Label-selector sweep is immune to spec changes.

Namespace-scoped resources on the CR deletion path do not need explicit cleanup — Kubernetes GC handles them via owner references. Only cluster-scoped resources require `DeleteOwnedClusterScopedResources`.

## CR deletion — `DeleteOwnedClusterScopedResources`

Covers `onCreate`, `onReconcile`, and `onDelete` blocks. Called from the reconciler deletion path.

### Currently handled resource types

| Resource |
|----------|
| Namespace |
| ClusterRole |
| ClusterRoleBinding |
| PersistentVolume |
| Custom Resources (cluster-scoped) |

### Adding a new cluster-scoped resource type

1. Add `DeleteIfOwned` to the resource package (`pkg/resources/<resource>/`)
2. Add a `deleteOwned<Resource>` function to `cluster_scoped_deletion.go` following the existing pattern — collect sources from `allHooks(box)`, call `ExpandForEach*`, resolve name via `resolver.Resolve`, call `DeleteIfOwned`
3. Call it from `DeleteOwnedClusterScopedResources`
4. Update the table above

## Surface switch — `SweepOwned*`

`SweepOwnedNamespacedResources` and `SweepOwnedClusterScopedResources` list all resources of each known type with `orkestra-owner=<prevOwnerKey>` and delete them.

### Adding a new resource type to the sweep

Add a list+delete block to the appropriate sweep function in `surface_sweep.go`:

```go
// namespaced
if list, err := cs.AppsV1().<Resources>(ns).List(ctx, opts); err == nil {
    for _, r := range list.Items {
        collect("<resource>/"+r.Name, cs.AppsV1().<Resources>(ns).Delete(ctx, r.Name, dopts))
    }
}

// cluster-scoped
if list, err := cs.<Group>().<Resources>().List(ctx, opts); err == nil {
    for _, r := range list.Items {
        collect("<resource>/"+r.Name, cs.<Group>().<Resources>().Delete(ctx, r.Name, dopts))
    }
}
```

## Integration points

**CR deletion** (`handleDeletion`):
```go
if err := runners.DeleteOwnedClusterScopedResources(ctx, kube, resolver, obj, box); err != nil {
    return err
}
```

**Surface switch** (`cleanupPreviousSurface`):
```go
runners.SweepOwnedNamespacedResources(ctx, kube, prevOwnerKey, ns)
runners.SweepOwnedClusterScopedResources(ctx, kube, prevOwnerKey)
```

---

→ Back: [README](../README.md)
