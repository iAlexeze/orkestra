# 02 — Stats Types

The health package defines the stats types that the webhook package writes to and the kordinator reads when building the `/katalog/{crd}` API response. Keeping these types here avoids importing `pkg/webhook` from `pkg/kordinator`.

## Rolling window

All latency-bearing stats use a ring buffer for percentile calculations. The window size is controlled by `CONVERSION_WINDOW` (default 1000 requests). Running totals (`Total`, `Success`, `Failures`) are unbounded counters.

## ConversionStats

```go
cs := health.NewConversionStats(windowSize)

cs.RecordSuccess(duration)
cs.RecordFailure()

snap := cs.GetStats()
// snap.TotalRequests   int64
// snap.SuccessRequests int64
// snap.FailedRequests  int64
// snap.AvgLatency      time.Duration
// snap.P95Latency      time.Duration
// snap.MaxLatency      time.Duration
```

## AdmissionStats

Tracks validation and mutation separately. One instance covers all CRDs.

```go
as := health.NewAdmissionStats(windowSize)

as.RecordValidationAllowed(duration)
as.RecordValidationDenied(duration)
as.RecordValidationWarned(duration)

as.RecordMutationApplied(duration)
as.RecordMutationSkipped(duration)

snap := as.GetStats(webhooksEnabled)
```

## DeletionProtectionStats

```go
ps := health.NewDeletionProtectionStats()

ps.RecordBlocked()
ps.RecordAllowed()

snap := ps.GetStats()
// snap.TotalRequests, snap.Blocked, snap.Allowed
```

## NamespaceProtectionStats

Same pattern as `DeletionProtectionStats` — counters for `Blocked` and `Allowed`.

## WebhookStats

Tracks webhook controller reconciliation cycles.

```go
ws := health.NewWebhookStats()

ws.RecordReconciled()
ws.RecordFailure()

snap := ws.GetStats()
// snap.Reconciled, snap.Failed
```

## Ownership

The `pkg/webhook` package holds the live instances and writes to them. The `pkg/kordinator` package reads snapshots to build the `/katalog/{crd}` response. This package defines the types.

→ Next: [03-routes.md](03-routes.md)
