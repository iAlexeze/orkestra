# 05 — Built-in kind registry

## What it is

`builtins.go` owns the authoritative registry of every Kubernetes built-in resource that Orkestra knows about. It is the single source of truth for:

- API group, version, plural, and path for each kind
- Whether the resource is namespaced or cluster-scoped
- Whether it uses a status subresource
- Whether it is "statusless" (ConfigMap, Secret, etc.) — ready on existence
- How to detect whether a CRD uses this resource (for RBAC generation)
- GVR variables used across the package

Any package that needs built-in kind metadata imports `pkg/children` — not `pkg/katalog` or any other package.

## Key functions

| Function | Purpose |
|----------|---------|
| `LookupBuiltIn(kind)` | Look up a kind by name (case-insensitive, expands shorthands like `"hpa"`) |
| `BuiltInMeta(kind)` | Return the `BuiltInKind` struct for a kind (zero value when not found) |
| `GVRForBuiltIn(kind)` | Return the GVR for a kind |
| `IsBuiltIn(kind)` | Report whether a kind is in the registry |
| `AllBuiltInKinds()` | Return all canonical Kind names, sorted |
| `AllBuiltInKindDefs()` | Return all registry entries (for RBAC generation and iteration) |
| `LookupBuiltInByResource(resource)` | Look up by singular key, shorthand, or plural resource name |
| `ChildGVRs()` | Return all child-resource GVR entries (used by kordinator) |

## GVR variables

All GVR variables (`DeploymentGVR`, `ServiceGVR`, `PodGVR`, `EventGVR`, etc.) are defined in `gvr.go` using `gvrOrPanic`, which panics at init time if a kind is missing from the registry — ensuring the registry and GVR list stay in sync.

Any file in this package (or importing this package) that needs a GVR uses these variables directly — there is no need to construct `schema.GroupVersionResource` by hand.

## BuiltInKind struct

```go
type BuiltInKind struct {
    Kind                  string
    Group                 string
    Version               string
    Plural                string
    APIPath               string
    Namespaced            bool
    Statusless            bool
    SkipStatusSubresource bool
    SkipObservedGeneration bool
    OrkestraInternal      bool // Orkestra's own control-plane resources
    Detect                func(crd orktypes.CRDEntry) bool
}
```

`Detect` is used by RBAC generation: `katalog.Uses("deployment")` returns true if any enabled CRD declares a Deployment template. This keeps RBAC rules declarative — adding a new resource type to the registry automatically makes it available for RBAC detection.

## Enrichment result

`LookupBuiltIn` returns an `EnrichmentResult`:

```go
type EnrichmentResult struct {
    Found        bool
    Kind         string // canonical (e.g. "Deployment" not "deployment")
    DisplayGroup string // "core" for empty group, otherwise the group
    BuiltIn      BuiltInKind
}
```

`katalog.EnrichCRDEntry` uses this to fill in group/version/plural for kind-only declarations in the Katalog YAML.
