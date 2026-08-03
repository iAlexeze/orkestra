# Target Mode

The Apply API accepts a simplified request format where callers submit a `target` and flat fields instead of a full Kubernetes CR. The gateway builds the CR from the IDP configuration.

---

## Why target mode exists

Every self-service caller — a browser form, a CI pipeline, a Slack bot — has the same problem: they want to describe what they need, not construct a Kubernetes object. A developer knows the repository and the image tag; they shouldn't need to know `apiVersion`, `kind`, `metadata`, or the difference between `spec` and `labels`.

Target mode hides Kubernetes behind the IDP contract. The platform team defines the fields. The gateway handles the rest.

---

## How it works

The Katalog declares `idp.target` and `idp.fields`:

```yaml
idp:
  enabled: true
  target: app
  name: '{{ repoSlug .repository }}'
  namespace: '{{ teamName }}-{{ environment }}'
  fields:
    repository:
      label: "Repository"
      type: string
      required: true
    image:
      label: "Container Image"
      type: string
      required: true
```

Callers submit `target` + fields:

```json
POST /api/v1/apply
{
  "target": "app",
  "repository": "myorg/payments-api",
  "image": "ghcr.io/myorg/app:v1.0.0",
  "team": "team-payments",
  "environment": "staging"
}
```

The gateway:

1. Looks up the CRD by `target`
2. Routes fields to `spec`, `metadata.labels`, or `metadata.annotations` based on `idp.fields` and `idp.additionalFields`
3. Resolves `idp.name` and `idp.namespace`
4. Applies the full CR via SSA

The caller never sees the CR.

---

## Two modes, one API

| Mode | Request format | When to use |
|------|----------------|-------------|
| **Target mode** | `{"target": "...", fields...}` | Self-service callers who don't know Kubernetes |
| **Full CR mode** | `{"apiVersion": "...", "kind": "...", ...}` | Advanced callers, existing clients, `kubectl` compatibility |

```bash
# Target mode — submit fields
curl -X POST /api/v1/apply \
  -d '{"target":"app","repository":"myorg/app","image":"..."}'

# Full CR mode — submit a complete CR
curl -X POST /api/v1/apply \
  -d '{"apiVersion":"platform.myorg.io/v1","kind":"App",...}'
```

Both modes produce the same result. The gateway detects which format you're using based on the presence of `target` or `apiVersion`+`kind`.

---

## The schema contract

Callers discover available targets and fields through the schema API:

```bash
# List all available targets
curl -X GET /api/v1/schema \
  -H "Authorization: Bearer $TOKEN"

# Get fields for a specific target
curl -X GET /api/v1/schema?target=app \
  -H "Authorization: Bearer $TOKEN"
```

The schema API returns a flat list of fields:

```json
{
  "target": "app",
  "title": "Application",
  "fields": {
    "repository": {
      "label": "Repository",
      "type": "string",
      "required": true
    },
    "image": {
      "label": "Container Image",
      "type": "string",
      "required": true
    }
  }
}
```

Callers don't need to know about `spec`, `labels`, or `annotations` — they just see fields.

---

## `idp.target` — the caller-facing identifier

`idp.target` decouples the caller-facing identifier from the Kubernetes `kind`.

```yaml
idp:
  enabled: true
  target: app   # callers use this, not "App" or "apprequests"
```

If omitted, defaults to the lowercased `kind` (e.g., `kind: App` → `target: app`).

`ork validate` ensures targets are unique across the Katalog.

---

## `idp.name` and `idp.namespace`

Target mode resolves `idp.name` and `idp.namespace` server-side, so callers don't need to know them:

```yaml
idp:
  enabled: true
  name: '{{ repoSlug .repository }}'          # → "payments-api"
  namespace: '{{ teamName }}-{{ environment }}' # → "team-payments-staging"
```

Callers never supply `metadata.name` or `metadata.namespace` in target mode.

When `idp.name` is not declared, the caller must supply a name. When `idp.namespace` is not declared on a namespaced CRD, the gateway rejects the request — self-service creation has no way to know where the CR belongs.

---

## Nested fields with `path`

Fields can map to nested locations in the CRD `spec` using `path`:

```yaml
idp:
  fields:
    repository:
      path: app.repository
      label: "Repository"
    cpu:
      path: app.resources.cpu
      label: "CPU Request"
```

Callers submit flat field names:

```json
{
  "target": "app",
  "repository": "myorg/app",
  "cpu": "500m"
}
```

The gateway maps to:

```yaml
spec:
  app:
    repository: myorg/app
    resources:
      cpu: 500m
```

→ [Nested fields with `path` reference](../../reference/schema/02-katalog/20-idp#idpfieldspath)

---

## Response: `pollUrl` and `payload`

A successful target-mode apply returns:

```json
{
  "accepted": true,
  "name": "payments-api",
  "namespace": "team-payments-staging",
  "kind": "AppRequest",
  "apiVersion": "platform.myorg.io/v1",
  "pollUrl": "/api/v1/resources/AppRequest/team-payments-staging/payments-api?field=status.phase",
  "payload": {
    "phase": "",
    "serviceURL": "https://payments-api.staging.myorg.io",
    "nextSteps": "Waiting for resources to be provisioned..."
  }
}
```

- **`pollUrl`** — where to GET the resource (configurable via `idp.config.response.poll`)
- **`payload`** — the platform team's curated view (`idp.config.response.payload`)

At apply time, `.status` is not yet available. Callers should poll `pollUrl` to see status updates.

→ [`idp.config.response` reference](../../reference/schema/02-katalog/20-idp.md#idpconfigresponse)

---

## Try it

```bash
ork init --pack use-cases/idp
```

Follow the README — it walks through target mode from schema discovery to apply to polling.

---

## See also

→ [`idp.target` schema reference](../../reference/schema/02-katalog/20-idp#idptarget)

→ [`idp.fields` schema reference](../../reference/schema/02-katalog/20-idp#idpfieldsname)

→ [`idp.namespace` reference](../../reference/schema/02-katalog/20-idp#idpnamespace)

→ [Apply API reference](../../reference/schema/02-katalog/17-katalog-applyapi.md)

→ [Additional Fields](01-additional-fields.md)