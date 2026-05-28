# The Konductor: How the CLI Became the Orkestrator

*Orkestra Project — May 2026*

---

## A function that waited

```go
// KonduktorKind returns the kind string for a Konduktor document.
func KonduktorKind() string {
    return kindKonductor
}
```

This function lived in the konfig package for a long time without a caller.
The name felt right. The concept felt real. But the implementation it was
pointing at — the thing that would make a Konductor concrete — was not yet
visible.

The obvious candidate was leader election. In any distributed system, one
instance becomes active and the others stand by. Orkestra calls that instance
the active Konductor. It reconciles. The others hold warm caches and wait.
That is a Konductor in the operational sense: the one currently conducting.

But a name for an elected instance is not what the function was reaching for.
Leader election happens at runtime, inside the reconciler. The function was
in the konfig package — the package that initialises before anything else
runs. It was pointing earlier in the lifecycle.

The real Konductor is the CLI.

---

## What the CLI actually does

Orkestra's `main.go` is twelve lines:

```go
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

`konfig.Init()` reads the environment and configuration files and returns a
configuration object. `cli.Execute(kfg, ctx)` takes it and runs whatever
command was invoked.

The twelve lines are not minimal by convention. They are minimal because
everything else belongs somewhere else. The CLI is responsible for one thing:
deciding what runs. KonductRuntime decides how to reconcile. KonductGateway
decides how to protect. The Control Center decides what to display. The CLI
calls the right one, with the right configuration, in the right combination.

That is the Konductor's job. Not reconciling — deciding who reconciles.
Not protecting — deciding who protects. Not displaying — deciding what
displays. Composition.

---

## Two runners, one Konductor

KonductRuntime and KonductGateway are the two major components of Orkestra.
Both are designed to run independently. Both read from the same Katalog.
Neither knows the other exists.

**KonductRuntime** is the reconciler. It watches CRDs, manages workqueues,
runs the reconcile loop, corrects drift, handles deletion, emits events.
When `ork run` is invoked, the CLI calls KonductRuntime. Nothing else.

**KonductGateway** is the protection layer. It runs webhooks — admission,
mutation, conversion, deletion protection. It reads the Katalog to understand
which CRDs to protect and what rules apply. When `ork gate` is invoked, the
CLI calls KonductGateway. Nothing else.

The deployment model follows from this separation. Run them together on one
node or run them on separate nodes — either works, because they share a
Katalog, not state. Run the gateway deployed via Helm in standalone mode,
disable the runtime. Use `ork run` as the runtime companion from your
development machine or from a separate pod. The gateway does not care who
the runtime is. It reads the Katalog. The runtime does not care who the
gateway is. It reads the Katalog. The Katalog is the coordination point.

The CLI composes them as needed. `ork run` is the runtime. `ork gate` is
the gateway. `helm install orkestra --set gateway.enabled=true,runtime.enabled=false`
is a standalone gateway. The Konductor decides the combination.

---

## The Operator as Library

Because the CLI is the composition root, embedding Orkestra in a custom
binary is not a special case. It is the same thing the Orkestra team does
to build `ork`.

Copy the twelve-line `main.go`. Add `github.com/orkspace/orkestra` to your
`go.mod`. Build. The resulting binary has the complete Orkestra feature set —
reconciler, gateway, and every CLI command — because the
CLI is the composition root and it composes them all.

```go
package main

import (
    "context"
    "github.com/orkspace/orkestra/cmd/cli"
    "github.com/orkspace/orkestra/pkg/konfig"
)

func main() {
    kfg, _ := konfig.Init()
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Add your handlers here — before the CLI runs
    registerCustomWebhooks(kfg)

    cli.Execute(kfg, ctx)
}
```

This is not a fork. The Orkestra package is a versioned dependency in your
`go.mod`. When a new version ships, you update the version pin and rebuild.
Your customisation — the `registerCustomWebhooks` call — stays intact. You
never touch Orkestra's source.

This design is possible because the CLI is the composition root. There is
no hidden state, no global initialiser, no registration side effect that
runs at import time. You call `konfig.Init()` and then `cli.Execute()`. In
between, you do whatever your binary needs to do.

---

## The same binary, different behaviour

The CLI's architecture means a single binary can serve completely different
purposes depending on how it is invoked:

```bash
ork run                     # reconciler runtime
ork gate                    # gateway, webhooks
ork validate -f katalog.yaml  # static analysis tool
ork simulate -f katalog.yaml --cr cr.yaml  # in-memory testing
ork plan -f katalog.yaml    # deployment diff
ork registry push pattern:v1 .  # OCI distribution
ork e2e -f e2e.yaml         # end-to-end testing
```

These are not separate tools that happen to be packaged together. They are
the commands of a single Konductor. In production, `ork run` is the command
that runs. In CI, `ork e2e` is the command that runs. In development,
`ork validate` and `ork simulate` are the commands that run. The binary
is the same. The Konductor decides what it does.

This is why the development commands are excluded from the production binary
with a build tag (`//go:build !runtime`). Not because they are different
software — they are the same CLI, the same Konductor. But the production
binary should not carry commands that have no production use. The build tag
enforces this at compile time. The binary is smaller and its surface area
is narrower, but the architecture is identical.

---

## What the Konductor enables

The CLI-as-Konductor design enables three things that would not be possible
otherwise.

**Operator as library.** Any team can import Orkestra as a Go library and
embed it in their platform binary without forking. The CLI is the API surface.
`cli.Execute(kfg, ctx)` is the entry point.

**Composable deployment.** Runtime and Gateway are independent. Any
combination is valid — runtime only, gateway only, both together. The
deployment model is a consequence of the design, not a special configuration.

**Local and CI parity.** The same `ork e2e` command that runs in GitHub
Actions runs on a developer's machine. The same `ork validate` that runs in
pre-commit hooks runs in the CI pipeline. The same binary, the same command,
the same result. There is no equivalent of "works on my machine" because the
Konductor does not distinguish local from CI — it only reads its configuration
and runs its command.

---

## The function's purpose

```go
func KonduktorKind() string {
    return kindKonductor
}
```

The function was always correct. The Konductor is a document kind — something
that can be described, declared, and versioned. The leader election winner
is a runtime event; it has no schema, no version, no file. The CLI is a
document kind: it has a binary, a version, a configuration schema, a set of
commands. It can be described precisely.

The function waited because the understanding was incomplete. The name knew
before the implementation did.