# Reusability and Composition in Orkestra

Reusability is not a single feature in Orkestra — it is the organizing principle the entire framework is built on. Every layer, from the runtime engine to individual field values, follows the same idea: share what is common, declare what varies.

This shapes how Orkestra operators are built, composed, and operated. An operator author does not write infrastructure. A platform team does not duplicate configuration. An organization does not repeat reconcile logic across operators that differ only in configuration or routing.

| Layer | What is shared | What varies |
|-------|---------------|-------------|
| Core | Reconcile engine, gateway, observability | Operator declarations, hook logic |
| Composition | Motif libraries, Katalog packages | Local overrides, environment values |
| Vocabulary | Notes (functions), Profiles (presets) | The expressions and CRDs that use them |
| Parameterisation | Reconcile logic | Args declared per-environment or per-target |
| Reconcile strategies | CRD schema, informer, gateway surface | What happens when a CR arrives |

---

## Pages in this section

| Page | What it covers |
|------|----------------|
| [The Core](01-core.md) | The runtime as shared foundation — one engine for every operator; gateway and Control Center across all of them |
| [Building Blocks](02-building-blocks.md) | Motifs, Katalogs, Komposers — composition, include, test aggregation |
| [User-Defined Reuse](03-user-defined.md) | Notes and Profiles — vocabulary and presets defined once, used everywhere |
| [Args](04-args.md) | Args — the same logic, different behaviour per environment or surface |
| [Reconcile Strategies](05-targets.md) | Reconcile strategies — multiple behaviours from a single CRD via named targets |

---

## Where to go next

- [Orkestra Core](01-core.md)
- [Composition](02-building-blocks.md)
- [Vocabulary](03-user-defined.md)
- [Parameterisation](04-args.md)
- [Reconcile Strategies](05-targets.md)
