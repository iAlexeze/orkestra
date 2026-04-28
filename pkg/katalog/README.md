# pkg/katalog

The katalog package is the schema registry for an Orkestra operator. It parses and validates the user's Katalog YAML, enriches each CRD entry with runtime metadata, and exposes the result as a queryable `*Katalog` value that every other subsystem reads from.

Nothing in Orkestra hardcodes a CRD — everything is driven by what the Katalog says.

## What the Katalog holds

| What | Where in code |
|------|--------------|
| Enabled CRD entries (enriched, validated) | `k.enabledCRDs` |
| Katalog metadata (name, version, author, license) | `k.metadata` |
| Security configuration (deletion protection, RBAC) | `k.Security` |
| Conversion rules (`/convert` webhook) | `k.conversionRegistry` |
| Admission rules (`/validate`, `/mutate` webhooks) | `k.admissionRegistry` |
| Dependency DAG | built lazily by `NewDependencyGraph(k)` |

## Boot sequence

```
merger.Merger (merges N katalog YAML files)
    ↓
NewKatalog(merger, konfig)
    → KomposeRuntimeKatalog  — decode YAML, enrich entries
    → ValidateConfig          — field-level, uniqueness, dependency, GVK, defaults
    → updateResourceMapAndReturn — build GVK → type index
    ↓
*Katalog (ready to use)
```

`NewKatalog` calls `utils.Exit` on any validation failure — the operator does not start with a broken Katalog.

## CRD entry lifecycle

Each `CRDEntry` goes through several enrichment phases before it is considered ready:

1. **Parse** — decoded from YAML via the merger.
2. **Enrich** — `EnrichCRDEntry` fills in computed fields (API path, plural, GVK).
3. **Validate** — uniqueness, dependency graph (existence + cycle), reconciler mode.
4. **Defaults** — workers, resync interval, namespace handling, finalizers, description.
5. **Runtime objects** — dynamic mode → `*unstructured.Unstructured` factory; typed mode → `ObjectRegistry` lookup.
6. **Reconcilers** — hooks from `HookRegistry`, constructors from `ReconcilerRegistry`.
7. **Status flags** — `IgnoreStatusPatch` and `IgnoreObservedGeneration` set from `builtins.go`.

After this pipeline, `k.enabledCRDs` contains fully-prepared entries. All runtime code reads from this map — it is never mutated after boot.

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Query the Katalog at runtime | [docs/01-querying.md](docs/01-querying.md) |
| Understand the dependency graph | [docs/02-dependencies.md](docs/02-dependencies.md) |
| Understand deletion protection | [docs/03-deletion-protection.md](docs/03-deletion-protection.md) |
| Understand security defaults | [docs/04-security.md](docs/04-security.md) |
| Understand built-in type handling | [docs/05-builtins.md](docs/05-builtins.md) |
