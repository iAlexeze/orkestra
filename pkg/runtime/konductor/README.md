# pkg/runtime/konductor

`konductor` wraps Kubernetes leader election so that only one Orkestra runtime pod processes reconcile events at a time. When a pod wins the election it becomes the "konductor" and its `run` function is called. When it loses the lease, the run context is cancelled and the controllers shut down cleanly.

```go
ko := konductor.NewKonductorElection(kube, event, run, onElected, konductor.Options{
    Namespace:     kfg.Konductor().Namespace,
    LeaseDuration: kfg.Konductor().LeaseDuration,
    RenewDeadline: kfg.Konductor().RenewDeadline,
    RetryPeriod:   kfg.Konductor().RetryPeriod,
})

ko.Start(ctx)
```

## How it works

`KonductorElection` uses `client-go/tools/leaderelection` backed by a `coordination.k8s.io/v1 Lease` object. Three callbacks drive the lifecycle:

| Callback | Triggered when | Action |
|----------|----------------|--------|
| `OnStartedLeading` | This pod wins the lease | Calls `run(ctx)` with a cancellable context; records a Kubernetes event |
| `OnStoppedLeading` | This pod loses the lease | Cancels the run context; clears the konductor identity |
| `OnNewLeader` | Any pod wins | Logs the new leader identity |

`ReleaseOnCancel: true` ensures the Lease is released when `Shutdown()` is called, allowing a new pod to immediately become konductor rather than waiting for the lease to expire.

## Identity

Each pod identifies itself by hostname (`os.Hostname()`). If the hostname call fails, a UUID is generated as a fallback. The identity appears in `Konductor()` after election and in the emitted Kubernetes events.

## Lean election timing

Configured via `pkg/konfig` ENV vars:

| ENV | Default | Purpose |
|-----|---------|---------|
| `LEASE_DURATION` | 60s | How long a lease is valid without renewal |
| `RENEW_DEADLINE` | 40s | How long the leader tries to renew before giving up |
| `RETRY_PERIOD` | 10s | How often non-leaders try to acquire the lease |

For faster failover in testing, use the defaults in `konfig.NewDefaultKonfig()` (15s/10s/2s).
