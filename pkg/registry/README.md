# pkg/registry

The registry package implements the Orkestra pattern registry — an OCI-based store for operator patterns that can be pushed, pulled, and listed by the `ork registry` CLI commands.

A **pattern** is a directory containing a `katalog.yaml` and `crd.yaml` plus optional documentation and example files. The registry pushes each file as a named OCI layer, tags the manifest, and automatically maintains an index artifact so `ork registry list` works without any server-side catalog support.

## What this package provides

| What | Where in code |
|------|--------------|
| Push/pull/info/list over OCI | `client.go` — `Client` |
| Reference resolution (bare name → full OCI ref) | `resolve.go` — `Resolve`, `Ref` |
| Pattern directory validation and metadata derivation | `pattern.go` — `ValidateDirectory`, `PatternMeta` |
| Index management (auto-updated on push) | `client.go` — `updateIndex`, `fetchIndex`, `pushIndex` |
| Local cache at `~/.orkestra/registry/` | `resolve.go` — `CachePath`, `IsCached` |

## Artifact format

Each pattern is an OCI artifact with media type `application/vnd.orkestra.pattern.v1+tar+gzip`. Files are pushed as individually-typed layers:

```
katalog.yaml   application/vnd.orkestra.katalog.v1+yaml       (required)
crd.yaml       application/vnd.kubernetes.crd.v1+yaml         (required)
README.md      text/markdown                                   (optional)
cr.yaml        application/vnd.kubernetes.cr.v1+yaml          (optional)
pattern.yaml   application/vnd.orkestra.pattern-meta.v1+yaml  (optional)
```

Pattern metadata (name, version, description, tags, author) is stored as OCI manifest annotations and also derived from `katalog.yaml` at push time.

## Push flow

```
ValidateDirectory(dir)           — check required files, derive PatternMeta
merger.New(katalog.yaml).Merge() — parse and validate katalog semantics   [CLI]
katalog.ValidateConfig(kfg)      — full field/GVK/dependency validation   [CLI]
validateCRDFile(crd.yaml)        — structural CRD YAML check               [CLI]
    ↓
client.Push(ctx, ref, dir)
    → read each file into memory store (nothing written to dir)
    → oras.Pack  — manifest with annotations
    → store.Tag  — register manifest under ref.Tag
    → oras.Copy  — push all blobs + manifest to remote
    → updateIndex — upsert pattern into index:latest (best-effort)
```

## Index artifact

`ork registry list` reads a single `index:latest` artifact from the registry namespace root (`ghcr.io/orkspace/orkestra-registry/patterns/index:latest`). This artifact is a JSON blob containing a `PatternIndex`.

`Push` automatically updates the index after every successful push: it fetches the existing index, upserts the new pattern entry, and pushes the updated index back. There is no separate reindex step. If the index push fails it is logged as a warning — the pattern push is not rolled back.

## Reference resolution

Bare references are resolved in this order:

1. Full OCI reference (`oci://host/repo:tag`) — used as-is after stripping `oci://`
2. `ORKESTRA_REGISTRY` environment variable + `/name:tag`
3. Default: `ghcr.io/orkspace/orkestra-registry/patterns/name:tag`

```go
ref, err := registry.Resolve("postgres:v14")
// → ghcr.io/orkspace/orkestra-registry/patterns/postgres:v14
```

## Authentication

Reads `~/.docker/config.json` via `credentials.NewStoreFromDocker`. Run `docker login ghcr.io` before pushing. No separate `ork registry login` command.

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand push and pull end-to-end | [docs/01-push-pull.md](docs/01-push-pull.md) |
| Understand the index and how list works | [docs/02-index.md](docs/02-index.md) |
| Understand reference resolution and the cache | [docs/03-resolve-cache.md](docs/03-resolve-cache.md) |
| Add a new pattern or extend the artifact format | [docs/04-patterns.md](docs/04-patterns.md) |
