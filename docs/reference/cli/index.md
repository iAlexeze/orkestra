# Orkestra CLI Reference

The `ork` CLI manages the full lifecycle of an Orkestra operator — scaffolding, validation, templating, code generation, runtime execution, and live inspection.

This page provides a brief overview of each command.  
Select a command to view its full documentation.

---

## Commands Overview

| Command | Description |
|--------|-------------|
| [`ork init`](./init.md) | Scaffold a new operator project (dynamic or typed) |
| [`ork validate`](./validate.md) | Validate and merge Katalogs or Komposers |
| [`ork template`](./template.md) | Render the merged, post‑validation Katalog |
| [`ork generate runtime`](./generate-runtime.md) | Generate runtime wiring for typed CRDs and hooks |
| [`ork run`](./run.md) | Start the operator runtime |
| [`ork status`](./status.md) | Show health and reconcile statistics |
| [`ork get`](./get.md) | List CRs of a given CRD |
| [`ork describe`](./describe.md) | Show spec, status, and events for a CR |
| [`ork reconcile`](./reconcile.md) | Trigger reconciliation for one or all CRs |
| [`ork events`](./events.md) | Show Kubernetes events for a CRD or CR |
| [`ork version`](./version.md) | Print version and build information |

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

- [Katalog Schema](../katalog-schema.md)
- [Komposer Schema](../komposer-schema.md)
- [Operator Runtime](../runtime.md)
