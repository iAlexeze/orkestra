# 02 — Apply

## POST /api/v1/apply

The apply handler is the core of the Apply API. It accepts any Kubernetes resource body and applies it to the cluster via server-side apply. Security enforcement happens through existing infrastructure — the Apply API does not duplicate any validation logic.

## Pipeline

```
1. Bearer token validation     ← auth.go
2. Decode body → unstructured  ← apply/handler.go
3. Server-side apply (SSA)     ← dynamic client, field manager: orkestra-gateway
4. Translate response          ← apply/handler.go
```

The Apply API is a delivery path, not a validation layer. Security — admission rules, namespace protection, deletion protection — is enforced by the existing infrastructure:

- **Webhooks enabled** (`security.webhooks.admission.enabled: true`): Kubernetes calls `/validate` before accepting the write. If the CR is rejected, the Kubernetes API server returns an error. The Apply API translates that error into the structured response format and returns it to the caller. No logic is duplicated.
- **Webhooks not enabled**: The CR is written. The reconciler's namespace guard (`run_namespace_guard.go`) and admission pipeline (`run_admission.go`, `run_validations.go`) run at reconcile time. Violations appear in the CR's status conditions.

The Apply API does not duplicate any validation logic. It is a thin layer: auth → SSA → translate response.

## Step 3 — Server-side apply (SSA)

Apply uses the dynamic client with field manager `orkestra-gateway`:

```go
client.Resource(gvr).
    Namespace(obj.GetNamespace()).
    Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{
        FieldManager: "orkestra-gateway",
        Force:        false,
    })
```

`Force: false` — the apply is rejected if another manager owns a conflicting field. This surfaces merge conflicts as structured errors rather than silently overwriting. The exact response shape for conflicts is an open question in the spec (see [gateway-apply-api-spec.md](../../../unknown/claude/sessions/post-0.7.9-improvements/undone/gateway-apply-api-spec.md)).

SSA is idempotent — re-applying the same CR body with the same field values is a no-op. Callers do not need to check "does it exist" before applying.

## Successful response

```json
{
  "status": "applied",
  "resource": {
    "kind": "Application",
    "name": "payments-api",
    "namespace": "team-payments",
    "uid": "abc-123",
    "resourceVersion": "12345"
  }
}
```

`applied` covers both create and update — the caller does not need to know which one happened.

## Polling

After apply, the CR is in Kubernetes but the operator may not have reconciled it yet. Poll `GET /api/v1/resources/{kind}/{namespace}/{name}` to check status:

```bash
# Apply
curl -X POST .../api/v1/apply -d @app.json

# Poll until Ready
until curl -s .../api/v1/resources/Application/team-payments/payments-api \
  | jq -e '.status.conditions[] | select(.type=="Ready" and .status=="True")'; do
  sleep 2
done
```

→ Next: [03-schema.md](03-schema.md)
