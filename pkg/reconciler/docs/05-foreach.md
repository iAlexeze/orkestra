# 05 — forEach Expansion

## What forEach does

`forEach` expands a single template source declaration into N declarations — one per element in a **list or map field** on the CR. The expansion happens **before** the runner is called. Every `run_*.go` function receives a flat, already-expanded slice and does not need to know about `forEach`.

```yaml
onCreate:
  ingresses:
    - name: "{{ .metadata.name }}-{{ .item }}"
      host: "{{ .item }}.example.com"
      forEach:
        field: spec.domains
        as: item
```

For a CR with `spec.domains: ["api", "admin", "webhooks"]`, this produces three `IngressTemplateSource` entries before `runIngresses` is called.

## List fields vs map fields

`forEach` accepts both list and map fields transparently.

### List field (existing behaviour)

```yaml
spec.regions: [us-east-1, eu-west-1, ap-southeast-1]
```

- `.item` / `.<as>` → the element value (`"us-east-1"`)
- `.index` → 0-based position
- `.value` → nil (not set for list items)

### Map field (new)

```yaml
spec.regions:
  us-east-1: {replicas: 3, port: 8080}
  eu-west-1:  {replicas: 1, port: 8081}
```

- `.item` / `.<as>` → the map key (`"us-east-1"`)
- `.value` → the map value (`{replicas: 3, port: 8080}`)
- `.value.replicas` → `"3"`, `.value.port` → `"8080"`
- `.index` → 0-based position (keys sorted alphabetically)

Map items iterate in **sorted key order** — deterministic across reconciles.

### Using both shapes from one Katalog template

Use `or` to fall back gracefully when a region entry omits a field:

```yaml
onReconcile:
  deployments:
    - name: "{{ .metadata.name }}-{{ .item }}"
      image: "{{ .spec.image }}"
      replicas: "{{ or .value.replicas .spec.defaultReplicas }}"
      port: "{{ or .value.port .spec.defaultPort }}"
      forEach:
        field: spec.regions
        as: item
```

This works for both CR shapes without changing the Katalog.

## How expansion works

`run_foreach.go` contains one `expandForEach*` function per resource type. The function:

1. Fast-paths to return `srcs` unchanged when no source has a `ForEach` field set.
2. For each source with `ForEach != nil`, calls `resolveForEachItems` to get the iteration steps.
3. For each step, creates a copy of the source with `ForEach = nil` (prevents re-expansion) and resolves key template fields with an item-scoped resolver.
4. Appends expanded copies; passes non-forEach sources through unchanged.

```go
func expandForEachIngresses(
    resolver *orktmpl.Resolver,
    srcs []orktypes.IngressTemplateSource,
) []orktypes.IngressTemplateSource {
    if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
        return srcs // fast path
    }
    var result []orktypes.IngressTemplateSource
    for _, src := range srcs {
        if src.ForEach == nil {
            result = append(result, src)
            continue
        }
        for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
            ir := itemResolver(resolver, fi, src.ForEach.As, i)
            expanded := src
            expanded.ForEach = nil
            expanded.Name, _      = ir.Resolve(src.Name)
            expanded.Namespace, _ = ir.Resolve(src.Namespace)
            expanded.Host, _      = ir.Resolve(src.Host) // resource-specific fields
            result = append(result, expanded)
        }
    }
    return result
}
```

## Which fields to resolve during expansion

Only resolve fields that are **commonly used in the name or namespace** and fields that use `.item` or `.value.*` expressions. Full template resolution happens later in B5 (`resolver.ResolveXxxTemplate`). Do not expand every field here — unexpanded fields are resolved correctly in B5.

The minimum required fields to expand:

- `Name` — used in `activeNames` pre-pass and as the Kubernetes resource name.
- `Namespace` — used in `activeNames` and guard.
- Any field that forms part of the selector (e.g. `Host` for Ingress, `Port` and `Selector` for Service) if it uses `.item` or `.value.*` expressions.

When in doubt, look at `expandForEachDeployments` or `expandForEachServices` and match the same depth.

## The item resolver

```go
// List item:
ir := resolver.WithItem(fi.key, src.ForEach.As, i)

// Map item (fi.value != nil):
ir := resolver.WithItemAndValue(fi.key, fi.value, src.ForEach.As, i)
```

The helper `itemResolver(base, fi, as, i)` in `run_foreach.go` picks automatically based on whether `fi.value` is set.

`WithItem` injects:
- `.item` — the current list element.
- `.<as>` — alias from `forEach.as`.
- `.index` — 0-based position.

`WithItemAndValue` additionally injects:
- `.value` — the map value (object or string). Access nested fields as `.value.replicas`.

`when:` and `anyOf:` conditions on a `forEach` source are evaluated per-item — each expanded copy may pass or fail independently.

## Calling expandForEach from runResourceGroup

`run_template_reconcile.go:runResourceGroup` must call the expand function before passing the slice to the runner:

```go
if err := runIngresses(ctx, kube, resolver, obj,
    expandForEachIngresses(resolver, t.Ingresses), update, guard); err != nil {
    return err
}
```

Never pass `t.Ingresses` directly to the runner. Always go through `expandForEachIngresses`.

## ForEachSpec type

Declared in `pkg/types/foreach.go`:

```go
// ForEachSpec declares dynamic expansion over a list or map field.
type ForEachSpec struct {
    // List field → .item = element, .value not set
    // Map field  → .item = key, .value = map value
    Field string `yaml:"field"`

    // As is the template alias for the current item key.
    // Default: "item" — {{ .item }}
    // When set: {{ .<as> }} also works alongside {{ .item }}
    As string `yaml:"as,omitempty"`
}
```

Every template source struct that supports `forEach` must embed it as `*ForEachSpec` (pointer, so `yaml:",omitempty"` works):

```go
type IngressTemplateSource struct {
    // ...
    ForEach *ForEachSpec `yaml:"forEach,omitempty"`
}
```

---

**Next →** [06 — normalize: Spec Normalization](06-normalize.md)
