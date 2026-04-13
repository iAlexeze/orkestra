# 05 — forEach Expansion

## What forEach does

`forEach` expands a single template source declaration into N declarations — one per element in a list field on the CR. The expansion happens **before** the runner is called. Every `run_*.go` function receives a flat, already-expanded slice and does not need to know about `forEach`.

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

## How expansion works

`run_foreach.go` contains one `expandForEach*` function per resource type. The function:

1. Fast-paths to return `srcs` unchanged when no source has a `ForEach` field set.
2. For each source with `ForEach != nil`, reads the list from `resolver.Data()` using `resolveListField`.
3. For each list item, creates a copy of the source with `ForEach = nil` (prevents re-expansion) and resolves the key template fields with an item-scoped resolver.
4. Appends expanded copies to the result; passes non-forEach sources through unchanged.

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
        for i, item := range resolveListField(resolver.Data(), src.ForEach.Field) {
            r := resolver.WithItem(item, src.ForEach.As, i)
            expanded := src
            expanded.ForEach = nil
            expanded.Name, _      = r.Resolve(src.Name)
            expanded.Namespace, _ = r.Resolve(src.Namespace)
            expanded.Host, _      = r.Resolve(src.Host) // resource-specific fields
            result = append(result, expanded)
        }
    }
    return result
}
```

## Which fields to resolve during expansion

Only resolve fields that are **commonly used in the name or namespace** and fields that appear in `{{ .item }}` expressions. The full template resolution happens later in B5 (`resolver.ResolveXxxTemplate`). Do not try to expand every field here — unexpanded fields will be resolved correctly in B5.

The minimum required fields to expand:

- `Name` — used in `activeNames` pre-pass and as the Kubernetes resource name.
- `Namespace` — used in `activeNames` and guard.
- Any field that forms part of the selector (e.g. `Host` for an Ingress, `Port` for a Service) if it is used in `{{ .item }}` expressions.

When in doubt, look at the existing functions (`expandForEachDeployments`, `expandForEachServices`) and match the same depth.

## The item resolver

```go
r := resolver.WithItem(item, src.ForEach.As, i)
```

`WithItem` creates a child resolver that injects:
- `.item` — the raw value of the current list element (string, number, or object).
- `.{{ src.ForEach.As }}` — an alias (the `as:` field). If `as: region`, then `{{ .region }}` works.
- `.index` — the zero-based position of this element in the list.

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

Declared in `pkg/types/`:

```go
type ForEachSpec struct {
    Field string `yaml:"field"` // dot-notation path to a list field on the CR
    As    string `yaml:"as"`    // template alias for the current item
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
