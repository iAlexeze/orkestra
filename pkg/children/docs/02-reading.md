# 02 — Reading child resources from the cluster

## readResourceGroup

`readResourceGroup` is the core read function. It takes a GVR and a list of resolved names, performs one GET per name, and returns `map[name → objectMap]`.

```
readResourceGroup(ctx, kube, obj, resolver, DeploymentGVR, names)
→ map["my-api"] = { apiVersion, kind, metadata, spec, status, ... }
```

Each GET is served from the informer cache (`ResourceVersion: "0"`) — no quorum read, which avoids unnecessary API server load during reconcile.

Namespaced resources are read from the owner CR's namespace. Cluster-scoped resources (PersistentVolume, Namespace, ClusterRole, etc.) are read without a namespace.

When a resource cannot be found (404) or the GET fails, the name is omitted from the map and the error is logged at debug level. Status patching proceeds with whatever was successfully read — a missing child resolves to `""` in template expressions.

## readCustomResourceGroup

Custom resources have dynamic GVRs resolved via the REST mapper at runtime (the kind is declared in the Katalog YAML but the GVR is not known statically). `readCustomResourceGroup` resolves each unique `APIVersion/Kind` pair to a GVR once, then reads all instances.

## readEndpointSlicesForServices

For every declared Service, `readEndpointSlicesForServices` fetches the matching EndpointSlice (by `kubernetes.io/service-name` label) and exposes it under `.children.endpointslices`. This is separate from the `_endpoints` enrichment — the raw EndpointSlice object is useful for callers that want full slice detail rather than the flattened `_endpoints` list.

## firstValue

`firstValue(m)` returns the first value from a map for use as the singular shorthand key (`.children.deployment`). When the map is empty it returns a placeholder `map[string]interface{}{}` so template expressions that navigate into it still resolve to `""` rather than failing.

## mergeTemplates

`mergeTemplates` collects all template sources from both `onCreate` and `onReconcile` hooks into a single `orktypes.HookTemplates` struct. `ReadChildren` works off this merged view so it reads resources declared in either hook.

→ Next: [03-names-foreach.md](03-names-foreach.md)
