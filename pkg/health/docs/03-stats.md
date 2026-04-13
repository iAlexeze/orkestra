# 03 — Stats

Both webhook types keep in-memory stats that are exposed through the `/katalog/{crd}` endpoint in the `conversion` and `admission` keys.

## Rolling window

Stats use a ring buffer, not an unbounded slice. The window size is controlled by the `CONVERSION_WINDOW` config value (shared between conversion and admission). The default is 1000 requests.

The ring buffer is used only for percentile calculations. Running totals (`Total`, `Success`, `Failures`) are unbounded counters and survive window rollover.

## ConversionStats

```go
cs := NewConversionStats(windowSize)

cs.RecordSuccess(duration) // success + latency
cs.RecordFailure()         // failure, no latency

snap := cs.GetStats()
// snap.TotalRequests   int64
// snap.SuccessRequests int64
// snap.FailedRequests  int64
// snap.AvgLatency      time.Duration
// snap.P95Latency      time.Duration
// snap.MaxLatency      time.Duration
// snap.MinLatency      time.Duration
```

Exposed in JSON as `milliseconds float64`:

```json
{
  "conversion": {
    "enabled": true,
    "total": 1200,
    "success": 1195,
    "failures": 5,
    "avgLatencyMs": 2.4,
    "p95LatencyMs": 8.1
  }
}
```

## AdmissionStats

Tracks validation and mutation separately. One `AdmissionStats` instance lives on the server and accumulates across all CRDs. Per-CRD granularity is available via Prometheus (the `crd` label on metric series).

```go
as := NewAdmissionStats(windowSize)

// Validation outcomes
as.RecordValidationAllowed(duration)
as.RecordValidationDenied(duration)
as.RecordValidationWarned(duration)

// Mutation outcomes
as.RecordMutationApplied(duration)
as.RecordMutationSkipped(duration)  // no-op — no rules matched

snap := as.GetStats(webhooksEnabled)
```

Exposed in JSON:

```json
{
  "admission": {
    "webhooksEnabled": true,
    "validationTotal": 800,
    "validationAllowed": 795,
    "validationDenied": 3,
    "validationWarned": 2,
    "valAvgLatencyMs": 1.2,
    "valP95LatencyMs": 4.8,
    "valMaxLatencyMs": 22.0,
    "mutationTotal": 800,
    "mutationApplied": 312,
    "mutationSkipped": 488,
    "mutAvgLatencyMs": 0.9,
    "mutP95LatencyMs": 3.1,
    "mutMaxLatencyMs": 11.0
  }
}
```

## Percentile algorithm

Both types use the same approach: copy the ring buffer contents, sort, pick by index. Called under a read lock — no allocation is hidden behind the lock (the sort is on a local copy).

```
idx = floor(windowSize × percentile)
```

For a 1000-request window, P95 reads position 950 in the sorted copy.

→ Back to: [README.md](../README.md)
