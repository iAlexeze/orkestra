# 02 — Loading and Validation

## Load vs LoadImport

`Load(path string)` is the simple path for loading a local file:

```go
m, err := motif.Load("./postgres/motif.yaml")
```

`LoadImport(imp *orktypes.MotifImport)` handles the full resolution chain — local file, OCI artifact, or Git registry — using the same semantics as `RegistrySource` in a Komposer:

```go
m, err := motif.LoadImport(&orktypes.MotifImport{
    Motif: "ghcr.io/orkspace/orkestra-registry/postgres@v16",
    OCI:   true,
})
```

Reference formats:

| Format | Example | Resolved by |
|--------|---------|-------------|
| Local file | `./postgres/motif.yaml` | `os.ReadFile` |
| OCI artifact | `ghcr.io/org/repo/postgres@v16` | `merger.PullMotifToDir` (OCI pull) |
| Git repo | `https://github.com/myorg/postgres-motif@main` | `merger.PullMotifToDir` (Git clone) |

The `@version` suffix overrides the `imp.Version` field. Default version: `latest` for OCI, `main` for Git.

## Strict structural validation

Both `Load` and `LoadImport` call the internal `parse` function, which uses `utils.StrictUnmarshal`:

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

`StrictUnmarshal` is a YAML decoder with `KnownFields(true)` — it rejects unknown fields immediately. This is the same decoder used for Katalog and Komposer YAML, so Motif files have the same strictness guarantee as the rest of the config surface.

### Why strict?

Without strict decoding, a typo like `storageSizes` instead of `storageSize` would be silently ignored — the field would be zero-valued and the Motif would appear to work but produce wrong resources. Strict decoding turns this silent failure into an explicit error at load time.

## Semantic validation

Structural validation alone does not catch semantic errors. `pkg/katalog/motif_validate.go` adds a second layer:

```go
errs := katalog.ValidateMotif("/path/to/postgres.motif.yaml")
```

See [`pkg/katalog/docs/06-motif-validation.md`](../../katalog/docs/06-motif-validation.md) for the full list of semantic checks.

→ Next: [03-expansion.md](03-expansion.md)
