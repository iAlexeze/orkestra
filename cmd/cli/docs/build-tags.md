# Build Tags

Orkestra uses Go build tags to control which sub-commands are compiled into a binary. This keeps each deployable artifact lean and makes the separation between runtime and gateway explicit at the build level.

## Tags

| Tag | What it includes | Typical use |
|-----|-----------------|-------------|
| *(none)* | `!runtime` files — all CLI commands, dev mode | `make ork` — local development |
| `runtime` | `run.go` — the `ork run` sub-command | Runtime container image |
| `gateway` | `gateway.go` — the `ork gateway` sub-command | Gateway container image |
| `runtime,gateway` | both sub-commands | Combined image (v1 default) |

## File naming convention

Each build-tag variant of a file follows this pattern:

```
<command>.go        //go:build <tag>        production / container-only
<command>_dev.go    //go:build !runtime     local development
```

`ork gateway` has no `_dev.go` counterpart. The gateway only runs inside a Kubernetes pod — it exits immediately if invoked outside one. There is nothing useful to dev-test locally.

## Building

```bash
# Local dev binary (make ork) — all commands, no runtime/gateway tag
go build ./cmd/ork

# Runtime image only — ork run
go build -tags runtime ./cmd/ork

# Gateway image only — ork gateway
go build -tags gateway ./cmd/ork

# Combined image — ork run + ork gateway (v1 production)
go build -tags "runtime gateway" ./cmd/ork
```

The Makefile targets:

```makefile
make ork              # dev build (no tags)
make ork-runtime      # -tags runtime
make ork-gateway      # -tags gateway
make ork-production   # -tags "runtime gateway"
```

## Why two tags instead of one

Using separate `runtime` and `gateway` tags means:

- A pure gateway image never compiles the reconciler stack (informers, kordinator, queues).
- A pure runtime image never compiles the webhook server.
- Both can be built from the same repository and the same Dockerfile with a `--build-arg TAG=...`.
- Adding a third process later (e.g. an API gateway) follows the same pattern: new tag, new file, no changes to existing files.

## The `!runtime` dev pattern

All CLI commands that are useful locally (validate, generate, deploy, doctor, simulate, etc.) live in files tagged `//go:build !runtime`. They are compiled in by default and excluded only from production builds.

The production binary (`-tags "runtime gateway"`) contains only:
- `ork run` — the reconcile loop
- `ork gateway` — TLS + webhooks

Everything else (`ork generate`, `ork doctor`, `ork validate`, etc.) is for the developer and is not shipped in the container image.

→ See [cmd/internal/docs/01-overview.md](../../internal/docs/01-overview.md) for the runtime/gateway architectural split.
