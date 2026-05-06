# pkg/buildx

The buildx package abstracts container image building and pushing across Docker, Podman, and Buildah. It is the single place in Orkestra that shells out to a container builder — no other package runs `docker build` directly.

## What lives here

| File | Role |
|------|------|
| `buildx.go` | `Builder` interface, `BuildImage` / `PushImage` public API, builder selection |
| `builders.go` | `DockerBuilder`, `PodmanBuilder`, `BuildahBuilder` — one struct per supported tool |
| `compose_init.go` | `InitConfig`, `AppEntry` — persist `ork doctor init` settings to `.orkestra/.init.ork` for `ork deploy` to read |

## Builder auto-selection

`BuildImage` and `PushImage` call `selectBuilder()`, which walks `AllBuilders` in order and returns the first one whose binary is found in `PATH`:

```
AllBuilders = [DockerBuilder, PodmanBuilder, BuildahBuilder]
```

The user never selects a builder — the first available tool wins. This makes the same `ork deploy` command work in Docker Desktop, Podman Machine, and CI environments that only have Buildah.

## Public API

```go
// Build an image from a directory.
err := buildx.BuildImage(dir, image, buildx.ComposeBuild{}, w)

// Push an already-built image.
err := buildx.PushImage(image, w)
```

`ComposeBuild` controls how the image is built:

```go
type ComposeBuild struct {
    UseCompose  bool   // true → docker compose build
    ComposeFile string // path to docker-compose.yaml (only when UseCompose)
    Dockerfile  string // path to Dockerfile; empty → "Dockerfile" in dir
}
```

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand builder selection and the Builder interface | [docs/01-builders.md](docs/01-builders.md) |
| Understand InitConfig and the .init.ork file format | [docs/02-init-config.md](docs/02-init-config.md) |
