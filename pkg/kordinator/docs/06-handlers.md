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
    protStats   *health.DeletionProtectionStats,
    nsStats     *health.NamespaceProtectionStats,
    isProtected bool,
    provStats   *health.ProviderStats,
) http.HandlerFunc
```

Serves `GET /katalog/{crd}`. Returns the full CRD detail. The response is assembled on every request from live atomic reads — no caching.

**Autoscaler workers section.** When `autoscale:` is declared on the CRD, the response includes an `autoscalerWorkers` object. This is populated by calling the `workerInfoFn` closure set on `CRDHealth` during startup — it reads the live semaphore state and autoscaler snapshot on every request. When no autoscaler is configured the field is omitted entirely.

```json
"autoscalerWorkers": {
  "configured": 4,
  "effective": 8,
  "inFlight": 3,
  "idle": 5,
  "max": 8,
  "autoscalerEnabled": true,
  "overrideActive": true,
  "overrideWorkers": 8,
  "queueDepth": 12,
  "queueDepthConfigured": 100,
  "queueDepthEffective": 200,
  "busyPercent": 37.5
}
```

**Rollback section.** When `crd.HasRollbackRules()` is true (the CRD declares `operatorBox.rollback:`), the response includes a `rollback` object populated from `CRDHealth.RollbackStats()`:

```json
"rollback": {
  "totalRollbacks": 3,
  "active": false,
  "lastRollbackAt": "2026-04-18T14:22:01Z"
}
```

When rollback is currently active (`active: true`), the Control Center renders an alert banner on the CRD detail page.

**Metrics section.** When `autoscale:` is declared, the response also includes a `"metrics"` object with the live `AutoMetrics` values. This is the endpoint that cross-binary autoscale conditions hit via `source.endpoint` — the remote autoscaler reads `metrics.*` fields from this response using the same fallback path as `readCross` uses for CR spec/status observation.

```json
"metrics": {
  "queueDepth": 342,
  "workersBusyPercent": 73.5,
  "workersIdlePercent": 26.5,
  "reconcileDurationP95Ms": 47.0,
  "errorRatePercent": 0.2
}
```

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

**Deletion protection section.** Appears when `security.deletionProtection` is enabled. `protStats` carries admission counters (total, blocked, allowed) for DELETE reviews only. When `protStats` is nil a zero response is returned with `enabled` set to `isProtected` — avoids a nil check in every caller.

**Namespace protection section.** Appears when `security.namespaceProtection` is declared (i.e., `crd.HasNamespaceRules()` is true), regardless of whether the webhook is active. Populated from `nsStats` (CREATE/UPDATE reviews). Includes `allowedNamespaces` and `restrictedNamespaces` from the Katalog schema — the declared rules, not live enforcement state. Distinct from deletion protection: different admission operations, different future evolution path.

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

## BuildCRListHandler

```go
func BuildCRListHandler(crd orktypes.CRDEntry, inf cache.SharedIndexInformer) http.HandlerFunc
```

Serves `GET /katalog/{crd}/cr`. Returns all CR instances for a CRD as a sorted list — no API server calls, reads from the informer cache. Each row contains: `name`, `namespace`, `phase`, `ready`, `readyReason`, `age`, `generation`.

```json
{
  "crd": "website",
  "gvk": "apps.example.io/v1, Kind=Website",
  "total": 3,
  "items": [
    { "name": "acme-site", "namespace": "prod", "ready": true, "age": "2d", "generation": 5 }
  ]
}
```

## BuildCRDetailHandler

```go
func BuildCRDetailHandler(crd orktypes.CRDEntry, inf cache.SharedIndexInformer, kube *kubeclient.Kubeclient, rc orktypes.OperatorBoxConfig) http.HandlerFunc
```

Serves `GET /katalog/{crd}/cr/{name}` (cluster-scoped) and `GET /katalog/{crd}/cr/{namespace}/{name}` (namespaced). Reads the CR from the informer cache and fetches child resources from the API server on demand.

The response includes:
- Full `status` subresource
- `ready` / `readyReason` / `readyMessage` from the Ready condition (or annotation-based for statusless CRDs)
- `children` — one entry per child resource kind, value is a single `ChildSummary` object (one child) or `[]ChildSummary` (multiple)
- `eventsEndpoint` — the URL to fetch Kubernetes events for this CR

## BuildCRDetailAndEventsHandler

Routes all sub-paths under `/katalog/{crd}/cr/`. Dispatches to the detail handler or the events handler based on whether the path ends with `/events`. Register at `/katalog/{crd}/cr/` (trailing slash required).

## rawToMap pattern

`cr_handlers.go` and `cr_children.go` read CRs from informer caches as `interface{}`. Rather than asserting `*unstructured.Unstructured` directly, both use `rawToMap`:

```go
objMap, err := rawToMap(raw)
if err != nil {
    continue // or error response
}
// navigate via objMap["metadata"], objMap["spec"], objMap["status"]
```

`rawToMap` has a fast path for `*unstructured.Unstructured` (returns `u.Object` directly) and a JSON round-trip fallback for any other type. This means the CR handlers work correctly whether the informer stores typed or unstructured objects — the same correctness guarantee as `objectToMap` in the template engine.

Helper `metaField(objMap, field)` extracts string fields from `objMap["metadata"]` with a safe nil check.

## Route registration

All handlers are registered in `konstructOrkestra` before the health server starts:

```go
hs.Handle("/katalog", kordinator.BuildKatalogHandler(kat, kfg, resourceKatalog, crdHealthMap))
hs.Handle("/katalog/"+crd.Name, kordinator.BuildCRDInfoHandler(...))
hs.Handle("/katalog/"+crd.Name+"/health", kordinator.BuildCRDHealthHandler(...))
hs.Handle("/katalog/"+crd.Name+"/cr", kordinator.BuildCRListHandler(crd, inf))
hs.Handle("/katalog/"+crd.Name+"/cr/", kordinator.BuildCRDetailAndEventsHandler(crd, inf, kube, rc))
```

Routes must be registered before `hs.Start()`. After the server is listening, no new routes can be added.
