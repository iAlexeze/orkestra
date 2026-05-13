# 02 — InitConfig and the .init.ork File

`InitConfig` bridges the `ork doctor init` step to `ork doctor deploy`. After init runs, deploy must know how to build each app — which directory, which Dockerfile, and whether to use compose. `.init.ork` is the handshake file.

## The file

`.orkestra/bundle/.init.ork` is a simple key=value file written by `ork doctor init` and read by `ork doctor deploy`. It is never committed (`.orkestra/bundle/` is gitignored, but `.init.ork` sits one level up at `.orkestra/`).

```
useCompose=false
apps=app,frontend
app[0].name=app
app[0].dir=/abs/path/to/app
app[0].dockerfile=
app[1].name=frontend
app[1].dir=/abs/path/to/frontend
app[1].dockerfile=Dockerfile.prod
```

For a compose-based project:

```
useCompose=true
composeFile=/abs/path/to/docker-compose.yaml
```

## Structs

```go
// AppEntry represents one application in a multi-app project.
type AppEntry struct {
    Name       string // app name — used for image tag, CR name, and katalog dir
    Dir        string // absolute path to the app's build context directory
    Dockerfile string // explicit Dockerfile path; empty = "Dockerfile" in Dir
}

// InitConfig is the full configuration persisted in .orkestra/bundle/.init.ork.
type InitConfig struct {
    UseCompose  bool       // true → compose-based build
    ComposeFile string     // path to docker-compose.yaml (only when UseCompose)
    Apps        []AppEntry // one entry per buildable app; empty = legacy single-app
}
```

`Apps` is empty for the single-app (legacy) path. `ork doctor deploy` checks `len(initCfg.Apps) > 0` to choose between single-app and multi-app deploy flows.

## API

```go
// Write for a single-app or compose project (no Apps list)
buildx.WriteInitConfig(projectDir, useCompose, composeFile)

// Write for a multi-app project
buildx.WriteInitConfigFull(projectDir, buildx.InitConfig{
    Apps: []buildx.AppEntry{
        {Name: "app",      Dir: "/abs/app",      Dockerfile: ""},
        {Name: "frontend", Dir: "/abs/frontend",  Dockerfile: "Dockerfile.prod"},
    },
})

// Read at deploy time
cfg, err := buildx.LoadInitConfig(projectDir)

// Remove after successful deploy (optional)
buildx.CleanupInitConfig(projectDir)
```

`LoadInitConfig` returns an empty `InitConfig` (no error) when `.init.ork` does not exist — this is the case when the user runs `ork doctor deploy` without having run `ork doctor init` first (legacy flow).

→ Back to: [README.md](../README.md)
