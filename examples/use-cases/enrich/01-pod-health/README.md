# Enrich 01 — Pod Health

`enrich: [pods]` embeds the live pod list into `.children.deployment._pods` on every reconcile. Status fields surface pod count, readiness, and crash detection — without reading the pod list anywhere in your Katalog explicitly.

**Cost:** one extra pod-list API call per reconcile cycle, always. This is the right pattern when you need pod health surfaced at all times — running count, crash detection, readiness. If you only need this data during degraded state, see [02-warning-events](../02-warning-events/README.md) for the conditional gate pattern that reduces this cost to zero in steady state.

**What you learn:** what `enrich` is for, how pod notes require it, the one-line declaration that unlocks `podCount`, `readyPodCount`, `hasCrashingPod`, and more.

---

## Step 1 — Validate

```bash
ork validate
```

Expected:
```
✓ microservice
    kind: Microservice
    group: demo.orkestra.io / version: v1 / plural: microservices
    mode: dynamic / workers: 2 / resync: 15s
```

---

## Step 2 — Start the operator

```bash
ork run
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **microservice-operator-pods**, then select the **Microservice** CRD.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

Wait for the pods to start (~15s). Watch the Control Center — `api-server` appears. Click it, then click **top-right** to see child resources. Open the `api-server` Deployment.

```bash
kubectl get microservice api-server -o yaml | grep -A15 "status:"
```

Expected once both pods are running:
```yaml
status:
  phase: Ready
  podCount: "2"
  readyPodCount: "2"
  podNames: api-server-xxxx-yyyy, api-server-xxxx-zzzz
  hasCrashingPod: "false"
  podMaxRestarts: "0"
```

---

## Step 5 — Scale and observe

```bash
kubectl patch microservice api-server --type=merge -p '{"spec":{"replicas":4}}'
```

Watch `podCount` climb as new pods start. `readyPodCount` lags behind until all four pods are ready.

```bash
kubectl get microservice api-server -o yaml | grep "podCount\|readyPodCount"
```

---

## Step 6 — Simulate a crash

```bash
# Force a bad image to trigger CrashLoopBackOff
kubectl patch microservice api-server --type=merge -p '{"spec":{"image":"nginx:does-not-exist"}}'
```

Wait a reconcile cycle (15s). `phase` flips to `Degraded`. `hasCrashingPod` becomes `"true"`. `crashReason` appears:

```bash
kubectl get microservice api-server -o yaml | grep "phase\|crashReason\|hasCrashingPod"
```

Fix it:
```bash
kubectl patch microservice api-server --type=merge -p '{"spec":{"image":"nginx:1.25"}}'
```

`phase` returns to `Ready`. `hasCrashingPod` goes back to `"false"`. `crashReason` is cleared to `""` — because the field declares `clearOnFalse: true`, which tells Orkestra to explicitly write an empty value when the condition is no longer true, rather than leaving the old value in place.

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies the CRD, starts the operator, applies the CR, asserts pod count and readiness appear in status, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Status has podCount
    after: cr-applied
    timeout: 90s
    commands:
      - run: kubectl get microservice api-server -o jsonpath='{.status.podCount}'
        outputContains: "2"

  - name: Status shows no crashing pods
    after: cr-applied
    timeout: 60s
    commands:
      - run: kubectl get microservice api-server -o jsonpath='{.status.hasCrashingPod}'
        outputContains: "false"
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
