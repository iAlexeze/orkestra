# 04 — Deduplication

CRD names must be unique. A duplicate is always an error — the merger never silently resolves conflicts between two independently-declared CRDs of the same name.

## Two deduplication scopes

### 1. Within one file's source tree (`localSeen`)

`loadKomposer` maintains a `localSeen map[string]string` (name → source label).

```
registry imports  ──┐
file imports       ──┼──► localSeen
helm imports       ──┘

inline spec.crds  (may override a source — valid; triggers mergeCRDEntry)
```

If the same name appears in two different imports (e.g., `file:a.yaml` and `file:b.yaml`), `checkDuplicate` returns an error with both source labels so the user can identify the conflict.

Inline `spec.crds` can override a source CRD with the same name — this is the intended mechanism for local overrides. The inline value is merged onto the source value via `mergeCRDEntry` (not a simple replacement).

Inline over inline is impossible — Go map keys are unique, so the same name cannot appear twice in one `spec.crds` block.

### 2. Across entry-point files (`seen`)

`Merge()` maintains a `seen map[string]string` across all entry-point files. If two entry points (both passed as `--file` flags) declare the same CRD name, the merge fails.

## Error messages

```
duplicate CRD "myresource": defined in "file:a.yaml" and "file:b.yaml" — names must be unique across all imports
```

The source labels follow a consistent `<type>:<identifier>` pattern:
- `file:<url-or-path>`
- `helm:<repo>/<chart>@<version>`
- `registry:<url>` or `registry:<index>`
- `inline:<path>`
