# Orkestra CLI Reference

The `ork` CLI manages the full lifecycle of an Orkestra operator — scaffolding, validation, templating, code generation, runtime execution, and live inspection.

This page provides a brief overview of each command.  
Select a command to view its full documentation.

---

!!! tip "Default file resolution"
    Commands that accept a Katalog or Komposer (`ork run`, `ork validate`, `ork plan`, `ork simulate`, `ork template`) look for `katalog.yaml` first, then `komposer.yaml`. If neither exists the command errors. To use a file with a different name, pass it explicitly: `ork run -f my-katalog.yaml`.

!!! note "Exceptions"
    `ork e2e` looks for `e2e.yaml`. `ork simulate` looks for `simulate.yaml`. Both fall back to the Katalog/Komposer resolution above when their primary file is not found.
---

## Operator Commands Overview

| Command | Description |
|--------|-------------|
| [`ork init`](./01-init.md) | Scaffold a new operator project (dynamic or typed) |
| [`ork validate`](./03-validate.md) | Validate and merge Katalogs or Komposers |
| [`ork plan`](./02-plan.md) | Show what would change if the local Katalog were applied |
| [`ork simulate`](./05-simulate.md) | Simulate operator reconciliation in memory — no cluster needed |
| [`ork gate`](./14-gate.md) | Evaluate admission rules locally against a CR — no cluster needed |
| [`ork token`](./15-token.md) | List, verify, and probe gateway OIDC token entries — no cluster needed |
| [`ork e2e`](./08-e2e.md) | Run declarative end-to-end tests against a real cluster |
| [`ork template`](./04-template.md) | Render the merged, post‑validation Katalog |
| [`ork generate registry`](./generate/registry.md) | Generate runtime registry for typed CRDs and hooks |
| [`ork run`](./07-run.md) | Start the runtime |
| [`ork version`](./version.md) | Print version and build information |

---

## Registry Commands Overview

| Command | Description |
|--------|-------------|
| [`ork push`](./09-push.md) | Publish a pattern to the OCI registry — runs simulate and E2E gates first |
| [`ork pull`](./10-pull.md) | Download a pattern to the local cache |
| [`ork inspect`](./11-inspect.md) | Show metadata and quality signals for a pattern without downloading it |
| [`ork patterns`](./12-patterns.md) | List available patterns in the registry |

---

## Tooling

| Command | Description |
|--------|-------------|
| [`ork proxy`](./proxy.md) | Forward deployed Orkestra component ports to localhost |
| [`ork control`](./control.md) | Start the Orkestra Control Center web UI |
| [`ork diff`](./diff.md) | Show a colorized unified diff between two files |
| [`ork notes`](./notes.md) | Browse and search built-in Katalog template functions |
| [`ork migrate`](./migrate.md) | Rewrite a controller-runtime Reconcile method to the Orkestra constructor signature |
| [`ork create cluster`](./create.md) | Create a local kind cluster for development or testing |
| [`ork delete cluster`](./delete.md) | Delete a local kind cluster |
| [`ork upgrade`](./upgrade.md) | Upgrade the Orkestra CLI to the latest or a specific version |
| [`ork uninstall`](./uninstall.md) | Remove the CLI, Control Center binary, cache, and completions |
