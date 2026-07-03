# CRD Missing Recovery — Runtime CRD Watch Without Deletion Protection

Orkestra watches every CRD it manages throughout its lifetime — not just at startup. If a CRD is deleted at runtime, the operator detects the disappearance, marks it missing, degrades, and retries in a loop. When the CRD is restored and a CR is reapplied, the operator recovers automatically with no restart.

This example shows that arc without the gateway's deletion protection in place. Deletion protection prevents the CRD from being deleted in the first place — this example shows what the runtime does when that layer is absent.

**What you learn:** How Orkestra monitors CRD existence at runtime, what the health endpoint and Control Center show during a missing-CRD degradation, and how the operator self-heals when the CRD comes back.

---

## What's in this example

| File | Purpose |
|------|---------|
| `katalog.yaml` | BlockchainApp operator — no validation rules, focused on CRD watch |
| `crd.yaml` | BlockchainApp CRD |
| `cr.yaml` | Valid BlockchainApp CR |

---

## Step 1 — Validate

```bash
ork validate
```

Expected output:

```text
● blockchainapp
    kind: BlockchainApp
    group: resilience.demo.orkestra.io / version: v1alpha1 / plural: blockchainapps
    mode: dynamic / workers: 3 / resync: 15s
```

---

## Step 2 — Simulate

```bash
ork simulate
```

Runs the CR through 5 reconcile cycles without a cluster — verifies the StatefulSet and Service are created in cycle 1. The CRD deletion arc requires a live cluster; we use [`ork e2e`](#e2e) for that.

---

## Step 3 — Start the runtime

```bash
ork run
```

> No cluster? Add `--dev` to spin up a temporary kind cluster.

Keep this terminal open.

---

## Step 4 — Apply the CR

In a second terminal:

```bash
kubectl apply -f cr.yaml
```

The operator reconciles. Check the health endpoint:

```bash
curl -s localhost:8080/katalog/blockchainapp/health | jq '{state,healthy}'
```

```json
{
  "state": "healthy",
  "healthy": true
}
```

---

## Step 5 — Delete the CRD

```bash
kubectl delete -f crd.yaml
```

Kubernetes cascade-deletes the CRD **and all instances** — the CR is gone too.

In the `ork run` terminal:

```text
{"level":"warn","gvk":"resilience.demo.orkestra.io/v1alpha1, Kind=BlockchainApp","message":"CRD disappeared at runtime — marking missing"}
{"level":"info","gvk":"resilience.demo.orkestra.io/v1alpha1, Kind=BlockchainApp","message":"stopping workers"}
{"level":"info","message":"retry loop: 1 CRD(s) still missing"}
{"level":"info","message":"retry loop: 1 CRD(s) still missing"}
```

Workers stop. Orkestra enters a retry loop waiting for the CRD to return.

---

## Step 6 — Check the health endpoints

```bash
curl -s localhost:8080/katalog/blockchainapp/health | jq
```

```json
{
  "name": "blockchainapp",
  "state": "degraded",
  "healthy": false,
  "consecutiveFails": 7,
  "lastError": "CRD missing at runtime"
}
```

Now check the katalog-level health:

```bash
curl -s localhost:8080/katalog | jq '{healthy,status,degradedReason,OrkReady}'
```

```json
{
  "healthy": false,
  "status": 503,
  "degradedReason": "1 degraded",
  "OrkReady": true
}
```

Three levels, three different answers:

| Level | Field | Value | Meaning |
|-------|-------|-------|---------|
| CRD | `state` | `degraded` | CRD is missing at runtime |
| Katalog | `healthy` | `false` | At least one CRD is degraded |
| Runtime | `OrkReady` | `true` | The Orkestra process itself is fully operational |

`OrkReady: true` is the key signal: the runtime is healthy and running. The degradation is contained to the missing CRD — nothing crashed, nothing needs restarting.

---

## Step 7 — Open the Control Center

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) (username: `orkestra`, password: `orkestra`).

The DEGRADED tile shows `1`. The runtime shows `Operational`. Drill into BlockchainApp → **Runtime Health** to see the consecutive fail counter climbing and `Last Error: CRD missing at runtime`.

---

## Step 8 — Re-apply the CRD

```bash
kubectl apply -f crd.yaml
```

In the `ork run` terminal:

```text
{"level":"info","message":"activating CRD BlockchainApp (resilience.demo.orkestra.io/v1alpha1, Kind=BlockchainApp)"}
{"level":"info","message":"CRD BlockchainApp activated"}
```

The health endpoint flips to `pending` — CRD is back but no CR has been reconciled yet:

```bash
curl -s localhost:8080/katalog/blockchainapp/health | jq '{state,pending}'
```

```json
{
  "state": "pending",
  "pending": true
}
```

The katalog-level health also recovers to healthy at this point — the CRD is no longer degraded, just waiting.

---

## Step 9 — Re-apply the CR

```bash
kubectl apply -f cr.yaml
```

Wait one resync cycle. The operator reconciles:

```text
{"level":"info","deployment":"hello-website","message":"statefulset created"}
{"level":"info","service":"hello-website-svc","message":"service created"}
{"level":"info","message":"reconciled resilience.demo.orkestra.io/v1alpha1, Kind=BlockchainApp"}
```

```bash
curl -s localhost:8080/katalog/blockchainapp/health | jq '{state,healthy,consecutiveFails,lastError}'
```

```json
{
  "state": "healthy",
  "healthy": true,
  "consecutiveFails": 0,
  "lastError": "CRD missing at runtime"
}
```

`lastError` is preserved after recovery — it is an audit trail, not a live state indicator.

---

## E2E

```bash
ork e2e
```

Exercises the full arc — initial healthy state, CRD deletion, degraded detection, re-apply, pending state, CR re-apply, full recovery, resource creation, and cleanup — without manual steps.

All health endpoint assertions use **leader election** to resolve the port-forward target directly to the konductor pod, guaranteeing assertions run against the replica with authoritative reconciler state.

The degradation and re-activation checkpoints use `timeout: 120s`. Orkestra's in-cluster CRD check loop runs every 90 seconds — the timeout must exceed one full tick or the assertion fires before detection has a chance to run. (Locally with `ork run`, the loop runs every 10s so the default 60s timeout is fine.)

For more: https://orkestra.sh/docs/concepts/e2e/leader-led-deployments/

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
