# Dev Server — Endpoint Reference

All responses are `application/json` unless noted. All handlers return immediately.
Port: `9999` (fixed). Start with `ork run --dev-server`.

---

## GET /health

Simulates a healthy upstream service. Used by 01-health-gate (healthy CR) and the full-stack gated-app.

**Response — 200**
```json
{"status": "ok"}
```

---

## GET /status/:code

httpbin-compatible status endpoint. Used by 01-health-gate dev CRs.

| Path | Status | Body |
|---|---|---|
| `/status/200` | 200 | `{"status":"ok"}` |
| `/status/503` | 503 | `{"status":"unavailable"}` |

---

## GET /config/:name

Returns a static JSON config blob. Used by 02-config-inject — the operator embeds the response body into a ConfigMap.

**Response — 200**
```json
{
  "name": "<name>",
  "env": "production",
  "debug": "false",
  "replicas": 2
}
```

`:name` is taken from the URL path and echoed back for realism.

---

## POST /sign

Simulates an image signing service. Used by 03-image-signing. Ignores the request body and any `Authorization` header.

**Response — 200**
```json
{"signed": true}
```

---

## POST /auth/token

Returns a fake bearer token. Used by 04-chained as the first call — the operator uses the response body as the `token:` for the second call.

**Response — 200** (`Content-Type: text/plain`)
```
dev-token-abc123
```

---

## GET /resources/:name

Simulates a protected resource API. Used by 04-chained as the second call (authenticated via the token from `/auth/token`). Ignores the `Authorization` header.

**Response — 200**
```json
{
  "name": "<name>",
  "status": "active",
  "ready": true
}
```

---

## GET /flags/:name

Returns static feature flags. Used by the full-stack gated-app's `featureFlags` external call.

**Response — 200**
```json
{
  "name": "<name>",
  "v2Enabled": true,
  "betaEnabled": false
}
```
