# Operator as library

Orkestra is a Go library. You can import it with a version-pinned `go.mod` entry and write your own entrypoint — full control, no fork needed.

This is not the common path. Most teams use `ork run` directly. This page is for teams who need to embed Orkestra in a larger binary, add a custom webhook, or integrate it with an existing Go application.

---

## The entrypoint

Orkestra's own `main.go` is 12 lines:

```go
package main

import (
    "context"

    "github.com/orkspace/orkestra/cmd/cli"
    "github.com/orkspace/orkestra/pkg/konfig"
    "github.com/orkspace/orkestra/pkg/logger"
    "github.com/orkspace/orkestra/pkg/utils"
)

func main() {
    kfg, err := konfig.Init()
    if err != nil {
        logger.Fatal().AnErr("failed to load configurations", err)
        utils.Exit(err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    cli.Execute(kfg, ctx)
}
```

Copy this as your `main.go`, add a `go.mod` import of `github.com/orkspace/orkestra` at your target version, and `go build` produces a binary with the full Orkestra CLI, runtime, gateway, and Control Center — exactly what `ork` is.

---

## Why you might do this

**Custom webhook**: You need a webhook endpoint that isn't `/validate` or `/mutate`. Add your handler to the Gateway's HTTP server before calling `cli.Execute`.

**Embedding in a larger binary**: Your organisation already has a platform binary. You want Orkestra's reconciler logic available as a subcommand or an internal package, not a separate process.

**Custom konfig source**: `konfig.Init()` reads from environment variables and the standard config path. If you have a custom secret management system, replace the init step with your own loader before passing `kfg` to `cli.Execute`.

**Pre-v1 integration testing**: Pin to a specific commit SHA in `go.mod` while testing unreleased changes before they land in a release.

---

## What you get

Importing and calling `cli.Execute(kfg, ctx)` gives you the complete Orkestra feature set:

- `ork run` — reconciler runtime
- `ork gate` — gateway, webhooks, TLS
- `ork control` — Control Center
- `ork validate`, `ork simulate`, `ork template`, `ork plan` — all CLI commands
- `ork registry` — OCI pattern management

Nothing is removed or locked. You own the binary.

---

## go.mod

```text
require (
    github.com/orkspace/orkestra v0.x.y
)
```

Pin to a release tag for production. Pin to a commit SHA only for development or pre-release testing.
