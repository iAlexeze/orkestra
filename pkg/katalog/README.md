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
2. **crdFile population** — if `crdFile:` is declared, `populateAPITypesFromCRDFile` reads the CRD YAML and fills `APITypes` (group, version, kind, plural) from it. `crdFile` is the source of truth and overwrites any inline `apiTypes:` block. Typed-mode fields (Object, List, Alias, Location) are preserved from the inline declaration if present.
3. **Enrich** — `EnrichCRDEntry` fills in computed fields (API path, plural, GVK). For built-in Kubernetes kinds (Deployment, ConfigMap, etc.), group/version/plural are looked up from `builtins.go`.
4. **Validate** — uniqueness, dependency graph (existence + cycle), reconciler mode.
5. **Defaults** — workers, resync interval, namespace handling, finalizers, description.
6. **Runtime objects** — dynamic mode → `*unstructured.Unstructured` factory; typed mode → `ObjectRegistry` lookup.
7. **Reconcilers** — hooks from `HookRegistry`, constructors from `ReconcilerRegistry`.
8. **Status flags** — `IgnoreStatusPatch` and `IgnoreObservedGeneration` set from `builtins.go`.

After this pipeline, `k.enabledCRDs` contains fully-prepared entries. All runtime code reads from this map — it is never mutated after boot.

Step 2 (crdFile population) runs inside `KomposeRuntimeKatalog`, before `ValidateConfig`. By the time any validation runs, `APITypes` is fully populated regardless of whether the user declared `apiTypes:` or `crdFile:` or both.

## Motif support

The katalog package also handles **Motif** imports — reusable infrastructure templates (databases, caches, message brokers) that a Katalog can import instead of writing all stateful resource declarations by hand.

| File | Role |
|------|------|
| `motif_imports.go` | `ResolveMotifImports` — expands `imports:` blocks into concrete resource declarations |
| `motif_validate.go` | `ValidateMotif`, `ValidateMotifImports` — structural and semantic validation of Motif YAML |

Motif YAML is loaded by `pkg/motif/loader.go` using `utils.StrictUnmarshal` (same strict decoder as Katalog/Komposer) and then validated by `motif_validate.go` before any import expansion runs.

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Query the Katalog at runtime | [docs/01-querying.md](docs/01-querying.md) |
| Understand the dependency graph | [docs/02-dependencies.md](docs/02-dependencies.md) |
| Understand deletion protection | [docs/03-deletion-protection.md](docs/03-deletion-protection.md) |
| Understand security defaults | [docs/04-security.md](docs/04-security.md) |
| Understand built-in type handling | [docs/05-builtins.md](docs/05-builtins.md) |
| Understand Motif validation and import expansion | [docs/06-motif-validation.md](docs/06-motif-validation.md) |
| Understand crdFile — auto-populating APITypes from a CRD YAML | [docs/07-crd-file.md](docs/07-crd-file.md) |
