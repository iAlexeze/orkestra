# 03 — Input Binding and Resource Expansion

`Expand(m *orktypes.Motif, bindings map[string]string)` is the core function that turns a Motif into concrete resources. It runs at Katalog startup, before any reconcile cycle.

## Steps

```
Expand(m, bindings)
   │
   ├── validateBindings    — required inputs present; no unknown keys
   ├── resolveDefaults     — merge defaults for optional inputs not in bindings
   ├── yaml.Marshal(m.Resources) → resourceYAML string
   ├── renderInputs(resourceYAML, resolved)
   │     → replaces {{ index .inputs "name" }} with bound values
   └── yaml.Unmarshal(rendered) → *orktypes.HookTemplates
```

## validateBindings

Before expansion, `validateBindings` enforces:

1. Every `required: true` input must have a binding in `with:`.
2. No key in `with:` may refer to an input that the Motif does not declare.

These checks mirror `ValidateMotifImports` in `pkg/katalog/motif_validate.go` — the katalog runs them before calling `Expand`, so by the time `Expand` is called the bindings are already known-good.

## Input resolution

Static vs dynamic bindings:

| Type | Example | Resolved when |
|------|---------|---------------|
| Static | `volumeSize: "10Gi"` | At Katalog startup, inside `Expand` |
| Dynamic | `image: "{{ .data.postgresImage }}"` | Per reconcile, by the template resolver |

`renderInputs` replaces `{{ index .inputs "name" }}` with the static binding value. Dynamic bindings (Go templates referencing `.data.*`) are carried through unchanged — after expansion, the resource templates still contain `{{ .data.postgresImage }}`, which is resolved at reconcile time by the normal template resolver.

This two-phase resolution is what makes Motifs useful for dynamic parameters: the Motif structure is expanded once, but the runtime values (image tags, user-provided sizes) are resolved on every reconcile from the live CR data.

## MergeHookTemplates

After `Expand`, the resulting `*HookTemplates` is merged into the CRD's `OnReconcile` block:

```go
motif.MergeHookTemplates(entry.OperatorBox.OnReconcile, expanded)
```

`MergeHookTemplates` appends each resource slice from `expanded` to the corresponding slice in the target — `StatefulSets`, `Services`, `ConfigMaps`, etc. It never overwrites existing entries.

## Example

Motif:
```yaml
resources:
  statefulSets:
    - name: "{{ .metadata.name }}-db"
      image: "{{ index .inputs \"image\" }}"
      storageSize: "{{ index .inputs \"volumeSize\" }}"
```

Binding:
```go
bindings := map[string]string{
    "image":      "{{ .data.postgresImage }}",  // dynamic
    "volumeSize": "10Gi",                        // static
}
```

After `Expand`:
```yaml
statefulSets:
  - name: "{{ .metadata.name }}-db"
    image: "{{ .data.postgresImage }}"    # ← dynamic, resolved at reconcile time
    storageSize: "10Gi"                   # ← static, already substituted
```

→ Back to: [README.md](../README.md)
