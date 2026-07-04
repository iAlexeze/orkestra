# Admission Protection — Runtime Validation as a Resilience Layer

A bad CR that reaches a reconciler causes repeated errors. Enough of them and the operator degrades. Orkestra's validation rules run on **every reconcile cycle** — not just at admission time — so invalid input is caught at the runtime layer regardless of whether the gateway is deployed.

This example shows the full arc: bad CR → degraded → patch to valid → healthy.

**What you learn:** How runtime validation keeps operators from acting on bad input, how the failure threshold controls when the operator signals a problem, and how Orkestra recovers automatically when the input is corrected — no restart needed.

---

## What's in this example

| File | Purpose |
|------|---------|
| `katalog.yaml` | BlockchainApp operator with validation rules and `failureThreshold: 3` |
| `crd.yaml` | BlockchainApp CRD |
| `cr-bad.yaml` | CR with a public image — fails the `myorg/` prefix check |
| `cr-good.yaml` | CR with a valid image |

**Validation rules:**
- `spec.image` must start with `myorg/` — deny otherwise
- `spec.replicas` must be > 0 — deny otherwise

**`failureThreshold: 3`** — the operator degrades after 3 consecutive validation failures. Without this setting the Orkestra default is 5.

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

Runs the valid CR through 5 reconcile cycles without a cluster — verifies the StatefulSet and Service are created in cycle 1.

To also see what the reconciler sees with the bad CR:

```bash
ork simulate -f simulate-bad.yaml
```

Errors are expected here — validation rejects the image on every cycle and no resources are created.

[`ork e2e`](#e2e) is used to test the full degradation arc (consecutive failures, health endpoint assertions) requires a live cluster.

---

## Step 3 — Start the runtime

```bash
ork run
```

> No cluster? Add `--dev` to spin up a temporary kind cluster.

Keep this terminal open.

---

## Step 4 — Apply the bad CR

In a second terminal:

```bash
kubectl apply -f cr-bad.yaml
```

---

## Step 5 — Watch the errors stream

In the `ork run` terminal:

```text
{"level":"error","gvk":"resilience.demo.orkestra.io/v1alpha1, Kind=BlockchainApp","message":"validation failed: images must come from the internal registry (myorg/)"}
{"level":"error","gvk":"resilience.demo.orkestra.io/v1alpha1, Kind=BlockchainApp","message":"validation failed: images must come from the internal registry (myorg/)"}
{"level":"error","gvk":"resilience.demo.orkestra.io/v1alpha1, Kind=BlockchainApp","message":"validation failed: images must come from the internal registry (myorg/)"}
```

After the third failure (`failureThreshold: 3`), the operator enters Degraded.

---

## Step 6 — Check the health endpoints

Check the CRD-level health:

```bash
curl -s localhost:8080/katalog/blockchainapp/health | jq
```

```json
{
  "name": "blockchainapp",
  "state": "degraded",
  "healthy": false,
  "consecutiveFails": 3,
  "lastError": "validation denied: images must come from the internal registry (myorg/)"
}
```

Now check the katalog-level health:

```bash
curl -s localhost:8080/katalog | jq '{healthy, status, degradedReason, OrkReady}'
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
| CRD | `state` | `degraded` | This operator has exceeded its failure threshold |
| Katalog | `healthy` | `false` | At least one CRD in this katalog is degraded |
| Runtime | `OrkReady` | `true` | The Orkestra process itself is fully operational |

`OrkReady: true` is the key signal: the runtime is healthy and running. The degradation is contained to one operator within the katalog — nothing crashed, nothing needs restarting.

No StatefulSet was created — the validation blocked every reconcile before the operatorBox ran:

```bash
kubectl get statefulset -n default
# No resources found.
```

---

## Step 7 — Open the Control Center

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) (username: `orkestra`, password: `orkestra`).

The same three levels are visible at a glance:

- **DEGRADED tile** — shows `1`, with a red progress bar. The CRD row below shows `● Degraded`.
- **Katalog Health** — `Degraded` in red.
- **Runtime** — `Operational` in green.

The runtime being `Operational` while the katalog is `Degraded` is the core of the story: Orkestra is running fine. One operator inside it has a problem, and it is contained.

Drill into the BlockchainApp CRD → **Runtime Health** panel to see:
- Consecutive failures counter climbing
- `Last Error` — the exact validation message
- Uptime — the runtime has been up the whole time

---

## Step 8 — Patch the CR to a valid image

```bash
kubectl patch blockchainapp my-chain --type merge \
  -p '{"spec":{"image":"myorg/blockchain-node:v1.0.0"}}'
```

---

## Step 9 — Watch the operator recover

The runtime reconciles on the next cycle and the validation passes:

```text
{"level":"info","message":"reconciled resilience.demo.orkestra.io/v1alpha1, Kind=BlockchainApp"}
```

```bash
curl -s localhost:8080/katalog/blockchainapp/health | jq
```

```json
{
  "state": "healthy",
  "healthy": true,
  "consecutiveFails": 0,
  "lastError": "validation failed: images must come from the internal registry (myorg/)"
}
```

`lastError` is preserved after recovery — it is an audit trail, not a live state indicator.

The StatefulSet and Service now exist:

```bash
kubectl get statefulset,service -n default
```

---

## E2E

```bash
ork e2e
```

Exercises the full arc — degraded state, health endpoint assertions, patch, recovery, and resource creation — without manual steps.

The health endpoint assertions use **leader election** to resolve the port-forward target. Instead of forwarding to the runtime Service (which routes to a random pod), the harness reads the `orkestra-konductor` Lease to find the elected leader and forwards directly to that pod. This guarantees assertions run against the replica with authoritative reconciler state.

The expect checkpoints are split across three files in `e2e/` and composed into [`e2e.yaml`](e2e.yaml) via `include:` — `degraded.yaml`, `recovery.yaml`, and `cleanup.yaml`. Each can be read, improved, and scaled independently without touching the others.

For more: https://orkestra.sh/docs/concepts/e2e/leader-led-deployments/

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
