# 03 — Webhook Registration and Controller

## Registration

`registration.go` creates and maintains the Kubernetes webhook configuration objects that tell the API server to call Orkestra during admission.

| Configuration | Created when |
|--------------|--------------|
| `orkestra-admission-validation` (`ValidatingWebhookConfiguration`) | `HasValidationRules()` |
| `orkestra-admission-mutation` (`MutatingWebhookConfiguration`) | `HasMutationRules()` |
| `orkestra-deletion-protection` (`ValidatingWebhookConfiguration`) | `IsDeletionProtectionEnabled()` |
| `orkestra-namespace-protection` (`ValidatingWebhookConfiguration`) | `IsNamespaceProtectionEnabled()` |

All registration functions are **idempotent** — they use a Get-then-Create-or-Update pattern (`applyWebhookConfig`, `applyMutatingWebhookConfig`). Safe to call on restart or from the reconciliation controller.

### CA bundle

The CA bundle is read from `hookReg.TLSCertFile` (same certificate used by the HTTPS server). The API server uses this to trust Orkestra's TLS endpoint. If the cert file changes (e.g. cert rotation), the controller will pick up the new bundle on its next reconciliation cycle.

### Failure policy

Controlled by `WEBHOOK_FAILURE_POLICY` env var (or Katalog `security.admission.failurePolicy`):

- `Ignore` (default) — allow the operation if Orkestra is unreachable
- `Fail` — reject the operation if Orkestra is unreachable

Deletion protection always uses `Fail` — this is intentional. If Orkestra is down, protected resources cannot be deleted.

## Cleanup

`UnregisterAdmissionWebhooks` is called from `Shutdown()` with `cleanupOnShutdown` flag from the Katalog:

```go
// In Shutdown():
if kat.HasMutationRules() {
    cleanupOpts.mutating = kat.DeletionProtectionCleanupOnShutdown()
}
if kat.HasValidationRules() {
    cleanupOpts.validating = kat.DeletionProtectionCleanupOnShutdown()
}
UnregisterAdmissionWebhooks(ctx, ws.kubeClient, cleanupOpts)
```

By default, webhook configurations persist across pod restarts. Set `cleanupOnShutdown: true` to remove them on graceful shutdown.

## Housekeeper

`housekeeper.go` runs a reconciliation loop that keeps webhook configurations in sync with the Katalog, independent of pod lifecycle.

```
housekeeper(ctx)
  goroutine:
    for {
      reconcileAdmissionWebhooks()
      reconcileDeletionProtectionWebhook()
      reconcileNamespaceProtectionWebhook()
      wait for: ctx.Done() (shutdown) or ticker.C (next interval)
    }
```

Sync interval is controlled by `kat.HousekeeperSyncInterval()`.

Each reconcile sub-function:
- If the capability is **enabled**: calls the appropriate register function (idempotent).
- If the capability is **disabled**: calls the cleanup function to remove the webhook config if it exists.

This ensures that disabling a webhook in the Katalog and redeploying the operator removes the webhook configuration from the cluster, even without pod restart.

→ Next: [04-tls.md](04-tls.md)
