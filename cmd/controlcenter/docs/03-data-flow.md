# 03 — Data Flow

Data travels in one direction: Orkestra runtime → `Client` → `ControlCenter` in-memory state → view-model struct → Go HTML template → browser.

## Orkestra runtime API endpoints

The Control Center calls these endpoints on each connected Orkestra runtime:

| Endpoint | Go type | Used by |
|----------|---------|---------|
| `GET /katalog` | `KatalogResponse` | Background fetch loop (every instance, every tick) |
| `GET /katalog/{crd}/health` | `CRDHealth` | `handleCRDDetail`, `handleCRDDocs` |
| `GET /katalog/{crd}` | `CRDInfo` | `handleCRDDetail`, `handleCRDDocs` |
| `GET /katalog/{crd}/cr` | `CRListResponse` | `handleCRList` |
| `GET /katalog/{crd}/cr/{ns}/{name}` | `CRDetailResponse` | `handleCRDetail` |
| `GET /katalog/{crd}/cr/{ns}/{name}/events` | `CREventsResponse` | `handleCRDetail` |
| `GET /katalog/raw` | raw JSON | `/katalog/{name}/raw` proxy |
| `GET /katalog/{crd}/raw` | raw JSON | `/katalog/{name}/crd/{crd}/raw` proxy |

All calls go through `cc/client.go`. The generic helper `getJSON[T]` handles JSON decoding and error wrapping.

## Periodic fetch

`FetchKatalog` is called for every instance on every tick. The result is stored directly on `inst.Katalog` (`*KatalogResponse`) under the write lock. This is the only field the background loop ever writes.

```
KatalogResponse
  .Name, .Description, .Version, .Author, .License
  .Healthy, .OrkReady, .DegradedReason
  .CRDs          []CRDSummary   ← high-level health per CRD
  .StatusCounts                 ← healthy/started/pending/degraded counts
  .RuntimeVersion               ← version string from the runtime
```

The `CRDSummary` slice is what drives the Katalog panel and the index. It does not include deep detail — for that, the user navigates to a CRD page, which triggers a fresh `FetchCRDDetail` call at request time.

## On-demand fetch

`FetchCRDDetail` merges two runtime calls — `GET /katalog/{crd}/health` and `GET /katalog/{crd}` — into a single `*CRDDetail`. This merged struct is the view-model for both the CRD dashboard (`crd.html`) and the CRD docs page (`crd_docs.html`).

```
CRDDetail
  ← CRDHealth:  state, workers, queue depth, error counts, dependencies
  ← CRDInfo:    GVK, GVR, scope, mode, workers config, RBAC, webhooks, providers
```

If the runtime is unreachable, `handleCRDDetail` renders a degraded view with `State: "offline"` rather than returning an error page. `handleCRDDocs` similarly falls back gracefully.

## View models

Each handler builds a typed view-model struct and passes it to `renderTemplate`. The struct name matches the template name:

| Handler | View-model | Template |
|---------|-----------|----------|
| `handleIndex` | `IndexData` | `index.html` |
| `handleKatalogPanel` | `KatalogData` | `katalog.html` |
| `handleCRDDetail` | `map[string]interface{}` | `crd.html` |
| `handleCRList` | `CRListView` | `cr_list.html` |
| `handleCRDetail` | `CRDetailView` | `cr_detail.html` |
| `handleDocsLanding` | `DocsLandingData` | `docs.html` |
| `handleCRDDocs` | `CRDDocsData` | `crd_docs.html` |

`CRDDocsData` adds `Has*` boolean flags (`HasAdmission`, `HasConversion`, `HasProtection`, `HasRBAC`, `HasAutoscaler`, `HasRollback`, `HasProviders`) computed from the fetched `*CRDDetail`. Templates use these to conditionally render sections so missing features produce no empty headers.

## Snapshot API

`GET /api/snapshot` returns a lightweight JSON summary (no per-CRD detail) used by the index page JS to do partial DOM updates after each SSE `update` event. This avoids a full page reload while keeping stats current.

→ Next: [04-ui-design.md](04-ui-design.md)
