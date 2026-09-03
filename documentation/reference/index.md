# Reference

Technical reference for the Orkestra runtime, schemas, and CLI.

!!! tip "Writing a resource declaration?"
    See the [Resources reference](./schema/06-resources/index.md) — one page per Kubernetes built-in and custom resource kind, with every field and a worked YAML example.

---

## Schema

Complete field reference for every Orkestra Pattern.

| Pattern | Description |
|----------|-------------|
| [Motif](./schema/01-motif/index.md) | Reusable resource primitive — parameterised inputs, no standalone runtime |
| [Katalog](./schema/02-katalog/index.md) | Top-level operator definition — CRDs, resources, status, admission rules |
| [Komposer](./schema/03-komposer/index.md) | Compose multiple Katalogs from files, Helm, or OCI registries |
| [E2E](./schema/04-e2e/index.md) | Declarative end-to-end test for a Katalog |
| [Simulate](./schema/05-simulate/index.md) | In-memory reconciler verification — no cluster |
| [Resources](./schema/06-resources/index.md) | Every Kubernetes built-in and custom resource declarable under `onCreate`/`onReconcile`/`onDelete`, one page per kind |

---

## CLI

| Command | Description |
|---------|-------------|
| [ork run](./cli/07-run.md) | Start the runtime |
| [ork validate](./cli/03-validate.md) | Validate a Katalog or Komposer |
| [ork template](./cli/04-template.md) | Render the merged, resolved Katalog |
| [ork simulate](./cli/05-simulate.md) | Simulate reconciliation in memory |
| [ork init](./cli/01-init.md) | Scaffold a new operator project |

---

## Where to go next

- **[Schema Reference](./schema/index.md)** — every field in every Pattern
- **[CLI Reference](./cli/index.md)** — full flag reference for every command
- **[Typed Operators](../concepts/typed-operators/index.md)** — hooks, constructors, and mixed patterns
