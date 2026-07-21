# 02 — Garbage Collection

## The Problem

Kubernetes GC does **not** cascade owner references from namespace-scoped resources (CRs) to cluster-scoped resources:

| Owner | Dependent | GC works? |
|-------|-----------|-----------|
| Namespace-scoped → | Namespace-scoped | ✅ |
| Namespace-scoped → | Cluster-scoped | ❌ |

When a ReconcilerProbe CR (namespace-scoped) creates cluster-scoped resources (Namespaces, ClusterRoles, ClusterRoleBindings, PersistentVolumes), they become orphaned on CR deletion.

## The Solution

[`cluster_scoped_deletion.go`](../cluster_scoped_deletion.go) provides `DeleteOwnedClusterScopedResources()` — called during CR deletion to explicitly clean up owned cluster-scoped resources.

### How it works

1. **Collect sources** from `OnCreate`, `OnReconcile`, `OnDelete`
2. **Resolve names** through the template resolver
3. **Check ownership** — only delete if the CR is the owner
4. **Delete** the resource

### Currently handled resources

| Resource |
|----------|
| Namespace |
| ClusterRole |
| ClusterRoleBinding |
| PersistentVolume |
| Custom Resources |

## Adding a new cluster-scoped resource

1. **Add `DeleteIfOwned`** to the resource package (`pkg/resources/<resource>/`)
2. **Add collection function** in `cluster_scoped_deletion.go` following the existing pattern
3. **Call it** from `DeleteOwnedClusterScopedResources()`
4. **Update the list** above

### Pattern to follow

```go
func deleteOwned<Resources>(
    ctx context.Context,
    kube kubeclient.KubeClient,
    resolver *orktmpl.Resolver,
    obj domain.Object,
    box orktypes.OperatorBoxConfig,
) error {
    var srcs []orktypes.<Resource>TemplateSource
    if box.OnCreate != nil {
        srcs = append(srcs, box.OnCreate.<Resources>...)
    }
    if box.OnReconcile != nil {
        srcs = append(srcs, box.OnReconcile.<Resources>...)
    }
    if box.OnDelete != nil {
        srcs = append(srcs, box.OnDelete.<Resources>...)
    }

    for i, src := range srcs {
        name, err := resolver.Resolve(src.Name)
        if err != nil || name == "" {
            continue
        }
        if err := ork<resource>.DeleteIfOwned(ctx, kube, obj, name); err != nil {
            return fmt.Errorf("<resource>[%d] %q: %w", i, name, err)
        }
    }
    return nil
}
```

## Integration point

In the reconciler deletion path:

```go
// Cluster-scoped resources require explicit deletion: GC does not cascade owner references to them.
if err := runners.DeleteOwnedClusterScopedResources(ctx, kube, resolver, obj, r.operatorBox); err != nil {
    return fmt.Errorf("cluster-scoped resource cleanup: %w", err)
}
```

---

→ Back: [README](../README.md)
