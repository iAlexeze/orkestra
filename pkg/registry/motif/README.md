# pkg/registry/motif

The motif package loads, validates, and expands Motif YAML files. A Motif is a reusable infrastructure template — a named set of Kubernetes resources (StatefulSets, Services, PVCs, etc.) with declared inputs that callers bind when importing.

A Katalog imports a Motif instead of writing all stateful resource declarations by hand:

```yaml
operatorBox:
  imports:
    - motif: postgres       # load from registry or local file
      with:
        image: "{{ .data.postgresImage }}"
        volumeSize: "{{ .data.postgresVolumeSize }}"
```

## What lives here

| File | Role |
|------|------|
| `loader.go` | `Load`, `LoadImport` — read a Motif YAML from disk or registry; strict structural validation via `utils.StrictUnmarshal` |
| `expander.go` | `Expand` — bind inputs and render the Motif's resource templates into a `HookTemplates` value; `MergeHookTemplates` — merge expanded resources into an existing block |

## Validation model

Motif YAML passes through two validation layers before any expansion runs:

1. **Structural** (`loader.go`) — `utils.StrictUnmarshal` rejects unknown fields. `kind: Motif` and `metadata.name` are required.
2. **Semantic** (`pkg/katalog/motif_validate.go`) — duplicate input names, `required` + `default` conflict, missing `resources` block, and unknown `{{ inputs.* }}` references are all errors.

See [`pkg/katalog/docs/06-motif-validation.md`](../katalog/docs/06-motif-validation.md) for the full validation pipeline.

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand the Motif YAML format and input system | [docs/01-structure.md](docs/01-structure.md) |
| Understand how Motifs are loaded and validated | [docs/02-loading.md](docs/02-loading.md) |
| Understand input binding and resource expansion | [docs/03-expansion.md](docs/03-expansion.md) |
