# Panic Recovery

Every reconcile call runs inside `safeReconcile`. This is the panic isolation boundary. If your hook, constructor, or any code it calls panics — nil pointer dereference, out-of-bounds slice access, failed type assertion without a guard — the panic does not escape the operatorBox. It is caught, logged with a full stack trace, and treated as a reconcile failure. The operator process continues running. Every other operatorBox keeps reconciling without interruption.

---

## How it works

`safeReconcile` wraps the reconcile call in a deferred function that calls `recover()`:

```go
defer func() {
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

        health.RecordFailure(err, failureThreshold)
        metrics.RecordReconcile(gvk, "error")
    }
}()

err = rec.Reconcile(ctx, key)
```

`recover()` only works inside a `defer` in the same goroutine as the panicking call. The named return `err` is set through the deferred function, so the caller sees a non-nil error — the workqueue re-queues the item with rate-limit backoff identically to a regular reconcile failure.

---

## What happens

When a panic occurs:

1. The panic unwinds the stack to `safeReconcile`'s deferred function.
2. `recover()` captures the panic value.
3. `runtime.Stack` captures the full goroutine stack trace.
4. The panic is logged at `ERROR` level with GVK, key, panic value, and stack trace.
5. The consecutive-failure counter for this operatorBox increments.
6. If the counter exceeds the failure threshold (default: 5), the operatorBox enters degraded state — see [Error Behavior](02-error-behavior.md#degraded-state).
7. `ork_reconcile_total{result="error"}` increments.
8. The workqueue re-queues the key with exponential backoff.

The worker goroutine is unaffected. It dequeues the next item normally. Other operatorBoxes running their own workers on their own queues are completely isolated — they are not aware the panic happened.

---

## What you see

**Logs:**

```json
{
  "level": "error",
  "gvk": "apps.safe.demo.orkestra.io",
  "key": "default/my-app",
  "panic": "runtime error: invalid memory address or nil pointer dereference",
  "stack": "goroutine 42 [running]:\nmain.onAppReconcile(...)\n\thooks/app_hooks.go:35 +0x2c\n...",
  "message": "reconciler panic recovered"
}
```

**Metrics:**

```
ork_reconcile_total{gvk="apps.safe.demo.orkestra.io",result="error"} 1
```

Other operatorBoxes keep accumulating successes on their own label:

```
ork_reconcile_total{gvk="monitors.safe.demo.orkestra.io",result="success"} 4
ork_reconcile_total{gvk="queues.safe.demo.orkestra.io",result="success"}   4
```

---

## The queue.Done guarantee

The worker loop wraps both `processItemForGVK` and `safeReconcile` inside a closure:

```go
func() {
    defer wq.Queue.Done(item)
    k.processItemForGVK(ctx, gvk, item)
}()
```

`wq.Queue.Done(item)` runs regardless of what happens inside — including panics that occur before `safeReconcile` can catch them. Without this, a panic between the worker and `safeReconcile` would permanently remove the item from the queue without marking it done, causing the workqueue's internal tracking to diverge.

---

## Common causes

| Panic | Likely cause |
|-------|--------------|
| `nil pointer dereference` | Optional pointer field (`*T`) not checked before use |
| `index out of range` | Slice access without bounds check |
| `interface conversion` | Type assertion without the two-return form (`v, ok := x.(T)`) |
| `send on closed channel` | Channel closed before all senders have finished |

All of these are caught by `safeReconcile`. None crash the operator.

---

## Try it

```bash
ork init --pack security/safe-reconcile
cd safe-reconcile

# Follow the steps in the README
```

This example demonstrates safeReconcile in action with a live operator. Two declarative CRDs (Monitor, Queue) reconcile cleanly. One typed CRD (App) has a nil pointer dereference in its hook — `obj.Spec.Config.Endpoint` where `Spec.Config` is nil. Apply the App CR and watch the panic appear in logs while Monitor and Queue keep reconciling without interruption.

---

**← Back to** [Error Behavior](02-error-behavior.md)
