# gateway.applyAPI

The `gateway.applyAPI` block enables the Orkestra Apply API — a CRUD REST surface for Custom Resources served by the gateway process.

```yaml
gateway:
  applyAPI:
    enabled: false              # opt-in
    auth:
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
| `enabled` | `false` | Enable the Apply API. When `true`, the gateway registers `POST /api/v1/apply`, `GET/DELETE /api/v1/resources/...`, and `GET /api/v1/schema/...` handlers. Also surfaces `idpEnabled: true` in the runtime `/katalog` response so the Control Center can render **[+ Create]** buttons. |

## `auth.tokens`

A list of bearer token entries. Every Apply API request must include `Authorization: Bearer <token>` matching one of these values.

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Human-readable identifier for logging and audit. Not sent in the request. |
| `secretRef.name` | — | Kubernetes Secret name to read the token value from. |
| `secretRef.key` | — | Key within the Secret. |
| `secretRef.namespace` | — | Secret namespace. Defaults to Orkestra's own namespace. |
| `secretRef.rotateAfter` | — | Duration (e.g. `90d`, `720h`). Gateway checks the `generated-at` annotation on the Secret; when expired, deletes and recreates it with a new `uuidv4` token. Uses the same rotation machinery as `pkg/runtime/runners`. |
| `token` | — | `${ENV_VAR}` reference. Expanded at startup. For local development with `ork run` — export the variable before running. Literal values are not accepted. |

One of `secretRef` or `token` is required per entry. `secretRef` is the production pattern — the gateway reads the Secret at startup using its in-cluster ServiceAccount. If the Secret does not exist and `secretRef` is configured, the gateway creates it with a generated `uuidv4` token.

Token scope is not per-CRD. Namespace restriction is handled by `allowedNamespaces` / `restrictedNamespaces` on the CRD entry — the gateway enforces those on every Apply API request regardless of which token was used.

## Endpoints

| Endpoint | Description |
|----------|-------------|
| `POST /api/v1/apply` | Create or update a CR via server-side apply |
| `GET /api/v1/resources/{kind}/{namespace}/{name}` | Read a CR's current spec and status |
| `GET /api/v1/resources/{kind}/{namespace}` | List all CRs of a kind in a namespace |
| `DELETE /api/v1/resources/{kind}/{namespace}/{name}` | Delete a CR (respects deletion protection) |
| `GET /api/v1/schema/{kind}` | Return the OpenAPI spec schema for a CRD |

`GET /api/v1/schema/{kind}` is only served for CRDs where `idp.enabled: true`.

## Security

No new security configuration. Every property the Apply API enforces is already declared on the CRD entry:

| Property | Where it lives |
|----------|---------------|
| Admission rules | `security.webhooks.admission` + `validation` / `mutation` blocks |
| Namespace restriction | `allowedNamespaces` / `restrictedNamespaces` on the CRD entry |
| Deletion protection | `security.deletionProtection` |

The Apply API calls the same validation functions the webhook path calls. There is no divergence between what `kubectl apply` would enforce and what `POST /api/v1/apply` enforces.

## Per-CRD: `idp`

Which CRDs appear with a **[+ Create]** button in the Control Center and have their schema served via `/api/v1/schema/{kind}` is controlled per CRD entry.

→ [crd-entry.md — idp block](02-crd-entry.md#idp)

## See also

→ [concepts/idp](../../../concepts/idp/) — conceptual overview
→ [pkg/gateway](https://github.com/orkspace/orkestra/blob/main/pkg/gateway/README.md) — developer documentation
→ [contributing-controlcenter.md](../../../contributing/contributing-controlcenter.md) — IDP follow-on contributions
