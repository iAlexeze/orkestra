# 01 — Push and Pull

## Push

`client.Push` validates the directory, stages all files in memory, packs an OCI manifest, and copies everything to the remote repository in one atomic step.

### What "atomic" means here

ORAS pushes blobs first, then the manifest. If the manifest push fails, unreferenced blobs are left in the registry but no tag points to them — the previous version of the tag is preserved. The registry never points to a partial artifact.

### Memory store

The push path uses `memory.New()` instead of a file-backed store. This is intentional: a file store rooted at the pattern directory would write the packed manifest back into `dir`, polluting the source tree. The memory store holds blobs only for the duration of the push call.

### Progress callback

```go
digest, err := client.Push(ctx, ref, dir, func(file string, size int64) {
    fmt.Printf("  → %s (%d bytes)\n", file, size)
})
```

The progress callback fires once per file before any network I/O begins (sizes are read from the in-memory data). Pass `nil` to suppress output.

### Annotations

Every pattern manifest carries a standard set of OCI annotations derived from the pattern metadata:

| Annotation | Source |
|-----------|--------|
| `org.opencontainers.image.title` | `PatternMeta.Name` |
| `org.opencontainers.image.version` | `PatternMeta.Version` |
| `org.opencontainers.image.description` | `PatternMeta.Description` |
| `org.opencontainers.image.authors` | `PatternMeta.Author` |
| `org.opencontainers.image.created` | time of push |
| `io.orkestra.pattern.name` | `PatternMeta.Name` |
| `io.orkestra.pattern.tags` | comma-separated `PatternMeta.Tags` |

`Info` reads these annotations to reconstruct `PatternMeta` without downloading the pattern files.

### Index update

After a successful push, `updateIndex` runs automatically. It fetches `index:latest`, upserts the pattern entry, and pushes the updated index. This is best-effort — a failure is logged to stderr but does not fail the push. See [02-index.md](02-index.md) for details.

---

## Pull

`client.Pull` downloads a pattern into the local cache at `~/.orkestra/registry/<host>/<repo>/<tag>/`. Subsequent pulls for the same ref return the cached directory immediately.

```go
cacheDir, err := client.Pull(ctx, ref, false) // false = use cache if present
cacheDir, err := client.Pull(ctx, ref, true)  // true  = bypass cache, re-pull
```

Pull uses a `file.Store` rooted at the cache directory. ORAS extracts each layer using the `org.opencontainers.image.title` annotation to determine the filename. Files land directly at:

```
~/.orkestra/registry/ghcr.io/orkspace/orkestra-registry/patterns/postgres/v14/
  katalog.yaml
  crd.yaml
  README.md
  cr.yaml
```

### Cache check

`IsCached()` checks for `katalog.yaml` in the expected cache path. A partial pull (interrupted mid-download) is detected because `katalog.yaml` is pushed first — if the file exists, the pull completed. Failed pulls remove the cache directory entirely before returning the error.

### --out flag

The CLI `pull` command supports `--out <dir>` to copy the cached files to a user-specified directory after pulling. This is implemented in the CLI layer (`cmd/cli/registry.go`) as a `copyDir` call after `client.Pull` returns the cache path.

→ Next: [02-index.md](02-index.md)
