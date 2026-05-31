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

Returns all flags for an app as JSON. Used by the full-stack gated-app's `featureFlags` external call.

**Response — 200**
```json
{
  "name": "<name>",
  "v2Enabled": true,
  "betaEnabled": false
}
```

---

## GET /flags/:name/:flag

Returns the current value of a single flag as plain text (`true` or `false`). Used by 05-feature-flags.

**Response — 200** (`Content-Type: text/plain`)
```
true
```

---

## POST /flags/:name/:flag/toggle

Flips the named flag and returns the new value as plain text. Persists for the lifetime of the process.

**Response — 200** (`Content-Type: text/plain`)
```
false
```

---

## GET /sbom/:image

Returns a vulnerability report for the given image. Used by 06-sbom-cosign as the first call. The operator gates deployment via a `when:` condition on `external.sbom.body` checking `"clean":true`.

| Image | Status | Behaviour |
|---|---|---|
| `nginx:vulnerable` | 200 | `critical: 3`, `high: 12`, `clean: false` — Deployment gated |
| `nginx:unknown` | 404 | No SBOM found — `continueOnError: false` halts the reconcile |
| anything else | 200 | `critical: 0`, `high: 0`, `clean: true` — passes the gate |

**Response — 200**
```json
{
  "image": "<image>",
  "scanner": "dev-scanner",
  "critical": 0,
  "high": 0,
  "medium": 2,
  "low": 7,
  "clean": true
}
```

---

## POST /cosign/verify

Verifies a container image signature. Used by 06-sbom-cosign as the second call, chained after `/sbom`. The operator passes `{"image": "{{ .spec.image }}"}` in the body.

| Image | Status | Behaviour |
|---|---|---|
| `nginx:unsigned` | 403 | No valid signature — Deployment gated |
| anything else | 200 | Verified |

**Response — 200**
```json
{
  "verified": true,
  "image": "<image>",
  "signer": "dev-signer@example.com",
  "algorithm": "ecdsa-p256"
}
```

**Response — 403**
```json
{
  "error": "no valid signature found",
  "image": "nginx:unsigned",
  "reason": "image has no cosign signature — sign it before deploying"
}
```

---

## GET /vault/v1/secret/data/:path

Mimics the Vault KV v2 API. Used by 07-vault-secret-gate. The operator gates the Deployment on the secret existing and not being expired.

| Path contains | Status | Behaviour |
|---|---|---|
| `expired` | 403 | Lease expired — secret must be rotated |
| `missing` | 404 | Secret not found at path |
| anything else | 200 | Returns secret data |

**Response — 200**
```json
{
  "data": {
    "data": {
      "value": "dev-secret-value",
      "expires_at": "2099-12-31T00:00:00Z"
    },
    "metadata": {
      "version": 1,
      "created_time": "2026-01-01T00:00:00Z",
      "deletion_time": "",
      "destroyed": false
    }
  }
}
```

---

## POST /v1/data/:policy

OPA REST API wire format. Used by 08-opa-policy. The operator sends `{"input": {"name": "...", "namespace": "..."}}` and gates resources on `result.allow == true`.

| Input | Decision | Reason |
|---|---|---|
| `namespace: "forbidden"` | deny | Namespace not permitted by org policy |
| `name` contains `"deny"` | deny | Name contains a denied prefix |
| anything else | allow | — |

**Response — 200**
```json
{
  "result": {
    "allow": true,
    "deny": false,
    "reason": ""
  }
}
```

---

## GET /certs/:name/status

Returns the current certificate status for a named resource. Used by 09-cert-readiness. Default state is `issued`. Toggle via `POST /certs/:name/toggle` to simulate a cert not yet ready.

| State | Status code | `status` field |
|---|---|---|
| issued (default) | 200 | `"issued"` |
| pending (after toggle) | 202 | `"pending"` |

**Response — 200**
```json
{
  "name": "<name>",
  "status": "issued",
  "issuer": "dev-ca",
  "expires_at": "2099-12-31T00:00:00Z"
}
```

---

## POST /certs/:name/toggle

Flips the cert between `issued` and `pending`. Returns the new status as plain text. Persists for the lifetime of the process.

**Response — 200** (`Content-Type: text/plain`)
```
pending
```
