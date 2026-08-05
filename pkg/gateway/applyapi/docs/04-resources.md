# 04 — Resources

## Two endpoints for two audiences

The runtime already serves rich CR data via `pkg/runtime/kordinator`:

```
GET /katalog/{crd}/cr/{namespace}/{name}   — full detail: ready state, children, events
GET /katalog/{crd}/cr                      — list from informer cache
```

The Control Center uses these for its CR views. After an IDP form submit, the CC polls the runtime's `/katalog/{crd}/cr/{namespace}/{name}` for status — it already knows the runtime URL, and the response is richer (children, ready extraction, enriched status from the informer cache).

The gateway's resource endpoints exist for **external callers** — CI pipelines, Terraform providers, Slack bots — that have the gateway URL but not the runtime URL. They need to poll something after `POST /api/v1/apply`. A raw Kubernetes response (spec + status) is enough: check `status.phase` or `status.conditions`, and you know whether the operator reconciled.

## Kind, not target

Unlike `POST /api/v1/apply` and `GET /api/v1/schema`, these three endpoints resolve `{kind}` by Kubernetes Kind — `resourcesHandler` looks it up with `LookupByKind`, not `LookupByTargetOrKind`. A caller that applied with `{"target": "smartapp", ...}` needs the CR's actual Kind (`AppRequest`, say) to read, list, or delete it directly — `target` doesn't work here. In practice this rarely matters: `ApplyResponse.PollURL` already points to the correct `{kind}` path, so a caller that always follows `pollUrl` instead of assembling its own path never has to know or care about this asymmetry.

## GET /api/v1/resources/{kind}/{namespace}/{name}

Returns the current state of a CR directly from Kubernetes — spec and status as stored, via the dynamic client.

```json
{
  "apiVersion": "platform.myorg.io/v1",
  "kind": "Application",
  "metadata": {
    "name": "payments-api",
    "namespace": "team-payments",
    "resourceVersion": "12345"
  },
  "spec": {
    "environment": "production",
    "image": "ghcr.io/myorg/payments:v2.1.0",
    "replicas": 3
  },
  "status": {
    "phase": "Ready",
    "conditions": [
      {
        "type": "Ready",
        "status": "True",
        "lastTransitionTime": "2026-07-16T10:22:00Z"
      }
    ]
  }
}
```

Not enriched — no children, no ready extraction, no age. Use the runtime's `/katalog/{crd}/cr/` endpoints for the enriched view.

## GET /api/v1/resources/{kind}/{namespace}

Lists all CRs of the given kind in the namespace. Returns the same shape as the single-resource GET wrapped in an `items` array. For the enriched list view, use the runtime's `/katalog/{crd}/cr` instead.

## DELETE /api/v1/resources/{kind}/{namespace}/{name}

Deletes a CR. The runtime handles the deletion lifecycle — finalizers, ordered teardown, and cleanup — exactly as it does for `kubectl delete`.

### Deletion protection

If the CRD has deletion protection enabled, the `ValidatingWebhookConfiguration` is already registered. Kubernetes calls the deletion protection webhook before accepting the delete — the gateway's Delete handler issues the Kubernetes delete call, and Kubernetes itself rejects it if the resource is protected. The error is surfaced to the caller in the same structured format as any other rejection.

Callers get the same answer from `DELETE /api/v1/resources/...` as they would from `kubectl delete` — the webhook is the enforcement point in both cases.

## Polling patterns

Apply is synchronous up to the point where the CR is written to Kubernetes. What happens after — reconciliation, child resource creation, status transitions — is asynchronous. Poll `pollUrl` from the apply response to observe it — don't assemble `/api/v1/resources/{kind}/{ns}/{name}` by hand, the platform team may have overridden it via `idp.config.response.poll`.

### Wait for Ready

```bash
POLL_URL=$(curl -s -X POST .../api/v1/apply -d @app.json | jq -r '.pollUrl')

until curl -s -H "Authorization: Bearer $TOKEN" "${GATEWAY}${POLL_URL}" \
  | jq -e '.status.conditions[]? | select(.type=="Ready" and .status=="True")' > /dev/null 2>&1; do
  sleep 3
done
echo "Application is Ready"
```

### Wait for a specific phase

With `idp.config.response.poll.field: status.phase` set, `pollUrl` already carries `?field=status.phase` — the response is the scalar value directly, not the full CR:

```bash
until [ "$(curl -s -H "Authorization: Bearer $TOKEN" "${GATEWAY}${POLL_URL}" | jq -r '.value')" = "Deployed" ]; do
  sleep 3
done
```

### Timeout

Wrap the poll loop in a deadline. The runtime may set a `ValidationFailed` condition if admission rules reject the CR at reconcile time (not at apply time, if rules run during reconcile). A caller that only waits for `Ready: True` will loop forever if the CR is stuck. Check for error conditions too:

```bash
STATUS=$(curl -s ... | jq -r '.status.phase')
case "$STATUS" in
  Ready)    echo "done"; break ;;
  Failed)   echo "operator reported failure"; exit 1 ;;
  *)        sleep 3 ;;
esac
```

→ Next: [05-auth.md](05-auth.md)
