# Endpoint Reference

All endpoints are served by the Orkestra runtime on port 8080. No authentication by default — the port is cluster-internal unless you expose it.

---

## Katalog endpoints

### `GET /katalog`

Returns the health summary for the entire Katalog.

```bash
curl -s localhost:8080/katalog | jq
```

**Response:**

```json
{
  "name": "my-katalog",
  "version": "0.1.0",
  "healthy": true,
  "OrkReady": true,
  "status": 200,
  "degradedReason": "",
  "total": 3,
  "totalEnabled": 3,
  "totalDegraded": 0,
  "statusCounts": {
    "healthy": 3,
    "degraded": 0,
    "pending": 0
  },
  "namespaces": {
    "default": ["website", "database"]
  },
  "gatewayEndpoint": "https://gateway.example.com",
  "crds": [...]
}
```

When any CRD is degraded, `healthy` is `false`, `status` is 503, and `degradedReason` names how many are degraded. `OrkReady` stays `true` as long as the runtime process itself is operational.

---

### `GET /katalog/raw`

Returns the Katalog exactly as you wrote it — no enrichment, no defaults applied.

---

### `GET /katalog/enriched`

Returns the Katalog with all Orkestra defaults applied: worker counts, resync intervals, queue config, enriched API types.

---

## Per-CRD endpoints

`{crd}` is the lowercase singular name of the CRD as declared in the Katalog (e.g. `blockchainapp`, `website`, `database`).

---

### `GET /katalog/{crd}`

Returns configuration and live stats for one operator.

```bash
curl -s localhost:8080/katalog/blockchainapp | jq
```

**Response shape:**

```json
{
  "name": "blockchainapp",
  "gvk": "resilience.demo.orkestra.io/v1alpha1, Kind=BlockchainApp",
  "workers": {
    "current": 3,
    "source": "default"
  },
  "resync": "15s",
  "queue": {
    "depth": 0,
    "failureThreshold": 3
  },
  "rbac": [...],
  "admission": {
    "webhooksEnabled": true,
    "validationTotal": 14,
    "validationDenied": 2,
    "valP95LatencyMs": 1
  },
  "deletionProtection": { "enabled": true, "total": 4, "blocked": 1 },
  "namespaceProtection": { "enabled": false },
  "conversion": { "enabled": false },
  "autoscaler": { ... },
  "metrics": {
    "workers": 3,
    "queueDepth": 0,
    "errorRate": 0.0,
    "successRate": 1.0
  }
}
```

**The `metrics` field** is always present regardless of whether `autoscale:` is declared. It is the hook for [ONCOP cross-binary observation](../oncop/) — a sibling runtime reads `cross.*.metrics.*` from this endpoint.

**`workers.source`** is one of:
- `default` — using the Orkestra built-in default
- `configured` — set in the Katalog
- `autoscaler` — currently overridden by the autoscaler

---

### `GET /katalog/{crd}/health`

Returns the live health state of one operator.

```bash
curl -s localhost:8080/katalog/blockchainapp/health | jq
```

**Response:**

```json
{
  "name": "website",
  "state": "healthy",
  "status": 200,
  "isKonductor": true,
  "healthy": true,
  "started": true,
  "pending": false,
  "startedAt": "2026-07-02T05:53:38Z",
  "uptime": "11m47s",
  "queueDepth": 0,
  "errorRate": 0,
  "consecutiveFails": 0,
  "totalReconciles": 10,
  "resourceCount": 1,
  "lastError": "",
  "lastReconcile": "2026-07-02T06:05:20Z",
  "hasUnhealthyDependencies": false
}
```

**`state`** values: `healthy`, `degraded`, `pending`.

**`isKonductor`** indicates whether this response came from the elected leader pod. In a multi-replica runtime deployment, only the leader (konductor) runs reconcilers — follower pods never start them, so their `state` is always `pending` and worker counts are zero. The Control Center retries this endpoint up to 3 times with keep-alives disabled (so the Service load-balancer routes each retry to a different pod) until it receives a response with `isKonductor: true`. If all retries hit followers, the last response is used rather than returning an error.

**`consecutiveFails`** resets to 0 on the first successful reconcile. **`lastError`** is preserved after recovery — it is an audit trail, not a live state indicator.

**`hasUnhealthyDependencies`** is `true` when at least one CRD listed in this operator's `dependsOn:` is currently degraded or missing. The operator itself may still be healthy — this flag surfaces upstream problems without requiring the consumer to walk the full dependency graph.

---

### `GET /katalog/{crd}/raw`

Returns the single CRD definition as you wrote it in the Katalog.

---

### `GET /katalog/{crd}/enriched`

Returns the CRD definition with all Orkestra defaults applied — workers, resync, queue config, enriched API types. Useful for debugging what Orkestra actually loaded.

---

## Per-CR endpoints

CR list and detail are served from the operator's **in-memory informer cache** — no API server call on read.

---

### `GET /katalog/{crd}/cr`

Returns all CR instances for this CRD.

```bash
curl -s localhost:8080/katalog/blockchainapp/cr | jq
```

**Response:**

```json
{
  "crd": "blockchainapp",
  "items": [
    {
      "name": "my-chain",
      "namespace": "default",
      "phase": "Running",
      "ready": true,
      "age": "4m32s",
      "generation": 2
    }
  ]
}
```

**`?field=<dot-path>`** additionally resolves that field against each instance's full object (available in the informer cache even though the default response doesn't include it) and returns it as `"value"` on every item:

```bash
curl -s "localhost:8080/katalog/blockchainapp/cr?field=spec.domain" | jq
```

```json
{
  "crd": "blockchainapp",
  "items": [
    { "name": "my-chain", "namespace": "default", ..., "value": "chain.example.com" }
  ]
}
```

This is what powers `operator: unique` at admission time — the gateway queries this endpoint instead of doing its own live List() against the API server. See [unique](../../reference/schema/02-katalog/07-validation.md#validationrules).

---

### `GET /katalog/{crd}/cr/{name}`

### `GET /katalog/{crd}/cr/{namespace}/{name}`

Returns the full status and children for one CR instance.

```bash
curl -s localhost:8080/katalog/blockchainapp/cr/my-chain | jq
curl -s localhost:8080/katalog/blockchainapp/cr/default/my-chain | jq
```

**Response:**

```json
{
  "name": "my-chain",
  "namespace": "default",
  "phase": "Running",
  "ready": true,
  "generation": 2,
  "status": { ... },
  "children": {
    "StatefulSet": [{ "name": "my-chain", "namespace": "default", "ready": true }],
    "Service": [{ "name": "my-chain-rpc", "namespace": "default" }]
  },
  "eventsEndpoint": "/katalog/blockchainapp/cr/default/my-chain/events"
}
```

Children are served from a 30-second in-memory cache to avoid concurrent LIST calls against the API server.

---

### `GET /katalog/{crd}/cr/{name}/events`

### `GET /katalog/{crd}/cr/{namespace}/{name}/events`

Returns the event stream for one CR instance — Orkestra reconcile events and raw Kubernetes events — sorted newest-first, capped at 100.

```bash
curl -s localhost:8080/katalog/blockchainapp/cr/my-chain/events | jq
```

**Response:**

```json
{
  "name": "my-chain",
  "namespace": "default",
  "events": [
    {
      "type": "Normal",
      "reason": "Reconciled",
      "message": "reconciled resilience.demo.orkestra.io/v1alpha1, Kind=BlockchainApp",
      "age": "2s"
    },
    {
      "type": "Warning",
      "reason": "ReconcileError",
      "message": "validation failed: images must come from the internal registry (myorg/)",
      "age": "45s"
    }
  ]
}
```

Event reasons emitted by Orkestra: `Reconciled`, `ReconcileError`, `ValidationWarning`, `Deleting`.

---

## Metrics query endpoint

### `GET /api/v1/query?query=<metric>`

Queries Orkestra's own Prometheus metrics by metric name. Served by the runtime on port 8080 alongside `/metrics`.

```bash
curl -s "localhost:8080/api/v1/query?query=go_goroutines" | jq
curl -s "localhost:8080/api/v1/query?query=process_cpu_seconds_total" | jq
```

**Response** follows the standard Prometheus instant query API format:

```json
{
  "status": "success",
  "data": {
    "resultType": "vector",
    "result": [
      {
        "metric": { "__name__": "go_goroutines" },
        "value": [1753228800, "42"]
      }
    ]
  }
}
```

Only bare metric names are accepted — PromQL functions (`rate(...)`, `sum(...)`, etc.) are not supported. This endpoint is the query target for `protocol: prometheus` external calls that point at `http://orkestra-runtime.orkestra-system.svc:8080`.

---

## Gateway endpoints

Served by the gateway process on port 8443 when the gateway is deployed. Not available from the runtime.

### `GET /katalog`

Returns security feature flags and per-CRD admission, conversion, deletion-protection, and namespace-protection stats. Does not include reconciler health (workers, queue, error rate) — those are runtime-only.

The response includes `"source": "gateway"` so the Control Center can distinguish it from the runtime response.

### `GET /katalog/{crd}`

Same stats for one CRD.

### `POST /notify`

Accepts a notification event body and dispatches it to the configured SMTP or Slack channels. The runtime POSTs here after its own throttle check — the gateway is the single dispatch point.

---

## Gateway API endpoints

Served by the gateway when `gateway.api.enabled: true` in the Katalog. All routes require `Authorization: Bearer <token>`.

### `POST /api/v1/apply`

Accepts any Kubernetes resource as JSON and applies it to the cluster via server-side apply (`fieldManager: orkestra-gateway`). Security is enforced by the existing webhook infrastructure — admission rules, namespace protection, and deletion protection all fire as normal before the write is accepted.

**Request:**

```json
{
  "apiVersion": "platform.myorg.io/v1",
  "kind": "AppRequest",
  "metadata": { "name": "payments-api", "namespace": "team-payments" },
  "spec": { "environment": "production", "image": "ghcr.io/myorg/payments:v2.1.0", "replicas": 3 }
}
```

**Response — applied:**

```json
{ "status": "applied", "resource": { "kind": "AppRequest", "name": "payments-api", "namespace": "team-payments", "uid": "abc-123" } }
```

**Response — rejected:**

```json
{ "status": "rejected", "reason": "AdmissionDenied", "violations": [{ "field": "spec.environment", "message": "must be one of: staging, production" }] }
```

---

### `GET /api/v1/resources/{kind}/{namespace}/{name}`

Returns the current CR state including Orkestra-managed status. Use this to poll after a `POST /api/v1/apply`.

### `GET /api/v1/resources/{kind}/{namespace}`

Returns all CR instances of this kind in the given namespace.

### `DELETE /api/v1/resources/{kind}/{namespace}/{name}`

Deletes a CR. Respects deletion protection — a protected CR returns a structured rejection rather than silently failing.

---

### `GET /api/v1/schema/{kind}`

Returns the OpenAPI spec schema for the CRD's `spec` properties. Only served for CRDs where `serve.enabled: true`. Used by the Control Center Serve form to render inputs from the schema without any additional configuration.

```json
{
  "properties": {
    "environment": { "type": "string", "enum": ["staging", "production"] },
    "image":       { "type": "string" },
    "replicas":    { "type": "integer", "minimum": 1, "maximum": 20, "default": 2 }
  },
  "required": ["environment", "image"]
}
```

---

## Control Center merge

The runtime `/katalog` response includes a `gatewayEndpoint` field pointing to the gateway's `/katalog`. The Control Center reads both:

1. Runtime `/katalog` — reconciler health, metrics, CR counts
2. Gateway `/katalog` — admission stats, protection stats

Merge key: `gvr` (`group/version/resource`). Neither process pushes to the other; each is independently queryable.

---

## Using from e2e

Port-forward to the `orkestra-runtime` service to assert live health in e2e tests:

```yaml
kubectl:
  port-forward:
    - service: orkestra-runtime
      namespace: orkestra-system
      port: 8080
      path: /katalog/blockchainapp/health
      jq: state
      equals: "healthy"
```
## Further Reading

- **[ONCOP](../oncop/)** — how cross-binary operators use these endpoints as observation targets
- **[Operator Autoscaler](../operator-autoscaler/)** — reading `metrics.*` cross-binary via `/katalog/{crd}`
- **[Health Subsystem](../health-subsystem/)** — operator-level probes (`/health`, `/ready`, `/startup`, `/metrics`)
