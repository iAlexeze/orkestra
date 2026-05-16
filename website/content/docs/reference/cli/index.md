---
title: "Orkestra CLI Reference"
weight: 27
---

# Orkestra CLI Reference

The `ork` CLI manages the full lifecycle of an Orkestra operator — scaffolding, validation, templating, code generation, runtime execution, and live inspection.

This page provides a brief overview of each command.  
Select a command to view its full documentation.

---

## Developer CLI

Commands for deploying your own project to Kubernetes — no operator knowledge required.

| Command | Description |
|---------|-------------|
| [`ork doctor`](./developer/doctor/) | Examine the project and generate `.orkestra/` configuration |
| [`ork doctor deploy`](./developer/deploy/) | Build, push, and deploy to the cluster |
| [`ork doctor deploy rollback`](./developer/rollback/) | Instantly restore the previous image |

→ [Developer CLI overview](./developer/__index/)

---

## Operator Commands Overview

| Command | Description |
|--------|-------------|
| [`ork init`](./init/) | Scaffold a new operator project (dynamic or typed) |
| [`ork validate`](./validate/) | Validate and merge Katalogs or Komposers |
| [`ork plan`](./plan/) | Show what would change if the local Katalog were applied |
| [`ork simulate`](./simulate/) | Simulate operator reconciliation in memory — no cluster needed |
| [`ork e2e`](./e2e/) | Run declarative end-to-end tests against a real cluster |
| [`ork template`](./template/) | Render the merged, post‑validation Katalog |
| [`ork generate registry`](./generate-runtime/) | Generate runtime registry for typed CRDs and hooks |
| [`ork run`](./run/) | Start the operator runtime |
| [`ork reconcile`](./reconcile/) | Trigger reconciliation for one or all CRs |
| [`ork version`](./version/) | Print version and build information |

---

## Typical Workflows

- [Zero‑code dynamic operator](./init.md#dynamic-project-default)
- [Typed operator with Go hooks](./init.md#typed-project)
- [Komposer with multiple sources](./validate.md#examples)
- [CI pipeline](./validate.md#examples)

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Validation error, merge error, or runtime failure |
| `2` | Invalid flags or arguments |

---
## Related Documentation

- [Katalog Schema](../katalog-schema/)
- [Komposer Schema](../komposer-schema/)
- [Operator Runtime](../runtime/)
