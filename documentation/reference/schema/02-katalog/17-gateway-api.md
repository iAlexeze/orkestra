# gateway.api

The `gateway.api` block enables the Orkestra Gateway API — a CRUD REST surface for Custom Resources served by the gateway process.

```yaml
gateway:
  api:
    enabled: false              # opt-in
    auth:
      include: ./tokens.yaml    # load tokens from external file
      tokens:
        - name: ci-pipeline
          secretRef:
            name: ork-apply-token
            key: token
            rotateAfter: 90d   # optional — gateway recreates the Secret when expired
            # namespace: orkestra-system  ← defaults to Orkestra's namespace
        - name: local-dev
          token: "${ORK_DEV_TOKEN}"       # env var — export before ork run
```

## Top-level fields

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `false` | Enable the Gateway API. When `true`, the gateway registers `POST /api/v1/apply`, `GET/DELETE /api/v1/resources/...`, `GET /api/v1/schema`, and `GET /api/v1/raw-schema` handlers. Also surfaces `serveEnabled: true` in the runtime `/katalog` response so the Control Center can render **[+ Create]** buttons. |
| `auth.include` | — | Path (relative to the Katalog file) to a YAML file containing a `tokens:` list. Inline `tokens:` entries override included entries with the same name. Expanded at load time. |

## `auth.tokens`

A list of bearer token entries. Every Gateway API request must include `Authorization: Bearer <token>` matching one of these values.

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Human-readable identifier for logging and audit. Not sent in the request. Also used in `serve.tokens` to reference this token. |
| `secretRef.name` | — | Kubernetes Secret name to read the token value from. |
| `secretRef.key` | — | Key within the Secret. |
| `secretRef.namespace` | — | Secret namespace. Defaults to Orkestra's own namespace. |
| `secretRef.rotateAfter` | — | Duration (e.g. `90d`, `720h`). Gateway checks the `generated-at` annotation on the Secret; when expired, deletes and recreates it with a new `uuidv4` token. Uses the same rotation machinery as `pkg/runtime/runners`. |
| `token` | — | `${ENV_VAR}` reference. Expanded at startup. For local development with `ork run` — export the variable before running. Literal values are not accepted. |

One of `secretRef` or `token` is required per entry. `secretRef` is the production pattern — the gateway reads the Secret at startup using its in-cluster ServiceAccount. If the Secret does not exist and `secretRef` is configured, the gateway creates it with a generated `uuidv4` token.

Token scope is not per-CRD. Per-CRD restriction is handled by `serve.tokens` — see below.

## Endpoints

| Endpoint | Description |
|----------|-------------|
| `POST /api/v1/apply` | Create or update a CR via server-side apply. Supports two modes: **target mode** (`{"target": "...", ...}`) and **full CR mode** (`{"apiVersion": "...", "kind": "...", ...}`). |
| `GET /api/v1/resources/{kind}/{namespace}/{name}` | Read a CR's current spec and status. |
| `GET /api/v1/resources/{kind}/{namespace}` | List all CRs of a kind in a namespace. |
| `DELETE /api/v1/resources/{kind}/{namespace}/{name}` | Delete a CR (respects deletion protection). |
| `GET /api/v1/schema` | Service catalog — lists all serve-enabled targets with pagination (`limit`, `offset`). |
| `GET /api/v1/schema?target=<t>` | Returns a flat field schema for a specific target. Callers don't need to know about `spec`, `labels`, or `annotations` — just fields. |
| `GET /api/v1/raw-schema?kind=<k>&apiVersion=<v>` | Returns the raw OpenAPI v3 schema for a CRD. For advanced callers who need direct access to the Kubernetes schema. Optional `apiVersion` for multi-version CRDs. |

`GET /api/v1/schema` and `GET /api/v1/schema?target=<t>` are only served for CRDs where `serve.enabled: true`.

## `serve.target` and target mode

When a CRD has `serve.target` set, callers can use **target mode**:

```bash
# Target mode — submit fields, not a CR
curl -X POST /api/v1/apply \
  -d '{"target": "smartapp", "repository": "...", "image": "...", "environment": "staging"}'
```

The gateway builds the full CR from the serve field declarations. Callers don't need to know `apiVersion`, `kind`, `spec`, or `metadata` structure.

**Full CR mode** still works for backward compatibility:

```bash
curl -X POST /api/v1/apply \
  -d '{"apiVersion": "platform.myorg.io/v1", "kind": "AppRequest", ...}'
```

### Nested fields with `path`

A submitted field name doesn't have to match its location in `spec`. Set `path` on the field:

```yaml
serve:
  fields:
    cpu:
      label: "CPU Request"
      path: app.resources.cpu   # spec.app.resources.cpu
```

Callers still submit the flat name (`{"target": "app", "cpu": "500m"}`) — the gateway writes it to the nested location. Omit `path` and the field name is used flat (`spec.cpu`).

## Response shape

A successful apply returns `ApplyResponse` (`200`/`201`):

```json
{
  "accepted": true,
  "name": "payments-api",
  "namespace": "team-payments-staging",
  "kind": "AppRequest",
  "apiVersion": "platform.myorg.io/v1",
  "pollUrl": "/api/v1/resources/AppRequest/team-payments-staging/payments-api",
  "warnings": [],
  "payload": { "phase": "", "serviceURL": "https://payments-api.staging.myorg.io" }
}
```

A rejected apply returns `422` with structured, field-level violations instead of a raw Kubernetes error string:

```json
{
  "accepted": false,
  "message": "name is required",
  "violations": [
    { "field": "metadata.name", "message": "name is required", "severity": "error" }
  ]
}
```

`violations[].field` is a dot-notation path (`spec.environment`, `metadata.name`) — enough for a form to highlight the offending input directly, rather than parsing a message string. `warnings` carries admission-webhook advisories even on a successful apply — the same experience `kubectl apply` gives on the command line.

### `serve.config.response` — shaping what callers see back

Controls `pollUrl` and `payload` above:

```yaml
serve:
  config:
    response:
      default: true              # include the full CR alongside payload (default)
      payload:
        phase:      '{{ .status.phase }}'
        serviceURL: 'https://{{ .metadata.name }}.{{ .spec.environment }}.myorg.io'
      exclude:
        - metadata.managedFields
      poll:
        field: status.phase      # → pollUrl gets ?field=status.phase appended
```

- **`payload`** — named template expressions, evaluated against the CR (`.spec`, `.metadata` at apply time; `.status` too once the runtime has written it, at `GET /api/v1/resources/...`). Unresolvable expressions become `""`, never an error.
- **`default: false`** — `payload` becomes the entire response instead of riding alongside the full CR. Use this to keep a curated, stable response shape independent of what the CRD's spec looks like.
- **`exclude`** — dot-notation paths stripped from the full-CR portion of `GET`/list responses (e.g. hiding `metadata.managedFields`). Applied before `payload`, and never removes a `payload` key.
- **`poll`** — overrides the derived `pollUrl`. `field` appends `?field=<path>` for lightweight single-value polling; `url` replaces the whole URL with a template.

→ [`serve.config.response` field reference](20-serve.md#serveconfigresponse)

## `serve.tokens` — fine-grained permissions

Per-CRD token scoping with operation-level permissions and namespace restrictions.

```yaml
serve:
  tokens:
    control-center:
      namespaces: ["default"]
      permissions:
        global: ["*"]           # full access
    ci-pipeline:
      namespaces: ["staging"]
      permissions:
        resources: ["create", "update", "get", "list"]
        schema: ["get", "list"]
```

| Scope | Endpoints | Valid operations |
|-------|-----------|------------------|
| `global` | All endpoints | `get`, `list`, `create`, `update`, `delete`, `*` |
| `schema` | `GET /api/v1/schema`, `GET /api/v1/raw-schema` | `get`, `list` (only) |
| `resources` | `POST /api/v1/apply`, `GET/DELETE /api/v1/resources` | `get`, `list`, `create`, `update`, `delete`, `*` |

`ork validate` ensures:
- Token names exist in `gateway.api.auth.tokens`
- Schema permissions only contain `get`/`list`
- Namespaces are within CRD's `allowedNamespaces` and outside `restrictedNamespaces`

## `include:` support

`include:` is supported in two places for cleaner Katalog composition:

### Gateway auth tokens

```yaml
gateway:
  api:
    auth:
      include: ./shared/tokens.yaml
      tokens:
        - name: control-center   # overrides if in included file
          secretRef:
            name: ork-apply-token
            key: token
```

### serve.tokens

```yaml
serve:
  tokens:
    include: ./shared/allowed-tokens.yaml
    control-center:   # overrides if in included file
      permissions:
        global: ["*"]
```

Both follow established merge semantics: included entries are loaded first, then inline entries override by name.

## Security

Nothing here needs a new configuration surface to learn — every property the Gateway API enforces is declared right on the CRD entry, same as it would be for any other caller:

| Property | Where it lives |
|----------|----------------|
| Admission rules | `security.webhooks.admission` + `validation` / `mutation` blocks |
| Namespace restriction (topology — same for every caller) | `allowedNamespaces` / `restrictedNamespaces` on the CRD entry |
| Token permissions (identity — per caller) | `serve.tokens` on the CRD entry — a real, separate authorization layer; see [Serve token permissions](../../../security/08-serve-permissions.md) |
| Deletion protection | `security.deletionProtection` |

The Gateway API calls the same validation functions the webhook path calls. There is no divergence between what `kubectl apply` would enforce and what `POST /api/v1/apply` enforces — `serve.tokens` is the one addition specific to the Gateway API, since `kubectl` has no notion of a bearer token to scope.

## Per-CRD: `serve`

Which CRDs appear with a **[+ Create]** button in the Control Center and have their schema served via `/api/v1/schema` is controlled per CRD entry.

→ [crd-entry.md — serve block](02-crd-entry.md#serve)

## Where to go next

→ [concepts/self-service](../../../concepts/self-service/) — conceptual overview

→ [security/serve-permissions](../../../security/08-serve-permissions.md) — `serve.tokens` as a security layer, not just a config block

→ [pkg/gateway](https://github.com/orkspace/orkestra/blob/main/pkg/gateway/README.md) — developer documentation

→ [contributing-controlcenter.md](../../../contributing/contributing-controlcenter.md) — Serve follow-on contributions
