# 04 — Slack

Slack is the one source where the apply doesn't happen inline in the request that triggered it. Slack requires an HTTP response within 3 seconds of the slash command, and an `ApplyTargetFields` call — permission check, CR construction, admission, SSA — routinely takes longer than that under load. `slack.go` acks first and applies in the background.

## The pipeline

```
verify X-Slack-Signature (HMAC over v0:{timestamp}:{body}, ±5min window)
  → parse form body (command, text, response_url)
  → check command against Commands
  → parse "<target> key=value ..." from text  (args.go: ParseSlackArgs)
  → ack immediately: 200, "Deploying <target>..."
  → submit to worker pool:
      api.ApplyTargetFields(...)
      slack.PostMessage(response_url, outcome)
```

Everything above the ack line runs synchronously, in the original request. Everything below runs on a separate goroutine, after the handler has already returned.

## The context bug this avoids

The submitted job does **not** use `r.Context()`:

```go
pool.Submit(context.Background(), func(_ context.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), slackApplyTimeout)
    defer cancel()
    resp, status := api.ApplyTargetFields(ctx, kube, kat, notes, src.Config.Name, fields, false)
    ...
})
```

`r.Context()` is canceled the instant `ServeHTTP` returns — which happens right after `writeSlackAck` writes the 200. Using it for the background apply would mean every single background apply fails with "context canceled," always, regardless of how fast or slow the apply actually is, because the cancellation races the goroutine's very first line. The fix is a fresh `context.Background()` with its own 30-second timeout (`slackApplyTimeout`), independent of the HTTP request's lifecycle. This was deliberate, not incidental — worth knowing before "simplifying" this to `r.Context()` in a future change.

## The worker pool exists for one reason: Slack retries

Slack retries a slash command that doesn't ack fast enough. Without a bound, a retry storm during an incident (exactly when `/deploy` traffic spikes) could spawn unbounded concurrent applies against the cluster. `pool.go`'s `workerPool` caps concurrency at `slackWorkerPoolSize` (8): `Submit` blocks the *caller* only long enough to acquire a semaphore slot, not until the job finishes, and recovers panics inside the spawned goroutine so one bad job can't take the pool down or leak its slot.

`Submit`'s blocking-when-full behavior is intentional backpressure, not a bug — but it means a test that submits more jobs than the pool's capacity from a single goroutine, expecting to then release them, will deadlock: the submitting goroutine can't reach the release step because it's still blocked on an earlier `Submit` call waiting for a slot that only frees once the release happens. `pool_test.go`'s `TestWorkerPool_CapsConcurrency` submits from per-job goroutines specifically to avoid this.

## Argument parsing has no structure beyond key=value

`ParseSlackArgs` (`args.go`) is deliberately minimal: the first token is the target, everything after is `key=value`, split on the first `=`. No quoting, no escaping, no nested structures — Slack slash command text is a single line a human typed, and the target-mode intent it needs to produce is already flat. Anything more structured than `<target> key=value key=value` belongs in a real intent file delivered by GitHub/GitLab, not a chat command.

## Unknown command and invalid args still ack `200`

`commandAllowed` and `ParseSlackArgs` failures both call `writeSlackAck` with an explanatory message, not `writeJSONError` — Slack always expects a `200` with a `text` field, even for a rejection the user needs to see in their client. Only signature verification failure returns `401`.

→ Next: [05-testing.md](05-testing.md)
