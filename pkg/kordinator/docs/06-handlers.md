# 06 — HTTP handlers

All three runtime introspection handlers live in `crd_health_handers.go`. They are registered before the operator starts so they are available from the first moment the health server is bound, even before informers have synced.

## BuildCRDHealthHandler

```go
func BuildCRDHealthHandler(
    crd orktypes.CRDEntry,
    kfg *konfig.Konfig,
    inf cache.SharedIndexInformer,
    h   *CRDHealth,
) http.HandlerFunc
```

Serves `GET /katalog/{crd}/health`. Returns the live health summary for a single CRD. Used by Kubernetes liveness probes, external health checks, and `ork status`.

The HTTP status code and the `state` string are derived from the CRD's atomic health fields:

| State | Condition | HTTP |
|---|---|---|
| `not started` | `!started && !pending` | 503 |
| `pending` | `pending` | 200 |
| `started` | `started && !healthy` | 200 |
| `healthy` | `started && healthy` | 200 |
| `degraded` | `degraded` | 503 |

Example response:

```json
{
  "name": "website",
  "state": "healthy",
  "healthy": true,
  "started": true,
  "queueDepth": 0,
  "errorRate": 0.02,
  "consecutiveFails": 0,
  "totalReconciles": 314,
  "resourceCount": 12,
  "lastReconcile": "2026-04-14T09:11:03Z",
  "lastError": ""
}
```

`resourceCount` comes from `len(inf.GetStore().List())` — the informer's local in-memory cache, updated in real time by the watch stream. No API call is made.

## BuildCRDInfoHandler

```go
func BuildCRDInfoHandler(
    crd         orktypes.CRDEntry,
    kfg         *konfig.Konfig,
    inf         cache.SharedIndexInformer,
    h           *CRDHealth,
    convStats   *health.ConversionStats,
    admStats    *health.AdmissionStats,
    protStats   *health.ProtectionStats,
    isProtected bool,
    provStats   *health.ProviderStats,
) http.HandlerFunc
```

Serves `GET /katalog/{crd}`. Returns the full CRD detail. The response is assembled on every request from live atomic reads — no caching.

**Provider section.** When the CRD declares provider blocks, the response includes a `providers` array. Each entry merges two sources:

- Static metadata from `crd.OperatorBox.ProviderBlocks` — the declared provider name and the list of kinds from the Katalog YAML
- Runtime stats from `provStats.GetSnapshot()` — total calls, error count, and error rate since startup

```json
"providers": [
  {
    "name": "postgres",
    "kinds": ["database", "role"],
    "total": 88,
    "errors": 2,
    "errorRate": 0.023
  }
]
```

When `provStats` is nil (CRD has no provider blocks), the `providers` field is omitted entirely.

**Protection section.** When `protStats` is nil, a zero `ProtectionStatsResponse` is returned with `enabled` set to `isProtected`. This avoids a nil check in every caller.

**Dependency section.** The `dependencies` map from `CRDHealth` is included verbatim — it is kept fresh by the `dependencyHealthChecker` goroutine (see [04 — Self-healing](04-self-healing.md)).

## BuildKatalogHandler

```go
func BuildKatalogHandler(
    kat       *katalog.Katalog,
    kfg       *konfig.Konfig,
    registry  *ResourceKatalog,
    healthMap map[string]*CRDHealth,
) http.HandlerFunc
```

Serves `GET /katalog`. Aggregates all CRDs into one response — the source of truth for `ork status` and the Control Center dashboard.

The response includes:

- `healthy` — true only when every CRD is in a healthy or started state
- `dependencyGraph` — the full ordered graph from the Katalog
- `crds` — a summary row per CRD

Each summary row includes `providerCount` (the number of declared provider blocks) when non-zero:

```json
{
  "name": "website",
  "state": "healthy",
  "resourceCount": 12,
  "workers": 3,
  "queueDepth": 0,
  "errorRate": 0.02,
  "providerCount": 2
}
```

## Route registration

All three handlers are registered in `konstructOrkestra` before the health server starts:

```go
hs.Handle("/katalog", kordinator.BuildKatalogHandler(kat, kfg, resourceKatalog, crdHealthMap))
hs.Handle("/katalog/"+crd.Name, kordinator.BuildCRDInfoHandler(...))
hs.Handle("/katalog/"+crd.Name+"/health", kordinator.BuildCRDHealthHandler(...))
```

Routes must be registered before `hs.Start()`. After the server is listening, no new routes can be added.
