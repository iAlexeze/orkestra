# Composition

Orkestra has two composition primitives: **imports** and **include**. They solve different problems and operate at different scopes.

| Primitive | Scope | Unit | Lifecycle |
|-----------|-------|------|-----------|
| `imports:` | Cross-unit | Separate artifact or file | Independent — versioned, distributed, reused across operators |
| `include:` | Within-unit | Same directory as the declaring file | Shared — cleared at load time, never reaches the runtime |

---

## Aggregators

Every Orkestra kind has a corresponding aggregator kind that composes multiple instances of it into one operation:

| Kind | Aggregator | What it composes |
|------|------------|-----------------|
| `Katalog` | `Komposer` | Multiple Katalogs into one runtime — from local files, OCI registry, or Helm |
| `E2E` | `E2E` (with `imports:`) | Multiple E2E suites into one test run |
| `Simulate` | `Simulate` (with `imports:`) | Multiple Simulate files into one simulation |

`Motif` has no aggregator — it is the reusable unit, not the runner. It is consumed by Katalogs and Komposers via `imports:`.

---

## The two primitives

### imports: — cross-unit reuse

`imports:` brings in content from a separate artifact: a Motif, another Katalog, an OCI image, a Helm chart, an E2E file, a Simulate file. The imported unit has its own path, its own version, and potentially its own author. It may live in a different repository.

Use `imports:` when:
- A concern is shared across multiple operators or teams
- You want to version and distribute the concern independently
- The imported unit has its own lifecycle (update it without touching the importing Katalog)

→ [imports in depth](01-imports.md)

### include: — file splitting

`include:` references a file in the same directory. Both files are loaded together as one logical unit — the include file is not an artifact, has no version, and is invisible after expansion. It is authoring ergonomics, not reuse.

Use `include:` when:
- A Katalog, Motif, E2E, or Simulate file is getting long
- You want to split a block (validation rules, status fields, notes, ops assertions) into a focused file
- Test katalog variants share a common block without a Motif dependency

→ [include in depth](02-include.md)

---

## Reference

| Kind | Schema reference |
|------|-----------------|
| Motif | [schema/01-motif](../../reference/schema/01-motif/index.md) |
| Katalog | [schema/02-katalog](../../reference/schema/02-katalog/01-top-level.md) |
| Komposer | [schema/03-komposer](../../reference/schema/03-komposer/index.md) |
| E2E | [schema/04-e2e](../../reference/schema/04-e2e/index.md) |
| Simulate | [schema/05-simulate](../../reference/schema/05-simulate/index.md) |
