# Design Document: Event-Driven Webhook Controller

## Status

Proposed

## Problem

The current webhook controller uses a fixed poll interval (`WEBHOOK_CONTROLLER_SYNC_INTERVAL`, default 30 s, minimum 1 s) to detect and repair missing or drifted `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` objects.

```
startup → tick every N seconds → reconcileAdmissionWebhooks()
                                → reconcileDeletionProtectionWebhook()
                                → reconcileNamespaceProtectionWebhook()
```

This works, but carries a structural gap: **between two poll ticks, a webhook can be absent**. During that window:

- A deletion-protected CR can be deleted
- A namespace-restricted CR can be applied to a forbidden namespace
- Any other admission invariant the webhook enforces is suspended

Setting `WEBHOOK_CONTROLLER_SYNC_INTERVAL=1` reduces the window to ~1 second, but even 1 second is measurable and requires sacrificing a reconcile API call per second even when nothing has changed. For high-security environments this is not the right trade-off.

The correct model is **event-driven**: react to webhook deletion the moment it happens, not at the next tick.

---

## Current Implementation

`webhookController` in `pkg/webhook/controller.go` runs a single goroutine with a `time.NewTicker`. On each tick it calls three reconcile functions:

```go
func (ws *WebhookServer) webhookController(ctx context.Context) error {
    go func() {
        ticker := time.NewTicker(kat.WebhookControllerSyncInterval())
        defer ticker.Stop()
        for {
            ws.reconcileAdmissionWebhooks()
            ws.reconcileDeletionProtectionWebhook()
            ws.reconcileNamespaceProtectionWebhook()
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
            }
        }
    }()
    return nil
}
```

Each reconcile function calls `RegisterXxxWebhook` which does a Create-or-Update against the API server. If the configuration already matches, the call is effectively a no-op at the etcd level (Kubernetes does not write if the object is unchanged), but the API round-trip still happens every tick.

---

## Proposed Architecture: Hybrid Watch + Baseline Poll

### Design principle

Use a **Kubernetes Watch** on `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` as the **fast path** for detection. Keep a **long-interval poll** (e.g. 5 minutes) as a **safety net** for drift that Watch silently misses (partial mutations, token expiry, silent stream drops).

```
         ┌─────────────────────────────────────────┐
         │          webhookController               │
         │                                          │
         │   ┌─────────┐     ┌──────────────────┐  │
         │   │  Watch  │────▶│  trigger channel  │  │
         │   │ (fast)  │     └────────┬─────────┘  │
         │   └─────────┘             │              │
         │                           ▼              │
         │   ┌─────────┐     ┌──────────────────┐  │
         │   │ 5m poll │────▶│   reconcileAll() │  │
         │   │ (safety)│     └──────────────────┘  │
         │   └─────────┘                           │
         └─────────────────────────────────────────┘
```

### Fast path — Watch

Start a `Watch` on `ValidatingWebhookConfiguration` filtered by a label selector that matches Orkestra-owned webhooks. On any `DELETED` or `MODIFIED` event, send a non-blocking signal to a trigger channel that the reconcile loop drains.

```go
func (ws *WebhookServer) watchValidatingWebhooks(ctx context.Context, trigger chan<- struct{}) {
    for {
        watcher, err := ws.kubeClient.AdmissionregistrationV1().
            ValidatingWebhookConfigurations().
            Watch(ctx, metav1.ListOptions{
                LabelSelector: "app.kubernetes.io/managed-by=orkestra",
            })
        if err != nil {
            logger.Warn().Err(err).Msg("webhook watch: failed to start, retrying in 5s")
            select {
            case <-ctx.Done():
                return
            case <-time.After(5 * time.Second):
                continue
            }
        }

        for event := range watcher.ResultChan() {
            switch event.Type {
            case watch.Deleted, watch.Modified:
                select {
                case trigger <- struct{}{}: // signal reconcile needed
                default: // already pending, drop duplicate
                }
            }
        }

        // ResultChan closed — Watch expired or network error. Retry.
        watcher.Stop()
        select {
        case <-ctx.Done():
            return
        case <-time.After(time.Second):
        }
    }
}
```

Do the same for `MutatingWebhookConfiguration`.

### Reconcile loop

Replace the fixed ticker with a select that drains both the trigger channel and a long-interval safety ticker:

```go
func (ws *WebhookServer) webhookController(ctx context.Context) error {
    trigger := make(chan struct{}, 1) // buffered: coalesce bursts
    safetyTicker := time.NewTicker(5 * time.Minute)
    defer safetyTicker.Stop()

    go ws.watchValidatingWebhooks(ctx, trigger)
    go ws.watchMutatingWebhooks(ctx, trigger)

    go func() {
        // Reconcile immediately on startup.
        ws.reconcileAll()

        for {
            select {
            case <-ctx.Done():
                return
            case <-trigger:
                ws.reconcileAll()
            case <-safetyTicker.C:
                ws.reconcileAll()
            }
        }
    }()

    logger.Info().Msg("webhook controller started (event-driven)")
    return nil
}
```

The channel buffer of 1 means any number of rapid DELETE/MODIFY events coalesce into a single reconcile call. The reconcile itself is idempotent, so double-firing is harmless.

---

## Key Implementation Details

### Watch reconnection

The Kubernetes Watch API closes the result channel after the server's `timeout` expires (default 5–6 minutes) or on any network error. The watch goroutine must loop and re-establish the watch. Use `resourceVersion: ""` or the last-seen `resourceVersion` to avoid replaying all history on reconnect:

```go
// Track the resourceVersion to resume from on reconnect.
var rv string
watcher, err := client.Watch(ctx, metav1.ListOptions{
    ResourceVersion: rv,
    LabelSelector:   "...",
})
// After each event:
rv = event.Object.(metav1.Object).GetResourceVersion()
```

### Ownership identification

Orkestra-owned webhooks are identified by the label `app.kubernetes.io/managed-by=orkestra` (set by `RegisterXxxWebhook`). Only events for those resources trigger reconciliation. Events for unrelated webhooks in the same cluster are ignored at the Watch filter level.

### Leader election

The webhook controller must only run on the elected leader. This is already the case in the current implementation (called from `kordinator` after leader election). No change needed — the Watch goroutines inherit the leader's context and stop when leadership is lost.

### Debounce

The buffered trigger channel (capacity 1) handles high-frequency events naturally. A user deleting and recreating the webhook repeatedly only enqueues one reconcile at a time. No explicit debounce timer is needed.

### Interaction with the baseline poll

The 5-minute safety poll handles scenarios the Watch cannot:

- **Silent stream expiry**: Some Kubernetes distributions silently drop Watch streams without closing the channel (observed on some managed clusters). The poll catches drift that the Watch missed.
- **Partial mutations**: If someone patches a webhook's rules rather than deleting it, the Watch fires a `MODIFIED` event and reconcile runs. The poll is redundant but harmless.
- **Token rotation**: If the watch goroutine fails to re-establish due to an expired token, the poll continues to correct drift until the watch recovers.

The poll interval can be made configurable separately from the Watch reconnect interval:

```
WEBHOOK_WATCH_SAFETY_INTERVAL=300   # safety poll in seconds (default 300)
```

The existing `WEBHOOK_CONTROLLER_SYNC_INTERVAL` becomes the safety poll interval in the hybrid model, preserving backward compatibility.

---

## Failure Modes and Mitigations

| Failure | Current behaviour | With event-driven |
|---------|------------------|-------------------|
| Webhook deleted | Detected at next tick (≤ N seconds) | Detected immediately (Watch event) |
| Watch stream silently dropped | N/A | Safety poll catches drift within 5 min |
| Orkestra pod restart (rolling) | New leader restores webhook at next tick | New leader restores webhook on startup + Watch |
| API server unreachable | Reconcile logs error, retries at next tick | Watch goroutine retries with backoff; poll retries at next tick |
| Mass concurrent deletes | Each tick repairs one-by-one | Channel coalesces into one reconcile; all repaired in one call |

---

## Migration Path

1. Add Watch goroutines alongside the existing ticker (no removal yet).
2. Add a feature flag `WEBHOOK_CONTROLLER_EVENT_DRIVEN=true` to enable the Watch path.
3. Change the ticker to a 5-minute safety interval when the Watch is active.
4. Remove the feature flag and make event-driven the only mode once validated in production.

The existing `reconcileAll()` functions (`reconcileAdmissionWebhooks`, `reconcileDeletionProtectionWebhook`, `reconcileNamespaceProtectionWebhook`) require no changes — they are called identically from both the ticker and the Watch trigger.

---

## Non-Goals

- **Replacing the reconcile functions**: The event-driven change is purely in the detection layer. How webhooks are created/updated is unchanged.
- **Watching individual webhook rules**: The Watch operates at the `ValidatingWebhookConfiguration` object level. Rule-level drift within a configuration object triggers a `MODIFIED` event, which is sufficient.
- **Cross-cluster watches**: Out of scope. Each Orkestra instance watches its own cluster.
