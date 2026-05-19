# 01 — Overview

The `cmd/internal` package is the wiring layer for Orkestra. It assembles every
komponent from its constituent packages, threads dependencies together as Go
pointers and closures, and hands the result to `pkg/orkestra` — the supervisor
that calls `Start()` and `Stop()` in the correct order.

No business logic lives here. `cmd/internal` knows how to wire; the packages it
imports know how to run.

## Two entrypoints

`cmd/internal` exposes exactly two public functions.

`Konduct` is the runtime entrypoint. It calls `konstructOrkestra`, which assembles
the full reconcile loop — informers, queues, reconcilers, health server, and the
dependency kordinator. `Konduct` then starts leader election via `pkg/konductor`
so only one replica runs the reconcile loop at a time. It is called by `ork run`.

`KonductGateway` is the gateway entrypoint. It assembles only the parts needed to
serve TLS and admission webhooks — the health server, the webhook server, and the
kubeclient. It does not run reconcilers and does not participate in leader election.
It is called by `ork gateway`. The gateway exits immediately if not running inside
a Kubernetes pod.

## The split

Runtime and gateway are separate OS processes with different resource and
availability profiles.

The runtime is leader-elected: only one replica reconciles at a time, though
multiple replicas may be running and waiting for the lease. Reconcilers are
stateful in the sense that each CRD has an in-memory informer cache and a
bounded work queue.

The gateway is stateless: every replica serves the same webhooks. Multiple
replicas run simultaneously without coordination. Because there is no in-memory
state to lose, the gateway can be rolled, scaled, or restarted at any time
without affecting in-flight reconciles.

In Helm terms, `runtime` and `gateway` are two separate Deployments. The runtime
Deployment exists by default; the gateway Deployment is opt-in
(`gateway.enabled: true` in `values.yaml`).

```
ork run      →  Konduct        →  konstructOrkestra  →  reconcile loop
ork gateway  →  KonductGateway →  gateway wiring     →  TLS + webhooks
```

→ Next: [02-runtime.md](02-runtime.md)
