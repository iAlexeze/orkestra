# 01 — Overview

## The fundamental insight

The Orkestra runtime watches for Custom Resources and reconciles when they appear. It has no "how did this CR get here" field, and it does not need one:

```
kubectl apply          ↓
ork run --apply-cr     ↓
GitHub webhook         ↓  →  CR in Kubernetes  →  Runtime reconciles
CI curl POST           ↓
Browser form submit    ↓
Any HTTP caller        ↓
```

The Gateway API is a new delivery path. From the runtime's perspective, a CR arrived. It reconciles. The delivery mechanism is invisible.

## Why the gateway is the right place

The gateway already:

- Reads the full Katalog config — it knows every CRD, its admission rules, its namespace protection config, and its deletion protection config
- Runs the same admission validation functions the webhook runs
- Resolves CRD OpenAPI schemas from the REST mapper — used for webhook conversion

Adding the Gateway API means calling the same functions in a different code path: an HTTP handler instead of a webhook handler. Nothing is duplicated.

The runtime has no role here. Adding the Gateway API to the runtime would mean either duplicating the security logic or introducing a dependency on the gateway. Neither is acceptable.

## What the gateway now owns

```
HealthServer   :8080   →  /katalog, /apply health
WebhookServer  :8443   →  /validate, /mutate, /convert, /deletion-protection
ApplyServer    :8080   →  /api/v1/apply, /api/v1/resources/..., /api/v1/schema, /api/v1/raw-schema
```

The Gateway API runs on the health server (HTTP :8080), not the webhook server (HTTPS :8443). Webhook callers are the Kubernetes API server — they require mTLS. Gateway API callers are humans and pipelines — they use bearer tokens over HTTPS if TLS is terminated at the ingress, or HTTP for in-cluster calls.

## Two fields on the runtime

The runtime's `/katalog` response carries `gatewayEndpoint`, and per CRD, `serveEnabled` and `target`:

```json
{
  "gatewayEndpoint": "https://gateway.myorg.internal",
  "crds": [
    { "name": "application", "serveEnabled": true, "target": "smartapp" }
  ]
}
```

`serveEnabled: true` when the CRD has `serve.enabled: true`. `target` is `serve.target` (or the lowercased Kind when unset) — the identifier the Control Center submits to the gateway; it never derives one from Kind or GVK itself. The Control Center reads both and renders the **[+ Create]** button. No new runtime endpoints beyond these two fields.

→ Next: [02-apply.md](02-apply.md)
