# Reference

Technical reference for the Orkestra runtime, schemas, and CLI.

---

## Schema

Complete field reference for every Orkestra document type.

| Document | Description |
|----------|-------------|
| [Katalog](./schema/katalog.md) | Top-level operator definition |
| [Komposer](./schema/komposer.md) | Multi-source composition |
| [CRD Entry](./schema/crd-entry.md) | Per-CRD configuration |
| [OperatorBox](./schema/operatorbox.md) | Reconciler and resource declarations |
| [Status](./schema/status.md) | Status field writes |
| [Validation](./schema/validation.md) | Admission and reconcile-time validation rules |
| [Mutation](./schema/mutation.md) | Admission and reconcile-time mutation rules |
| [When Conditions](./schema/when-conditions.md) | Conditional resource creation |
| [API Types](./schema/apitypes.md) | `apiTypes` field reference |
| [Conversion](./schema/conversion.md) | Multi-version CRD conversion |
| [Motif](./schema/motif.md) | Reusable operator patterns |

---

## CLI

| Command | Description |
|---------|-------------|
| [ork run](./cli/run.md) | Start the runtime |
| [ork validate](./cli/validate.md) | Validate a Katalog or Komposer |
| [ork template](./cli/template.md) | Render the merged, resolved Katalog |
| [ork simulate](./cli/simulate.md) | Simulate reconciliation in memory |
| [ork init](./cli/init.md) | Scaffold a new operator project |

---

## Where to go next

- **[Schema Reference](./schema/katalog.md)** — every field in every document type
- **[CLI Reference](./cli/run.md)** — full flag reference for every command
- **[Typed Operators](../concepts/typed-operators/index.md)** — hooks, constructors, and mixed patterns
