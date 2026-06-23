# 08 — safeReconcile — Panic Isolation

## Purpose

`safeReconcile` is the isolation boundary between operator code and the Orkestra worker pool. Every call to `rec.Reconcile(ctx, key)` goes through it. If the reconciler panics — nil pointer dereference, out-of-bounds slice access, failed type assertion — the panic is caught inside `safeReconcile`, logged with a full stack trace, and converted into a reconcile error. The worker goroutine continues. All other CRDs keep reconciling without interruption.

A standard operator process that panics in a reconciler dies. Orkestra operators do not.

## Implementation

```go
func (k *Kontroller) safeReconcile(
    rec domain.Reconciler,
    health *CRDHealth,
    ctx context.Context,
    key string,
    gvk string,
) (err error) {

    start := time.Now()
    defer func() {
        metrics.ObserveReconcileDuration(gvk, time.Since(start).Seconds())

        if r := recover(); r != nil {
            buf := make([]byte, 4096)
            n := runtime.Stack(buf, false)

            err = fmt.Errorf("reconciler panic: %v", r)

            logger.Error().
                Str("gvk", gvk).
                Str("key", key).
                Str("panic", fmt.Sprint(r)).
                Str("stack", string(buf[:n])).
                Msg("reconciler panic recovered")

            health.RecordFailure(err, k.failureThreshold[gvk])
            metrics.RecordReconcile(gvk, "error")
        }
    }()

    err = rec.Reconcile(ctx, key)
    if err != nil {
        health.RecordFailure(err, k.failureThreshold[gvk])
        metrics.RecordReconcile(gvk, "error")
        return err
    }

    health.RecordSuccess()
    k.successReconcile(gvk)
    metrics.RecordReconcile(gvk, "success")
    return nil
}
```

The `recover()` call is inside a `defer`. This is the only place in Go where `recover()` catches a panic from a called function — a deferred function in the same goroutine as the panicking call.

## What happens when a panic occurs

1. The panic unwinds the stack to the nearest `defer` with a `recover()` call.
2. `recover()` returns the panic value (usually a runtime error string).
3. `runtime.Stack` captures the goroutine's stack trace at that point.
4. The stack trace is logged at `error` level alongside the GVK and key.
5. `err` is set (via the named return) — the caller sees a non-nil error.
6. `health.RecordFailure` increments the consecutive failure counter for this CRD. If it exceeds `failureThreshold`, the CRD's health state transitions to `Degraded`.
7. `metrics.RecordReconcile(gvk, "error")` increments `ork_reconcile_total{result="error"}`.
8. The worker returns the error to `processItemForGVK`, which re-queues the key with rate-limit backoff.

The worker goroutine is unaffected. The queue continues draining. Other CRDs running their own workers on their own queues are completely isolated.

## Isolation scope

Each CRD has its own worker pool and its own queue. A panic in a hook for CRD A does not affect:

- Workers for CRD B or CRD C
- The CRD B or CRD C queues
- The informer caches
- The HTTP handler serving `/katalog` and `/health`
- The leader election lease

The only thing affected is the specific reconcile cycle for the panicking key. It is re-queued with backoff and retried.

## What you see

**Logs:**

```json
{
  "level": "error",
  "gvk": "apps.safe.demo.orkestra.io",
  "key": "default/my-app",
  "panic": "runtime error: invalid memory address or nil pointer dereference",
  "stack": "goroutine 42 [running]:\n...",
  "message": "reconciler panic recovered"
}
```

**Metrics:**

```
ork_reconcile_total{gvk="apps.safe.demo.orkestra.io",result="error"} 1
```

**Health:** The CRD's consecutive failure counter increments. If it exceeds `failureThreshold` (default: 5), the CRD transitions to `Degraded` — visible in the Control Center and in the `/katalog/{crd}/health` endpoint response.

## The `wq.Queue.Done(item)` guarantee

`safeReconcile` is called from inside a deferred closure in the worker loop:

```go
func() {
    defer wq.Queue.Done(item)
    k.processItemForGVK(ctx, gvk, item)
}()
```

`wq.Queue.Done(item)` runs regardless of whether `processItemForGVK` returns normally or panics (before `safeReconcile` can catch it). This prevents the workqueue's internal tracking from diverging — an item is always marked done, even if something unexpected happens between the worker and `safeReconcile`.

See [05 — Workers and drain](05-workers.md) for the full worker loop.

## Tryit

```bash
ork init --pack security/safe-reconcile
cd safe-reconcile

# Follow the steps in the README
```

This example demonstrates safe-reconcile in action with a live operator. Two declarative CRDs (Monitor, Queue) reconcile cleanly. One typed CRD (App) has a nil pointer dereference in its hook. Apply the App CR and watch the panic in logs while Monitor and Queue keep reconciling without interruption.

---

**← Back to** [07 — Gateway stats](07-gateway-stats.md)
