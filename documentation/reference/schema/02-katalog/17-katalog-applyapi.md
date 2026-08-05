# gateway.applyAPI

The `gateway.applyAPI` block enables the Orkestra Apply API — a CRUD REST surface for Custom Resources served by the gateway process.

```yaml
gateway:
  applyAPI:
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
| `enabled` | `false` | Enable the Apply API. When `true`, the gateway registers `POST /api/v1/apply`, `GET/DELETE /api/v1/resources/...`, `GET /api/v1/schema`, and `GET /api/v1/raw-schema` handlers. Also surfaces `idpEnabled: true` in the runtime `/katalog` response so the Control Center can render **[+ Create]** buttons. |
| `auth.include` | — | Path (relative to the Katalog file) to a YAML file containing a `tokens:` list. Inline `tokens:` entries override included entries with the same name. Expanded at load time. |

## `auth.tokens`

A list of bearer token entries. Every Apply API request must include `Authorization: Bearer <token>` matching one of these values.

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Human-readable identifier for logging and audit. Not sent in the request. Also used in `idp.allowedTokens` to reference this token. |
| `secretRef.name` | — | Kubernetes Secret name to read the token value from. |
| `secretRef.key` | — | Key within the Secret. |
| `secretRef.namespace` | — | Secret namespace. Defaults to Orkestra's own namespace. |
| `secretRef.rotateAfter` | — | Duration (e.g. `90d`, `720h`). Gateway checks the `generated-at` annotation on the Secret; when expired, deletes and recreates it with a new `uuidv4` token. Uses the same rotation machinery as `pkg/runtime/runners`. |
| `token` | — | `${ENV_VAR}` reference. Expanded at startup. For local development with `ork run` — export the variable before running. Literal values are not accepted. |

One of `secretRef` or `token` is required per entry. `secretRef` is the production pattern — the gateway reads the Secret at startup using its in-cluster ServiceAccount. If the Secret does not exist and `secretRef` is configured, the gateway creates it with a generated `uuidv4` token.

Token scope is not per-CRD. Per-CRD restriction is handled by `idp.allowedTokens` — see below.

## Endpoints

| Endpoint | Description |
|----------|-------------|
| `POST /api/v1/apply` | Create or update a CR via server-side apply. Supports two modes: **target mode** (`{"target": "...", ...}`) and **full CR mode** (`{"apiVersion": "...", "kind": "...", ...}`). |
| `GET /api/v1/resources/{kind}/{namespace}/{name}` | Read a CR's current spec and status. |
| `GET /api/v1/resources/{kind}/{namespace}` | List all CRs of a kind in a namespace. |
| `DELETE /api/v1/resources/{kind}/{namespace}/{name}` | Delete a CR (respects deletion protection). |
| `GET /api/v1/schema` | Service catalog — lists all IDP-enabled targets with pagination (`limit`, `offset`). |
| `GET /api/v1/schema?target=<t>` | Returns a flat field schema for a specific target. Callers don't need to know about `spec`, `labels`, or `annotations` — just fields. |
| `GET /api/v1/raw-schema?kind=<k>&apiVersion=<v>` | Returns the raw OpenAPI v3 schema for a CRD. For advanced callers who need direct access to the Kubernetes schema. Optional `apiVersion` for multi-version CRDs. |

`GET /api/v1/schema` and `GET /api/v1/schema?target=<t>` are only served for CRDs where `idp.enabled: true`.

## `idp.target` and target mode

When a CRD has `idp.target` set, callers can use **target mode**:

```bash
# Target mode — submit fields, not a CR
curl -X POST /api/v1/apply \
  -d '{"target": "smartapp", "repository": "...", "image": "...", "environment": "staging"}'
```

The gateway builds the full CR from the IDP field declarations. Callers don't need to know `apiVersion`, `kind`, `spec`, or `metadata` structure.

**Full CR mode** still works for backward compatibility:

```bash
curl -X POST /api/v1/apply \
  -d '{"apiVersion": "platform.myorg.io/v1", "kind": "AppRequest", ...}'
```

## `idp.allowedTokens` — fine-grained permissions

Per-CRD token scoping with operation-level permissions and namespace restrictions.

```yaml
idp:
  allowedTokens:
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
- Token names exist in `gateway.applyAPI.auth.tokens`
- Schema permissions only contain `get`/`list`
- Namespaces are within CRD's `allowedNamespaces` and outside `restrictedNamespaces`

## `include:` support

`include:` is supported in two places for cleaner Katalog composition:

### Gateway auth tokens

```yaml
gateway:
  applyAPI:
    auth:
      include: ./shared/tokens.yaml
      tokens:
        - name: control-center   # overrides if in included file
          secretRef:
            name: ork-apply-token
            key: token
```

### IDP allowedTokens

```yaml
idp:
  allowedTokens:
    include: ./shared/allowed-tokens.yaml
    control-center:   # overrides if in included file
      permissions:
        global: ["*"]
```

Both follow established merge semantics: included entries are loaded first, then inline entries override by name.

## Security

No new security configuration. Every property the Apply API enforces is already declared on the CRD entry:

| Property | Where it lives |
|----------|----------------|
| Admission rules | `security.webhooks.admission` + `validation` / `mutation` blocks |
| Namespace restriction | `allowedNamespaces` / `restrictedNamespaces` on the CRD entry |
| Token permissions | `idp.allowedTokens` on the CRD entry |
| Deletion protection | `security.deletionProtection` |

The Apply API calls the same validation functions the webhook path calls. There is no divergence between what `kubectl apply` would enforce and what `POST /api/v1/apply` enforces.

## Per-CRD: `idp`

Which CRDs appear with a **[+ Create]** button in the Control Center and have their schema served via `/api/v1/schema` is controlled per CRD entry.

→ [crd-entry.md — idp block](02-crd-entry.md#idp)

## See also

→ [concepts/idp](../../../concepts/idp/) — conceptual overview

→ [pkg/gateway](https://github.com/orkspace/orkestra/blob/main/pkg/gateway/README.md) — developer documentation

→ [contributing-controlcenter.md](../../../contributing/contributing-controlcenter.md) — IDP follow-on contributions
