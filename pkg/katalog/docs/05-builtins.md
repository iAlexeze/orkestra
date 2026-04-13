# 05 — Built-in Type Handling

Orkestra operators can manage Kubernetes built-in resources (ConfigMap, Deployment, Service, etc.) as children. The `builtins.go` registry tells Orkestra how to treat each one correctly.

## The problem

Built-in resources don't all behave the same way in the Kubernetes API:

- `ConfigMap` has no `/status` subresource — PATCHing status returns 404.
- `v1/Pod` has status but no `observedGeneration` field — writing it would cause a diff loop.
- `apps/v1/Deployment` has both, so normal status patching works.

The reconciler's status path needs to know these differences before it attempts any API calls.

## BuiltInKind

```go
type BuiltInKind struct {
    Group      string
    Version    string
    Plural     string
    Namespaced bool
    APIPath    string  // "/api" for core, "/apis" otherwise

    Statusless             bool // treat as ready on existence; no status to read
    SkipStatusSubresource  bool // no /status endpoint; never PATCH status
    SkipObservedGeneration bool // has status but no observedGeneration field
    IsChild                bool // Orkestra may create this as a child resource
}
```

## Lookup

```go
meta := BuiltInMeta("configmap")  // case-insensitive, returns zero value for unknown kinds
```

Returns zero value (all `false`) for CRD kinds that are not in the built-in registry — the safe default for user-defined CRDs.

## How flags are used

`validateStatus()` runs once during Katalog boot and stamps these flags onto each `CRDEntry`:

```go
crd.IgnoreStatusPatch       // set when SkipStatusSubresource is true
crd.IgnoreObservedGeneration // set when SkipObservedGeneration is true
```

In the hot reconcile path, `r.crd.SkipStatusSubresource()` and `r.crd.SkipObservedGeneration()` read these pre-computed flags — no registry lookup at reconcile time.

## Adding a new built-in

Add an entry to `builtInRegistry` in `builtins.go`. The key is the lowercase Kind name. Look at an existing entry (e.g. `"configmap"`, `"deployment"`) for the correct field values. The `IsChild: true` flag marks resources that Orkestra creates as child resources.

→ Back to: [README.md](../README.md)
