# 02 — Apply

## POST /api/v1/apply

The apply handler is the core of the Gateway API. It accepts two request shapes and applies the result to the cluster via server-side apply. Security enforcement happens through existing infrastructure — the Gateway API does not duplicate any validation logic.

## Two request shapes, one pipeline

```
1. Bearer token validation                          ← auth.go
2. Decode body → raw map[string]interface{}          ← apply.go
3. Detect shape: "target" key present → target mode,
   otherwise → full CR mode                          ← isTargetRequest
4. Target mode:  BuildCRFromTarget builds the CR      ← target.go
   Full CR mode: raw map used directly as the CR      ← apply.go
5. Resolve serve.name / serve.namespace (if declared)     ← target.go / apply.go
6. Server-side apply (SSA)                            ← dynamic client, field manager: orkestra-gateway
7. Build ApplyResponse (pollUrl, payload, violations) ← apply.go, payload.go
```

**Target mode** — the caller submits a flat field map, not a Kubernetes object:

```bash
curl -X POST /api/v1/apply \
  -d '{"target": "smartapp", "repository": "myorg/payments-api", "image": "ghcr.io/myorg/app:v1.0.0"}'
```

The gateway looks up the CRD by `target`, then builds the full CR from `serve.fields`/`serve labels/annotations` — the caller never constructs `apiVersion`/`kind`/`metadata`/`spec` themselves. See [Target Mode](../../../../documentation/concepts/idp/02-target-mode.md).

**Full CR mode** — the caller submits a complete Kubernetes object, exactly as `kubectl apply` would send one:

```bash
curl -X POST /api/v1/apply \
  -d '{"apiVersion": "platform.myorg.io/v1", "kind": "AppRequest", "metadata": {"name": "payments-api"}, "spec": {...}}'
```

Detection is by shape, not by a query parameter or header: presence of a `"target"` key means target mode, regardless of whether `apiVersion`/`kind` are also present — this lets a caller migrate incrementally by adding `"target"` without immediately stripping the rest.

Security — admission rules, namespace protection, deletion protection — is enforced identically for both shapes, by the existing infrastructure:

- **Webhooks enabled** (`security.webhooks.admission.enabled: true`): Kubernetes calls `/validate` before accepting the write. If the CR is rejected, the Kubernetes API server returns an error, which the Gateway API translates into `ApplyViolation`s. No logic is duplicated.
- **Webhooks not enabled**: The CR is written. The reconciler's namespace guard and admission pipeline run at reconcile time instead. Violations appear in the CR's status conditions rather than in the apply response.

`serve.tokens`, when declared on the CRD, is checked before either path runs — see [05-auth.md](05-auth.md#scoping-idpallowedtokens).

## Server-side apply (SSA)

Apply uses the dynamic client with field manager `orkestra-gateway`:

```go
client.Resource(gvr).
    Namespace(obj.GetNamespace()).
    Patch(ctx, obj.GetName(), types.ApplyPatchType, body, metav1.PatchOptions{
        FieldManager: "orkestra-gateway",
        Force:        overwrite, // false by default; true when ?overwrite=true
    })
```

By default `Force` is `false` — if another field manager owns a conflicting field, SSA returns an error rather than silently taking ownership. Pass `?overwrite=true` to set `Force: true` and claim ownership of conflicting fields.

SSA is idempotent — re-applying the same CR body with the same field values is a no-op. Callers do not need to check "does it exist" before applying, in either mode.

### `?dryRun=true`

Runs the same pipeline — build (target mode) or decode (full CR mode), resolve identity, SSA with Kubernetes' own server-side dry run — without persisting anything. Returns the same `ApplyResponse` shape a real apply would, with `"dryRun": true` set, so a form's "Preview" button gets the exact same violations/warnings a real submit would.

## Response

Every response, success or failure, is `ApplyResponse`:

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

`accepted` covers both create and update — the caller does not need to know which one happened. `warnings` carries admission-webhook advisories even on a successful apply, the same experience `kubectl apply` gives on the command line.

### `pollUrl` — where to check status, and it's configurable

By default `pollUrl` is derived from the applied resource: `/api/v1/resources/{kind}/{namespace}/{name}`. The platform team can override it per CRD with `serve.config.response.poll`:

```yaml
serve:
  config:
    response:
      poll:
        field: status.phase   # → pollUrl gets ?field=status.phase appended, for a single-value GET
        # url: '/api/v2/resources/{{ .kind }}/{{ .namespace }}/{{ .name }}'  # or replace it entirely
```

Callers should always use the `pollUrl` the response gave them, not assemble one themselves from `kind`/`namespace`/`name` — the platform team may have changed what it points to.

### `payload` — the curated view

When the CRD declares `serve.config.response.payload`, the response also carries a flat map of the platform team's chosen fields, evaluated as templates against the applied CR:

```json
{ "payload": { "phase": "", "serviceURL": "https://payments-api.staging.myorg.io", "nextSteps": "Waiting for resources..." } }
```

At apply time `.status` is not yet populated by the runtime — payload fields referencing it resolve to `""`, never an error. Poll `pollUrl` for the live value. → [`serve.config.response`](../../../../documentation/reference/schema/02-katalog/17-gateway-api.md#response-shape)

### Rejection — structured, not a raw Kubernetes error

```json
{
  "accepted": false,
  "message": "name is required",
  "violations": [
    { "field": "metadata.name", "message": "name is required", "severity": "error" }
  ]
}
```

`violations[].field` is a dot-notation path — enough for a form to highlight the offending input directly. When Kubernetes/the admission webhook rejects the SSA patch itself (a genuine field-manager conflict, an admission `deny`), the same `violations` shape carries that translated error too — the caller never has to parse a raw `Status` object.

### Conflict response (`?overwrite=true` not set)

```json
{
  "accepted": false,
  "message": "Apply failed with 1 conflict: conflict with \"kubectl-client-side-apply\" ...",
  "violations": [
    { "field": "spec.replicas", "message": "conflict with \"kubectl-client-side-apply\"" }
  ]
}
```

Pass `?overwrite=true` to take ownership of the conflicting fields and proceed.

## Polling

After apply, the CR is in Kubernetes but the operator may not have reconciled it yet. Poll the `pollUrl` the response gave you:

```bash
POLL_URL=$(curl -s -X POST .../api/v1/apply -d @app.json | jq -r '.pollUrl')

until curl -s "${GATEWAY}${POLL_URL}" \
  | jq -e '.status.conditions[]? | select(.type=="Ready" and .status=="True")' > /dev/null 2>&1; do
  sleep 2
done
```

→ Next: [03-schema.md](03-schema.md)
