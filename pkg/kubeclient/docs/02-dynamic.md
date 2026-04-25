# 02 — DynamicListerWatcher

## Purpose

`DynamicListerWatcher` provides a `cache.ListerWatcher` for CRDs that have no registered Go type (`IsDynamic() == true`). These CRDs use `*unstructured.Unstructured` as their in-memory representation. The informer factory receives this ListerWatcher via `ForListerWatcher` and passes it directly to `cache.NewSharedIndexInformer`.

## Construction

```go
func (k *Kubeclient) NewDynamicListerWatcher(info CRDInfo, opts ListOptions) cache.ListerWatcher {
    return &DynamicListerWatcher{
        kube:          k,
        gvr:           schema.GroupVersionResource{Group: info.Group, Version: info.Version, Resource: info.Plural},
        namespace:     info.Namespace,
        namespaced:    info.Namespaced,
        labelSelector: opts.LabelSelector,
        fieldSelector: opts.FieldSelector,
    }
}
```

`kube` is stored as a pointer, not copied — the dynamic client (`kube.dynamic`) is resolved lazily at `List`/`Watch` call time, which happens after `kubeclient.Start()`. This is intentional: the REST config may not be valid at construction time.

## Namespace resolution in List/Watch

```go
if d.namespaced {
    ns := d.namespace
    if ns == "" { ns = metav1.NamespaceAll }
    return d.kube.dynamic.Resource(d.gvr).Namespace(ns).List(ctx, options)
}
// cluster-scoped: no Namespace() call
return d.kube.dynamic.Resource(d.gvr).List(ctx, options)
```

When `d.namespace` is empty and the resource is namespaced, `metav1.NamespaceAll` (`""` in ListOptions) instructs the API server to return resources from all namespaces. When `d.namespace` is set, only that namespace is watched.

## Tier 1 namespace scoping

When the informer namespace filter's `IsSingleNamespace()` is true, `konstructOrkestra` resolves the namespace before calling `NewDynamicListerWatcher`:

```go
dynNamespace := crd.Namespace      // operator-level default
if opts.Namespace != "" {
    dynNamespace = opts.Namespace  // Tier 1 filter overrides
}
lw := kube.NewDynamicListerWatcher(kubeclient.CRDInfo{
    ...
    Namespace: dynNamespace,
    ...
}, ...)
```

This ensures the informer cache for this CRD only ever holds objects from the single allowed namespace. There is no Tier 2 filter check needed at the queue level for these CRDs — events from other namespaces never arrive.

## Selector injection

Both `List` and `Watch` inject `LabelSelector` and `FieldSelector` into `ListOptions` before calling the dynamic client, using `utils.Merge` to combine with any selector already present in the `options` argument (from the reflector):

```go
if d.labelSelector != "" {
    utils.Merge(&options.LabelSelector, d.labelSelector, ",")
}
```

This is additive — the informer's own selector requirements are preserved.

## CRDInfo fields used

| Field | Used for |
|-------|----------|
| `Group`, `Version`, `Plural` | `schema.GroupVersionResource` — identifies the API endpoint |
| `Namespace` | Scopes the watch; empty = all namespaces |
| `Namespaced` | Controls whether `Namespace()` is called in the builder chain |
| `Kind`, `APIPath`, `GroupVersion` | Not used by DynamicListerWatcher directly; used by other kubeclient paths |

---

**← Back to** [01 — GenericClient](01-genericclient.md)
