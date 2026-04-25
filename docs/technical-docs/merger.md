# Merger

`pkg/merger.Merger` resolves all Katalog and Komposer sources into one validated list of `CRDEntry` values. It is the first thing `KomposeKatalogFromYaml` calls, and its output is what the Katalog uses to start the runtime.

---

## What the Merger does

Given one or more file paths (Katalogs or Komposers), the Merger:

1. Parses each file to determine its kind (`Katalog` or `Komposer`)
2. For Katalogs: reads `spec.crds` directly
3. For Komposers: resolves each source in the `sources` block, then applies inline `spec.crds` overrides
4. Deduplicates CRD entries by name — duplicate names in non-inline sources are an error
5. Returns `[]CRDEntry` in the order they were discovered

The output is the complete, merged set of CRD definitions ready for enrichment and runtime startup.

---

## Source types

### File sources (`sources.files`)

Files can be local paths, remote URLs, or environment variable references:

```yaml
sources:
  files:
    - ./katalogs/website.yaml          # local
    - https://platform.io/katalog.yaml # remote
    - $SECURITY_KATALOG_URL            # env var — resolved at startup
    - url: https://private.io/k.yaml   # authenticated
      auth:
        type: bearer
        fromEnv: MY_TOKEN
```

**Local files** are read with `os.ReadFile`. A missing local file is a fatal error.

**Remote URLs** are fetched with `LoadFileWithAuth`. A 404 is a fatal error. Auth credentials are resolved from environment variables at fetch time — never stored in YAML.

**Environment variable references** (`$VAR_NAME`) are resolved via `os.Getenv`. An unset variable resolves to empty string — the source is skipped with a warning, not a fatal error. This allows optional sources that are only present in certain environments.

!!! warning "Komposer cannot source another Komposer"
    A file declared in `sources.files` must be a Katalog. If it is a Komposer,
    the Merger returns an error. This prevents unbounded composition depth.
    Use `sources.registry` with `useKomposer: true` to load a Komposer from
    a registry pattern.

### Helm sources (`sources.helm`)

The Merger uses the Helm Go library to render charts without a running Tiller or Helm release. The rendered output is parsed for documents with `kind: Katalog`. Any other document kinds are silently ignored.

```yaml
sources:
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 2.1.0
      valueFiles:
        - ./values/production.yaml
```

Repo types:
- **Remote Helm repo** — downloaded and indexed via `repo.NewChartRepository`
- **Local directory** — loaded via `loader.Load`
- **Git URL** (`.git` suffix or `git@`) — shallow cloned to a temp dir, then loaded

The rendered output must contain at least one `kind: Katalog` document. If it contains none, the source fails with a descriptive error.

### Registry sources (`sources.registry`)

Registry sources pull a complete five-file pattern from a Git or OCI registry:

```yaml
sources:
  registry:
    - url: ghcr.io/orkspace/orkestra-registry/postgres@v14
      oci: true
    - url: https://github.com/myorg/registry@main
```

After pulling, the Merger validates the five required files exist and are non-empty (fail fast). Then it loads either `katalog.yaml` (default) or `komposer.yaml` (`useKomposer: true`).

The registry URL is resolved: `src.URL` → `ORK_REGISTRY` env var → `m.registryURL` (set by `konstructOrkestra`).

---

## Merge rules

**Across sources:** CRD names must be unique. Two sources providing the same CRD name is a fatal error. The only exception is an inline `spec.crds` entry — it replaces the source definition silently.

**Inline wins:** `spec.crds` entries on a Komposer are processed last and always win on name conflict with any source. This is the override mechanism — add the CRD name to `spec.crds` with only the fields you want to change, and the rest inherit from the source.

**Within inline:** Duplicate names within `spec.crds` are always a fatal error, even if they would be overrides.

---

## Merger struct

```go
type Merger struct {
    metadata    orktypes.KatalogMeta
    enabled     []orktypes.CRDEntry
    all         []orktypes.CRDEntry
    registryURL string               // set from ORK_REGISTRY by konstructOrkestra
}
```

The `enabled` slice contains CRDs where `enabled: true` (or the default). The `all` slice contains every CRD including disabled ones. The Katalog uses `enabled` for runtime startup and `all` for the health API documentation.

---

## Key functions

### `Merge(paths ...string) ([]CRDEntry, error)`

Entry point. Takes one or more file paths, processes each through `loadKatalogFile`, and returns the deduplicated result. Called from `KomposeKatalogFromYaml`.

### `loadKatalogFile(path string) ([]CRDEntry, error)`

Reads a file, determines its kind, and dispatches to `loadKatalog` or `loadKomposer`.

### `loadKatalog(path string, doc *KatalogFile) ([]CRDEntry, error)`

Reads `spec.crds` from a Katalog document. Validates that no `sources` block is present (sources are a Komposer concern). Returns the CRD entries.

### `loadKomposer(path string, doc *KatalogFile) ([]CRDEntry, error)`

Resolves all sources in order (registry → files → helm), then applies inline `spec.crds` overrides. Source CRDs are merged into `allCRDs`. Inline entries remove and replace their name from `allCRDs`.

### `loadRegistrySource(src RegistrySource) ([]CRDEntry, error)`

Pulls a registry pattern, validates structure, loads the source file. See [Registry Sources](../runtime-manual/concepts/registry-sources/index.md) for the full flow.

---

## Error messages

| Error | Cause |
|---|---|
| `"a Komposer cannot source another Komposer"` | `sources.files` entry is a Komposer |
| `"duplicate CRD %q"` | Same CRD name in two non-inline sources |
| `"CRD with no name"` | A `spec.crds` entry has no `name` field |
| `"chart produced no Katalog templates"` | Helm render produced no `kind: Katalog` docs |
| `"pattern failed structure validation"` | Registry pattern missing required files |
| `"no registry URL configured"` | `sources.registry` entry with no URL and `ORK_REGISTRY` unset |
