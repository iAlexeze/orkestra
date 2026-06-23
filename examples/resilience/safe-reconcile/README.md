# Safe Reconcile — Panic Recovery

A reconciler bug — a nil pointer, an unchecked slice index, a type assertion without a guard — would crash a standard operator process and take everything down with it. Every CRD it was managing would stop reconciling until someone noticed and restarted the pod.

Orkestra wraps every reconcile call in `safeReconcile`. If the reconciler panics, the panic is caught, logged with a full stack trace, and converted into a reconcile error. The workqueue re-queues the item with backoff. The worker goroutine continues. All other CRDs keep reconciling without interruption.

**What you learn:** How `safeReconcile` works, what you see streaming in real time when a panic occurs, and how declarative CRDs are isolated from a panicking typed hook.

**Requirement:** `ork` CLI — install from [orkestra-install](https://github.com/orkspace/orkestra#getting-started)

---

## What's in this example

| CRD | Kind | Type | Behaviour |
|-----|------|------|-----------|
| `monitors.safe.demo.orkestra.io` | Monitor | Declarative | Creates a ConfigMap. Reconciles cleanly. |
| `queues.safe.demo.orkestra.io` | Queue | Declarative | Creates a ConfigMap. Reconciles cleanly. |
| `apps.safe.demo.orkestra.io` | App | Typed (hooks) | Hook dereferences a nil pointer. Panics on every reconcile. |

The App CR deliberately omits the optional `spec.config` field. The hook accesses `obj.Spec.Config.Endpoint` without a nil check. This panics.

- `api/v1alpha1/app_types.go` — the Go structs for the App CRD.
- `hooks/app_hooks.go` — the `onAppReconcile` hook with the intentional nil dereference.
- `katalog.yaml` — declares all three CRDs.
- `Makefile` — registry, build, validate, simulate targets.

---

## Step 1 — Generate the registry and entrypoint

```bash
make registry
```

Generates `cmd/orkestra/main.go` and `pkg/typeregistry/zz_generated_typeregistry.go` from `katalog.yaml`. Re-run whenever you change `apiTypes` fields.

---

## Step 2 — Build

```bash
make build
```

Builds a binary that includes your generated type registry and replaces `~/.orkestra/bin/ork`. `ork` now knows the App type.

---

## Step 3 — Validate

```bash
ork validate
```

Expected output:

```
Validating Katalog...

● monitor  kind: Monitor  / group: safe.demo.orkestra.io / ...
● queue    kind: Queue    / group: safe.demo.orkestra.io / ...
● app      kind: App      / group: safe.demo.orkestra.io / ...

3 CRDs valid (0 built-in, 3 custom)
```

---

## Step 3b — Simulate

```bash
ork simulate
```

Runs three simulations without a cluster:

- **Monitor** — ConfigMap created in cycle 1, steady state, no errors.
- **Queue** — ConfigMap created in cycle 1, steady state, no errors.
- **App** — shows the panic on cycle 1.

---

## Step 4 — Start the Runtime

> `ork run --dev` spins up a local kind cluster and starts the 3 operators. Skip `--dev` if you already have a cluster running.

```bash
ork run --dev
```

The runtime starts and streams logs directly to your terminal. Keep this running.

---

## Step 5 — Apply the CRDs and CRs

In a separate terminal:

```bash
kubectl apply -f crd-declarative.yaml
kubectl apply -f crd-typed.yaml

kubectl apply -f cr-monitor.yaml
kubectl apply -f cr-queue.yaml
kubectl apply -f cr-app.yaml
```

---

## Step 6 — Watch the panic stream in real time

Back in the `ork run` terminal, Monitor and Queue reconcile cleanly:

```json
{"level":"info"...,"time":1782183002,"message":"reconciled safe.demo.orkestra.io/v1alpha1, Kind=Monitor"}
{"level":"info"...,"time":1782183002,"message":"reconciled safe.demo.orkestra.io/v1alpha1, Kind=Queue"}
```

Then the App CR panics:

```json
{
  "level": "error",
  "gvk": "apps.safe.demo.orkestra.io",
  "key": "default/my-app",
  "panic": "runtime error: invalid memory address or nil pointer dereference",
  "stack": "goroutine 42 [running]:\ngithub.com/orkspace/safe-reconcile-demo/hooks.onAppReconcile(...)\n\thooks/app_hooks.go:35 +0x...",
  "message": "reconciler panic recovered"
}
```

The `ork run` process does not crash. The next line in the stream shows Monitor or Queue reconciling again as normal. The panic is isolated to the App CR's reconcile cycle.

---

## Step 7 — Observe in the Control Center

In a third terminal:

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081).

Click the **App** CRD. The health panel shows:

- **Status**: Degraded (once consecutive failures exceed the threshold)
- **Consecutive failures**: incrementing on each reconcile cycle
- **Last error**: `reconciler panic: runtime error: invalid memory address or nil pointer dereference`
- **Stack trace**: the full goroutine stack, pinpointing `hooks/app_hooks.go`

Click **Monitor** or **Queue**. Their health panels show **Healthy** — successes accumulating, zero failures. The isolation is visible directly in the UI.

---

## Step 8 — Observe the metrics

```bash
curl -s http://localhost:8080/metrics | grep reconcile_total
```

```text
controller_reconcile_total{crd="safe.demo.orkestra.io/v1alpha1, Kind=App",result="error"} 3
controller_reconcile_total{crd="safe.demo.orkestra.io/v1alpha1, Kind=Monitor",result="success"} 9
controller_reconcile_total{crd="safe.demo.orkestra.io/v1alpha1, Kind=Queue",result="success"} 11
```

Monitor and Queue accumulate successes. App accumulates errors. All three keep reconciling — each on its own independent workqueue.

---

## Build and push the production image

> This is required by `ork e2e`

```bash
export IMAGE_REPO=ghcr.io/myorg/safe-reconcile-demo
export IMAGE_TAG=1.0.0
make release
```

`make release` compiles with the `runtime` build tag (no validate/simulate/e2e commands), builds the distroless image, and pushes it.

## Update [values.yaml](values.yaml) with your image

The e2e gate runs automatically during push and needs to pull your custom runtime image. Update [values.yaml](values.yaml) to point to the image you just built:

```yaml
runtime:
  image:
    repository: ghcr.io/myorg/safe-reconcile-demo
    tag: 1.0.0
```
---

## E2E

```bash
ork e2e --use-current
```

This runs everything defined in [e2e.yaml](./e2e.yaml): `--use-current` reuses the current cluster, starts the runtime, applies the CRs, asserts that Monitor and Queue ConfigMaps are created, asserts that orkestra is still running despite the App panic, and checks for the `reconciler panic recovered` log line.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
