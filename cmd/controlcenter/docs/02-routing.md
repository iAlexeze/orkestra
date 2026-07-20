# 02 — Routing

All requests arrive at `ControlCenter.ServeHTTP`, which strips the `/controlcenter` prefix and dispatches by path. There is no external router dependency — dispatching is a plain `switch` on string prefix matching.

## Full route table

| Path (after `/controlcenter`) | Handler | Template |
|-------------------------------|---------|----------|
| `/` | `handleIndex` | `index.html` |
| `/sse` | `ServeSSE` | — (streaming) |
| `/api/snapshot` | `handleAPISnapshot` | — (JSON) |
| `/assets/**` | `serveAsset` | — (embedded FS) |
| `/metrics` | `handleMetricsPage` | `metrics.html` |
| `/debug/file` | `handleDebugFile` | — (plain text) |
| `/docs` | `handleDocsLanding` | `docs.html` |
| `/docs/{katalog}/{crd}` | `handleCRDDocs` | `crd_docs.html` |
| `/katalog/{name}` | `handleKatalogPanel` | `katalog.html` |
| `/katalog/{name}/raw` | `handleProxyKatalogSpec` | — (JSON proxy) |
| `/katalog/{name}/enriched` | `handleProxyKatalogSpec` | — (JSON proxy) |
| `/katalog/{name}/crd/{crd}` | `handleCRDDetail` | `crd.html` |
| `/katalog/{name}/crd/{crd}/raw` | `handleProxyCRDSpec` | — (JSON proxy) |
| `/katalog/{name}/crd/{crd}/enriched` | `handleProxyCRDSpec` | — (JSON proxy) |
| `/katalog/{name}/crd/{crd}/cr` | `handleCRList` | `cr_list.html` |
| `/katalog/{name}/crd/{crd}/cr/{crname}` | `handleCRDetail` (cluster-scoped) | `cr_detail.html` |
| `/katalog/{name}/crd/{crd}/cr/{ns}/{crname}` | `handleCRDetail` (namespaced) | `cr_detail.html` |
| `GET /api/instances` | `handleListInstances` | — (JSON) |
| `POST /api/instances` | `handleAddInstance` | — (JSON) |
| `PUT /api/instances/{url}` | `handleUpdateInstance` | — (JSON) |
| `DELETE /api/instances/{url}` | `handleDeleteInstance` | — (JSON) |
| `GET /api/idp/schema/{kind}` | `handleIDPSchema` | — (JSON proxy) |
| `POST /api/idp/apply` | `handleIDPApply` | — (JSON proxy) |

## Katalog sub-routing

`/katalog/**` paths are split into parts and dispatched by `routeKatalog`. The priority order matters — more specific patterns appear first:

```
parts[4] == "cr"          →  routeCR  (CRD instance list/detail)
parts[4] == "raw"|enriched →  proxy CRD spec JSON
parts[2] == "crd"         →  CRD detail page
parts[2] == "raw"|enriched →  proxy katalog spec JSON
default                    →  katalog panel
```

`routeCR` then dispatches by the number of remaining path segments:

```
len(crParts) == 1  →  CR list
len(crParts) == 2  →  cluster-scoped CR detail  (/cr/{name})
len(crParts) == 3  →  namespaced CR detail       (/cr/{ns}/{name})
```

## Proxy endpoints

`/raw` and `/enriched` sub-paths on both katalog and CRD routes proxy the request directly to the Orkestra runtime and pipe the JSON response body back to the browser. They are used by the YAML viewer modal in the UI. The proxying is done by `proxyJSON`, which issues a plain `http.Get` to the runtime URL and copies the body.

## IDP proxy endpoints

`/api/idp/schema/{kind}` and `/api/idp/apply` are server-side proxies to the companion gateway's Apply API. The CC holds the `GATEWAY_TOKEN` bearer token; the browser never sees it. All IDP form traffic flows through these two routes — schema fetch and CR apply — so the gateway can be behind a different origin without CORS or token-exposure issues.

`handleIDPSchema` forwards `GET {gatewayEndpoint}/api/v1/schema/{kind}`. `handleIDPApply` forwards `POST {gatewayEndpoint}/api/v1/apply` with the request body unchanged. Both respond with the gateway's status code and JSON body verbatim.

## Template rendering

All page handlers call `cc.renderTemplate(w, templateName, data)`. Templates are loaded from the embedded `assets/templates/*.html` filesystem. The helper is in `cc/helper.go`.

→ Next: [03-data-flow.md](03-data-flow.md)
