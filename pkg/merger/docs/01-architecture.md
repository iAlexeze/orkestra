# 01 — Architecture

## Pipeline overview

```
CLI / konstructOrkestra
    │
    │  one or more file paths (--file flags or comma-separated)
    ▼
merger.New(paths...)
    │
    │  .Merge()
    ▼
┌─────────────────────────────────────────┐
│  for each entry-point path              │
│                                         │
│  loadKatalogFile(path)                  │
│    │                                    │
│    ├── kind: Katalog → loadKatalog      │
│    │     reads spec.crds, sets          │
│    │     m.security / m.notification /  │
│    │     m.providers as side-effects    │
│    │                                    │
│    └── kind: Komposer → loadKomposer    │
│          resolves sources in order:     │
│          1. registry sources            │
│          2. file sources                │
│          3. helm sources                │
│          4. inline spec.crds            │
│          accumulates top-level fields   │
│          from each source               │
└─────────────────────────────────────────┘
    │
    │  duplicate check across entry-point files
    ▼
m.result  (map[string]CRDEntry, all sources merged)
m.security / m.notification / m.providers (accumulated)

    │
    │  callers consume via:
    ▼
m.ToSpec()         → orktypes.KatalogSpec
m.ToSecurity()     → orktypes.KatalogSecurity
m.ToNotification() → *orktypes.KatalogNotification
m.ToProviders()    → []orktypes.KatalogProviderRequirement
m.Enabled()        → map[string]CRDEntry  (enabled only)
```

## Key invariants

- `Merge()` is called exactly once. All `To*` and query methods panic if called before it.
- The merger is not thread-safe during `Merge()`. After it returns, reads from `Enabled`, `All`, `Get` are safe without synchronisation because the result map is never mutated.
- A Komposer may not reference another Komposer as a source — only Katalog kind is valid inside `sources:`. This prevents unbounded recursion.
- CRD names must be globally unique across all sources. A duplicate is always an error; it is never silently resolved.
