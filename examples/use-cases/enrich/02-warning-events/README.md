# Enrich 02 — Warning Events

`enrich: [events]` with a conditional gate. In steady state — all replicas ready — Orkestra skips the event-list call entirely. When the deployment is degraded, events are fetched and warning details appear in status. Fix the image and the events disappear.

**What you learn:** conditional enrichment, matching the status field gate to the enrichment gate, the cost drop in steady state.

---

## Step 1 — Validate

```bash
ork validate
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

Open [http://localhost:8081](http://localhost:8081). Select **microservice-operator-events**, then select the **Microservice** CRD.

---

## Step 4 — Apply the healthy CR

```bash
kubectl apply -f cr-healthy.yaml
```

This CR has a valid image. Wait for the pod to start (~15s).

```bash
kubectl get microservice healthy-app -o yaml | grep -A10 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  podCount: "1"
  readyPodCount: "1"
  hasCrashingPod: "false"
  # hasWarnings, firstWarning, warningCount — absent
  # Events were NOT fetched this cycle
```

No `hasWarnings`. No `firstWarning`. No event-list call was issued — the gate (`replicasReady == false`) was false, so the enrichment was skipped.

---

## Step 5 — Apply the broken CR

```bash
kubectl apply -f cr-broken.yaml
```

This CR has `image: nginx:does-not-exist`. Kubernetes will fail to pull it, generating `BackOff` warning events.

Wait ~30s for the pod to fail and for events to accumulate.

```bash
kubectl get microservice broken-app -o yaml | grep -A15 "status:"
```

Expected:
```yaml
status:
  phase: Degraded
  podCount: "1"
  readyPodCount: "0"
  hasCrashingPod: "true"
  hasWarnings: "true"
  warningCount: "3"
  firstWarning: "Back-off pulling image \"nginx:does-not-exist\""
```

In the Control Center, click `broken-app`. The phase shows `Degraded`. The `firstWarning` field tells you exactly what Kubernetes reported.

---

## Step 6 — Fix the image and watch events disappear

```bash
kubectl patch microservice broken-app --type=merge -p '{"spec":{"image":"nginx:1.25"}}'
```

Wait a reconcile cycle. `phase` becomes `Ready`. `hasWarnings`, `warningCount`, and `firstWarning` are gone — the enrichment gate is now false, events are no longer fetched, and the status fields are no longer written.

```bash
kubectl get microservice broken-app -o yaml | grep "Warning\|warning\|phase"
# phase: Ready
# <nothing else>
```

---

## What happened to the API calls

| State | Pod-list calls | Event-list calls |
|---|---|---|
| Ready (healthy-app) | 1 per reconcile | **0** — gate skipped |
| Degraded (broken-app) | 1 per reconcile | **1** — gate passed |
| Fixed (back to Ready) | 1 per reconcile | **0** — gate skipped again |

The event-list call appears only when it is needed.

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies the CRD, starts the operator, applies the healthy CR and the broken CR, asserts the conditional gate holds in steady state and fires when degraded, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Healthy CR has no event details (gate held)
    after: cr-applied
    timeout: 60s
    commands:
      - run: kubectl get microservice healthy-app -o jsonpath='{.status.warningEvents}'
        outputContains: ""

  - name: Broken CR has warning events in status
    after: cr-applied
    timeout: 120s
    commands:
      - run: kubectl get microservice broken-app -o jsonpath='{.status.warningEvents}'
        outputContains: ImagePullBackOff
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
