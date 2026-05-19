# 03 — Reference Resolution and Local Cache

## Reference resolution

`Resolve(input string) (*Ref, error)` converts any user-supplied reference to a fully-qualified OCI `*Ref`. Resolution order:

1. **Full OCI reference** — starts with `oci://` or contains a `.` before the first `/` (e.g. `ghcr.io/myorg/redis:v7`). Used as-is after stripping `oci://`.
2. **`ORK_REGISTRY` env var** — `ORK_REGISTRY=oci://myregistry.io/patterns` + `/name:tag`
3. **Default registry** — `ghcr.io/orkspace/orkestra-registry/patterns/name:tag`

```go
// These all resolve to the same Ref:
registry.Resolve("postgres:v14")
registry.Resolve("oci://ghcr.io/orkspace/orkestra-registry/patterns/postgres:v14")

// With env override:
os.Setenv("ORK_REGISTRY", "oci://myregistry.io/patterns")
registry.Resolve("postgres:v14")  // → myregistry.io/patterns/postgres:v14
```

### Ref fields

```go
type Ref struct {
    Registry   string // "ghcr.io"
    Repository string // "orkspace/orkestra-registry/patterns/postgres"
    Tag        string // "v14"
    Full       string // "ghcr.io/orkspace/orkestra-registry/patterns/postgres:v14"
}

ref.String()    // "oci://ghcr.io/orkspace/orkestra-registry/patterns/postgres:v14"
ref.ShortName() // "postgres:v14"
```

### Default tag

If no tag is specified (e.g. `postgres` with no colon), `parseRef` defaults the tag to `latest`. The `ork registry push` command uses the tag from the command-line argument (`push postgres:v14`), not from the pattern metadata — so users control the tag explicitly.

## Local cache

Pulled patterns are cached under `~/.orkestra/registry/` with the OCI reference mirrored as a directory tree:

```
~/.orkestra/registry/
  ghcr.io/
    orkspace/
      orkestra-registry/
        patterns/
          postgres/
            v14/
              katalog.yaml
              crd.yaml
              README.md
              cr.yaml
```

### Cache path derivation

```go
path, err := ref.CachePath()
// → /home/user/.orkestra/registry/ghcr.io/orkspace/orkestra-registry/patterns/postgres/v14
```

Colons in registry hostnames (for non-standard ports like `localhost:5000`) are replaced with `_` for filesystem safety.

### Cache lookup

`IsCached()` checks for `katalog.yaml` in the expected directory. It does not validate the file's contents — it only confirms the pull completed (since a partial pull is cleaned up by deleting the directory on failure).

```go
if ref.IsCached() {
    cacheDir, _ := ref.CachePath()
    // serve from cache
}
```

### Bypassing the cache

`client.Pull(ctx, ref, true)` re-pulls even when the cache is warm. The CLI exposes this as `ork registry pull postgres:v14 --refresh`.

→ Next: [04-artifacts.md](04-artifacts.md)
