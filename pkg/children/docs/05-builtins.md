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

`BuiltInKind` carries Kubernetes API identity and Orkestra readiness policy — it does not carry enrichment configuration.

```go
type BuiltInKind struct {
    // Kubernetes API identity
    Kind       string
    Group      string
    Version    string
    Plural     string
    APIPath    string
    Namespaced bool
    Shorthands []string

    // Usage detection (RBAC generation)
    Detect func(crd orktypes.CRDEntry) bool

    // Orkestra readiness policy
    Statusless             bool
    SkipStatusSubresource  bool
    SkipObservedGeneration bool
    IsChild                bool
    OrkestraInternal       bool
}
```

`Detect` is used by RBAC generation: `katalog.Uses("deployment")` returns true if any enabled CRD declares a Deployment template. This keeps RBAC rules declarative — adding a new resource type to the registry automatically makes it available for RBAC detection.

## enrichmentMeta map

Enrichment configuration lives in a parallel map — separate from `BuiltInKind` so Kubernetes API identity and enrichment concerns can evolve independently.

```go
type enrichmentEntry struct {
    Target     bool     // marks this kind as a valid enrich: target
    EnrichKeys []string // synthetic keys this kind provides (e.g. "owner", "replicasets")
}

var enrichmentMeta = map[string]enrichmentEntry{
    "deployment":  {Target: true, EnrichKeys: []string{"replicasets"}},
    "statefulset": {Target: true, EnrichKeys: []string{"pvcs"}},
    "replicaset":  {Target: true, EnrichKeys: []string{"owner"}},
    "service":     {Target: true, EnrichKeys: []string{"backingpods"}},
    // ...
}
```

`enrichmentMeta` is read once at package init to build two cached indexes:

- `enrichmentGroups` — `canonical name → all valid identifiers` (name, plural, shorthands, synthetic keys)
- `enrichmentIndex` — `any identifier → canonical name` (reverse lookup, O(1))

`enrichmentEnabled`, `IsValidEnrichmentTarget`, and `SupportedEnrichmentGroups` all read from these indexes — `buildEnrichmentGroups` is called exactly once at init.

When adding a new resource kind that supports context enrichment, update **both** `builtInRegistry` and `enrichmentMeta` in `builtins.go`.

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
