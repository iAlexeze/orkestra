# 03 — Name resolution and forEach expansion

## Resolved names

Each declared template source produces one or more `resolvedChildName` values:

```go
type resolvedChildName struct {
    name       string // concrete resource name (after template evaluation)
    namespace  string // empty for cluster-scoped resources
    namespaced bool
}
```

`resolveName` evaluates any Go template expressions in the name field against the resolver's current data (the owner CR's fields). A name like `"{{ .metadata.name }}-api"` is expanded before the GET is issued.

## forEach expansion

A template source can declare `forEach` to expand one source into N sources — one per item in a list field on the owner CR. For example:

```yaml
deployments:
  - name: "{{ .item }}-api"
    forEach:
      field: spec.tenants
```

`ExpandForEachDeployments` (and the equivalent for each built-in type) processes this: it reads `spec.tenants` from the resolver, iterates over its items, and produces one template source per item with `.item` bound to the current value.

The `forEach` expansion functions are exported so that `pkg/reconciler` can call them during the template application phase (before any resources are created). The same expansion logic runs in both the reconcile phase (to know what to create) and the read phase (to know what to read back).

## Custom resource forEach

Custom resources use `ExpandForEachCustomResources` which follows the same pattern but handles the additional `APIVersion` and `Kind` fields required to resolve the GVR dynamically.

→ Next: [04-enrichment.md](04-enrichment.md)
