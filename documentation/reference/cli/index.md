# Orkestra CLI Reference

The `ork` CLI manages the full lifecycle of an Orkestra operator — scaffolding, validation, templating, code generation, runtime execution, and live inspection.

This page provides a brief overview of each command.  
Select a command to view its full documentation.

---

!!! tip Default file resolution
    Commands that accept a Katalog or Komposer (`ork run`, `ork validate`, `ork plan`, `ork simulate`, `ork template`) look for `katalog.yaml` first, then `komposer.yaml`. If neither exists the command errors. To use a file with a different name, pass it explicitly: `ork run -f my-katalog.yaml`.

    `ork e2e` follows the same logic but looks for `e2e.yaml` instead.

---
## Operator Commands Overview

| Command | Description |
|--------|-------------|
| [`ork init`](./01-init.md) | Scaffold a new operator project (dynamic or typed) |
| [`ork validate`](./03-validate.md) | Validate and merge Katalogs or Komposers |
| [`ork plan`](./02-plan.md) | Show what would change if the local Katalog were applied |
| [`ork simulate`](./05-simulate.md) | Simulate operator reconciliation in memory — no cluster needed |
| [`ork e2e`](./08-e2e.md) | Run declarative end-to-end tests against a real cluster |
| [`ork template`](./04-template.md) | Render the merged, post‑validation Katalog |
| [`ork generate registry`](./generate/registry.md) | Generate runtime registry for typed CRDs and hooks |
| [`ork run`](./07-run.md) | Start the runtime |
| [`ork version`](./version.md) | Print version and build information |
