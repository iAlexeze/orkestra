# cmd/cli

CLI command definitions for the `ork` binary. Each file registers one command or sub-command with the root Cobra command.

## Entry points

| Command | File | What it does |
|---------|------|-------------|
| `ork run` | `run.go` / `run_dev.go` | Start the runtime (reconcile loop) |
| `ork gate` | `gateway.go` | Start the gateway (TLS + webhooks, cluster-only) |
| `ork generate` | `generate.go` | Generate RBAC, bundles, ConfigMaps, CRDs, docs |
| `ork validate` | `validate.go` | Validate a Katalog file |
| `ork deploy` | `deploy.go` | Deploy an operator via `ork doctor` |
| `ork plan` | `plan.go` | Dry-run a Katalog against a live cluster |

For the full command reference see [documentation/reference/cli](../../documentation/reference/cli).

## Design docs

- [build-tags.md](docs/build-tags.md) — how `//go:build runtime` and `//go:build gateway` control which sub-commands ship in each image
