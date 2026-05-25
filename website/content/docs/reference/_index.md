---
title: "Reference"
date: 2026-05-25
weight: 110
---

Technical reference for the Orkestra runtime, schemas, and CLI.

---

## Schema

Complete field reference for every Orkestra document type.

| Document | Description |
|----------|-------------|
| [Katalog](schema/katalog/) | Top-level operator definition |
| [Komposer](schema/komposer/) | Multi-source composition |
| [CRD Entry](schema/crd-entry/) | Per-CRD configuration |
| [OperatorBox](schema/operatorbox/) | Reconciler and resource declarations |
| [Status](schema/status/) | Status field writes |
| [Validation](schema/validation/) | Admission and reconcile-time validation rules |
| [Mutation](schema/mutation/) | Admission and reconcile-time mutation rules |
| [When Conditions](schema/when-conditions/) | Conditional resource creation |
| [API Types](schema/apitypes/) | `apiTypes` field reference |
| [Conversion](schema/conversion/) | Multi-version CRD conversion |
| [Motif](schema/motif/) | Reusable operator patterns |

---

## CLI

| Command | Description |
|---------|-------------|
| [ork run](cli/run/) | Start the operator runtime |
| [ork validate](cli/validate/) | Validate a Katalog or Komposer |
| [ork template](cli/template/) | Render the merged, resolved Katalog |
| [ork simulate](cli/simulate/) | Simulate reconciliation in memory |
| [ork init](cli/init/) | Scaffold a new operator project |

---

## Where to go next

- **[Schema Reference](schema/katalog/)** — every field in every document type
- **[CLI Reference](cli/run/)** — full flag reference for every command
- **[Typed Operators](../concepts/typed-operators/)** — hooks, constructors, and mixed patterns
