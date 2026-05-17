# Orkestra CLI Reference

The `ork` CLI manages the full lifecycle of an Orkestra operator — scaffolding, validation, templating, code generation, runtime execution, and live inspection.

This page provides a brief overview of each command.  
Select a command to view its full documentation.

---

## Operator Commands Overview

| Command | Description |
|--------|-------------|
| [`ork init`](./init.md) | Scaffold a new operator project (dynamic or typed) |
| [`ork validate`](./validate.md) | Validate and merge Katalogs or Komposers |
| [`ork plan`](./plan.md) | Show what would change if the local Katalog were applied |
| [`ork simulate`](./simulate.md) | Simulate operator reconciliation in memory — no cluster needed |
| [`ork e2e`](./e2e.md) | Run declarative end-to-end tests against a real cluster |
| [`ork template`](./template.md) | Render the merged, post‑validation Katalog |
| [`ork generate registry`](./generate/registry.md) | Generate runtime registry for typed CRDs and hooks |
| [`ork run`](./run.md) | Start the operator runtime |
| [`ork reconcile`](./reconcile.md) | Trigger reconciliation for one or all CRs |
| [`ork version`](./version.md) | Print version and build information |
