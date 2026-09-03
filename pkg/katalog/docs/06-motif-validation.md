# 06 — Motif Validation and Import Expansion

Motifs are reusable infrastructure templates (databases, caches, brokers) that a Katalog imports instead of writing stateful resource declarations by hand. The katalog package handles two concerns: validating that a Motif file is structurally and semantically correct, and expanding `imports:` blocks into concrete resource declarations at load time.

## Validation pipeline

Motif validation runs in two layers:

### Layer 1 — Structural (pkg/registry/motif/loader.go)

`motif.Load(path)` reads a Motif YAML file and decodes it with `utils.StrictUnmarshal` — the same strict decoder used for Katalog and Komposer YAML. Unknown fields are rejected immediately, before any semantic checks run.

```go
func parse(data []byte) (*orktypes.Motif, error) {
    var m orktypes.Motif
    if err := utils.StrictUnmarshal(data, &m); err != nil {
        return nil, fmt.Errorf("parsing motif: %w", err)
    }
    if !konfig.IsMotifKind(m.Kind) {
        return nil, fmt.Errorf("expected kind: Motif, got: %s", m.Kind)
    }
    if m.Metadata.Name == "" {
        return nil, fmt.Errorf("motif metadata.name is required")
    }
    return &m, nil
}
```

This catches typos in field names early — a field spelled `storageSizes` instead of `storageSize` is an error, not silently ignored.

### Layer 2 — Semantic (pkg/katalog/motif_validate.go)

`ValidateMotif(path)` calls `motif.Load` first, then applies semantic rules:

| Check | Rule |
|-------|------|
| `metadata.name` | Must not be empty |
| `inputs[i].name` | Must not be empty; must be unique |
| `inputs[i].required` + `default` | A required input must not have a default value |
| `resources` | Block must be present |
| Template expressions | `ValidateMotifTemplates` checks that `{{ index .inputs "x" }}` references refer to declared inputs |

`ValidateMotif` returns `[]MotifValidationError` — a slice of `{Path, Message}` pairs. Empty means valid.

```go
errs := katalog.ValidateMotif("/path/to/postgres.motif.yaml")
for _, e := range errs {
    fmt.Println(e)  // "inputs[0]: duplicate input name: image"
}
```

### Import-level validation (pkg/katalog/motif_validate.go)

`ValidateMotifImports(crdName, imports)` validates the `with:` bindings in a Katalog's `operatorBox.imports:` block:

- Every required Motif input must be supplied in `with:`.
- No unknown input names (inputs not declared by the Motif) may appear in `with:`.

This runs after the Motif file passes structural validation, so both layers must pass before import expansion begins.

## Import expansion (pkg/katalog/motif_imports.go)

`expandMotifImports()` is called once during `KomposeRuntimeKatalog`, after the merger produces the enabled CRD map and before validation.

```
operatorBox.imports:
  - motif: postgres          # resolved via pkg/registry/motif loader
    with:
      image: "{{ .data.postgresImage }}"
      volumeSize: "{{ .data.postgresVolumeSize }}"
```

For each import:

1. `motif.LoadImport(&imp)` — resolves the motif reference and loads the file.
2. `motif.Expand(m, imp.With)` — binds `with:` values to the Motif's declared inputs and expands the resource templates.
3. `motif.MergeHookTemplates(entry.OperatorBox.OnReconcile, expanded)` — merges the expanded resources into the CRD's `onReconcile` block.

After all imports are expanded, `entry.OperatorBox.Imports` is set to nil — the Motif contents are inlined and the import list is cleared. The reconciler never sees `imports:` — by the time reconciliation starts, every resource is already in `OnReconcile` as a concrete declaration.

Static `with:` values (plain strings) are substituted immediately during expansion. Dynamic `with:` values (Go template expressions like `{{ .data.image }}`) are carried through as-is and resolved per reconcile cycle by the template resolver.

## Motif YAML structure

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Motif
metadata:
  name: postgres
  version: v1
  description: PostgreSQL stateful database

inputs:
  - name: image
    required: true
    description: Container image (e.g. postgres:16)

  - name: volumeSize
    default: "10Gi"
    description: PVC size

resources:
  statefulSets:
    - name: "{{ .metadata.name }}-db"
      image: "{{ index .inputs \"image\" }}"
      replicas: "1"
      storageSize: "{{ index .inputs \"volumeSize\" }}"
      mountPath: /var/lib/postgresql/data
```

Field reference for `resources`:

| Key | Kubernetes kind |
|-----|----------------|
| `statefulSets` | StatefulSet |
| `services` | Service |
| `configMaps` | ConfigMap |
| `secrets` | Secret |
| `persistentVolumeClaims` | PersistentVolumeClaim |

→ Back to: [README.md](../README.md)
