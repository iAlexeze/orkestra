# pkg/registry

The registry is Orkestra's artifact store and testing layer. It owns OCI push/pull for operator patterns and motifs, the simulate harness for in-memory reconcile testing, the e2e runner for cluster integration tests, and motif expansion.

| Sub-package | Responsibility |
|-------------|----------------|
| [simulate/](simulate/README.md) | In-memory reconcile harness — `ork simulate` |
| [e2e/](e2e/README.md)          | Cluster integration test runner — `ork e2e` |
| [motif/](motif/README.md)      | Motif loading, validation, and expansion — `ork push`/`ork pull` |

OCI push/pull, reference resolution, and artifact kind detection live in this package directly.

Artifact kind is detected automatically from the primary YAML file (`kind: Katalog` → pattern, `kind: Motif` → motif). No separate code path per kind is needed — adding a new artifact kind is a one-line entry in `artifactSpecs` in `artifact.go`.

## What this package provides

| What | Where in code |
|------|--------------|
| Generic artifact kind detection and validation | `artifact.go` — `DetectKind`, `ValidateArtifactDirectory`, `LoadArtifactMeta` |
| Push/pull/info/list over OCI | `client.go` — `Client` |
| Reference resolution (bare name → full OCI ref) | `resolve.go` — `Resolve`, `ResolveForKind`, `Ref` |
| Pattern directory validation (legacy) | `pattern.go` — `PatternMeta`, `PatternIndex` |
| Index management (auto-updated on push) | `client.go` — `updateIndex`, `fetchIndex`, `pushIndex` |
| Local cache at `~/.orkestra/registry/` | `resolve.go` — `CachePath`, `IsCached` |
| File and media-type constants | `constant.go` |

## Artifact kinds

### Pattern (kind: Katalog)

A Katalog-based operator pattern. Required files: `katalog.yaml`, `crd.yaml`.

```
my-operator/
  katalog.yaml   application/vnd.orkestra.katalog.v1+yaml       (required)
  crd.yaml       application/vnd.kubernetes.crd.v1+yaml         (required)
  README.md      text/markdown                                   (optional)
  cr.yaml        application/vnd.kubernetes.cr.v1+yaml          (optional)
```

Manifest media type: `application/vnd.orkestra.pattern.v1+tar+gzip`  
Default registry: `ghcr.io/orkspace/orkestra-registry/patterns`  
Override: `ORK_REGISTRY`

### Motif (kind: Motif)

A reusable resource primitive (stateful service, shared infrastructure). Required file: `motif.yaml`.

```
my-motif/
  motif.yaml     application/vnd.orkestra.motif.v1+yaml         (required)
  README.md      text/markdown                                   (optional)
  example/       (directory)                                     (optional)
```

Manifest media type: `application/vnd.orkestra.motif.v1+tar+gzip`  
Default registry: `ghcr.io/orkspace/orkestra-motifs`  
Override: `ORK_MOTIFS_REGISTRY`

## Push flow

```
ValidateArtifactDirectory(dir)   — detect kind, check required files, list all files
LoadArtifactMeta(dir, spec)      — read name/version/description from primary YAML
    ↓
client.Push(ctx, ref, dir)
    → read each file into memory store (nothing written back to dir)
    → oras.Pack  — manifest with kind-specific media type and artifact annotations
    → store.Tag  — register manifest under ref.Tag
    → oras.Copy  — push all blobs + manifest to remote
    → updateIndex — upsert entry into index:latest (best-effort)
```

Additional validation layers run in the CLI before `client.Push` is called (see [docs/04-patterns.md](docs/04-patterns.md)).

## Annotations

Every manifest carries standard OCI annotations derived from the artifact's primary YAML:

| Annotation | Source |
|-----------|--------|
| `org.opencontainers.image.title` | `ArtifactMeta.Name` |
| `org.opencontainers.image.version` | `ArtifactMeta.Version` |
| `org.opencontainers.image.description` | `ArtifactMeta.Description` |
| `org.opencontainers.image.authors` | `ArtifactMeta.Author` |
| `org.opencontainers.image.created` | time of push |
| `io.orkestra.artifact.kind` | `ArtifactMeta.Kind` (`Katalog` or `Motif`) |
| `io.orkestra.artifact.name` | `ArtifactMeta.Name` |
| `io.orkestra.artifact.version` | `ArtifactMeta.Version` |
| `io.orkestra.artifact.author` | `ArtifactMeta.Author` |
| `io.orkestra.artifact.license` | `ArtifactMeta.License` |
| `io.orkestra.artifact.tags` | comma-separated `ArtifactMeta.Tags` |

`Info` reads these annotations to reconstruct metadata without downloading any files. The legacy `io.orkestra.pattern.*` keys are still read for backward compatibility with artifacts pushed before the generic artifact layer.

## Index artifact

`ork patterns` reads a single `index:latest` artifact from the registry namespace root. Each registry has its own index:

- Patterns: `ghcr.io/orkspace/orkestra-registry/patterns/index:latest`
- Motifs: `ghcr.io/orkspace/orkestra-motifs/index:latest`

`Push` automatically updates the index after every successful push. If the index push fails it is logged as a warning — the artifact push is not rolled back.

## Reference resolution

`Resolve` routes bare names to the pattern registry (for backward compatibility). `ResolveForKind` picks the correct registry based on detected kind:

```go
// Pattern
ref, err := registry.ResolveForKind("postgres:v1", registry.KatalogKind)
// → ghcr.io/orkspace/orkestra-registry/patterns/postgres:v1

// Motif
ref, err := registry.ResolveForKind("redis:v7", registry.MotifKind)
// → ghcr.io/orkspace/orkestra-motifs/redis:v7

// Full OCI ref — used as-is regardless of kind
ref, err := registry.ResolveForKind("oci://ghcr.io/myorg/motifs/redis:v7", registry.MotifKind)
// → ghcr.io/myorg/motifs/redis:v7
```

## Authentication

Reads `~/.docker/config.json` via `credentials.NewStoreFromDocker`. Run `docker login ghcr.io` before pushing. No separate `ork login` command — use `docker login`.

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand push and pull end-to-end | [docs/01-push-pull.md](docs/01-push-pull.md) |
| Understand the index and how list works | [docs/02-index.md](docs/02-index.md) |
| Understand reference resolution and the cache | [docs/03-resolve-cache.md](docs/03-resolve-cache.md) |
| Publish a pattern or motif / extend the artifact format | [docs/04-artifacts.md](docs/04-artifacts.md) |
