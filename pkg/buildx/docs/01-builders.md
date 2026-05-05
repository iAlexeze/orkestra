# 01 — Builder Selection and the Builder Interface

## The Builder interface

Every supported container tool implements `Builder`:

```go
type Builder interface {
    Name() string
    Available() bool
    Build(dir, image string, compose ComposeBuild, w io.Writer) error
    Push(image string, w io.Writer) error
}
```

`Available()` checks whether the tool's binary exists in `PATH` using `exec.LookPath`. It never makes network calls or shell out — it is safe to call at startup.

## Supported builders

| Struct | Binary | Build command |
|--------|--------|--------------|
| `DockerBuilder` | `docker` | `docker build -t <image> [-f <dockerfile>] <dir>` or `docker compose build` |
| `PodmanBuilder` | `podman` | `podman build -t <image> [-f <dockerfile>] <dir>` or `podman compose build` |
| `BuildahBuilder` | `buildah` | `buildah bud -t <image> [-f <dockerfile>] <dir>` |

Selection order: Docker → Podman → Buildah. When none is available, `BuildImage` returns a descriptive error with install instructions.

## ComposeBuild

`ComposeBuild` is passed to every `Build` call and controls the build strategy:

```go
type ComposeBuild struct {
    UseCompose  bool   // true → use compose build instead of single Dockerfile build
    ComposeFile string // path to docker-compose.yaml; only used when UseCompose is true
    Dockerfile  string // explicit Dockerfile path; empty → default "Dockerfile" in dir
}
```

### Dockerfile selection (non-compose)

When `UseCompose` is false:

1. If `Dockerfile` is set → pass `-f <dockerfile>` to the build command.
2. If `Dockerfile` is empty → the builder uses the default `Dockerfile` in `dir` (no `-f` flag; the tool picks it up automatically).

### Compose builds

When `UseCompose` is true, the builder delegates to the compose build subsystem:

```
DockerBuilder:  docker compose -f <ComposeFile> build
PodmanBuilder:  podman compose -f <ComposeFile> build
BuildahBuilder: buildah does not support compose — falls back to single-file build
```

`BuildahBuilder` logs a warning and falls back to building with the compose file treated as a Dockerfile when `UseCompose` is true, since Buildah has no native compose support.

## Adding a new builder

1. Implement the `Builder` interface in `builders.go`.
2. Append the new builder to `AllBuilders` in `buildx.go`.

```go
var AllBuilders = []Builder{
    DockerBuilder{},
    PodmanBuilder{},
    BuildahBuilder{},
    MyNewBuilder{},  // ← add here
}
```

The selection loop picks it automatically — no other changes needed.

→ Next: [02-init-config.md](02-init-config.md)
