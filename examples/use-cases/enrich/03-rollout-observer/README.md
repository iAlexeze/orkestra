# Enrich 03 — Rollout Observer

`enrich: [replicasets]` with `anyOf:` — replicaset data is fetched when the deployment is not fully ready (rolling update in progress) OR when `spec.debug` is `"true"`. In steady state both conditions are false: the replicaset-list call never fires. During a rollout you can watch the old and new ReplicaSet counts change in real time.

**Cost:** zero API calls for the replicaset enrichment in steady state — `anyOf:` acts as a circuit breaker. The pod-list from `enrich: [pods]` still runs unconditionally. Setting `spec.debug: "true"` on a single CR enables the expensive enrichment for that CR only; other CRs in the same operator are unaffected.

**What you learn:** `anyOf:` in enrichment conditions, combining always-on and conditional targets, debug-mode enrichment without affecting other CRs.

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

Open [http://localhost:8081](http://localhost:8081). Select **microservice-operator-rollout**, then select the **Microservice** CRD.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

Wait for all 3 pods to start (~20s).

```bash
kubectl get microservice web-frontend -o yaml | grep -A10 "status:"
```

Expected (steady state):
```yaml
status:
  phase: Ready
  podCount: "3"
  readyPodCount: "3"
  hasCrashingPod: "false"
  # replicaSetCount — absent (gate did not fire)
  # oldReplicaSets  — absent
```

In steady state: **1 API call** (pod-list). ReplicaSet enrichment skipped.

---

## Step 5 — Trigger a rollout

Update the image to trigger a rolling update:

```bash
kubectl patch microservice web-frontend --type=merge -p '{"spec":{"image":"nginx:1.26"}}'
```

During the rollout, the old ReplicaSet is scaling down while the new one scales up. Watch the status update in real time:

```bash
watch kubectl get microservice web-frontend -o yaml
```

Expected mid-rollout:
```yaml
status:
  phase: Rollout
  podCount: "4"          # old + new pods overlapping
  readyPodCount: "2"
  replicaSetCount: "2"   # old and new ReplicaSets
  oldReplicaSets: "..."  # old ReplicaSet names
```

Once the rollout completes:
```yaml
status:
  phase: Ready
  podCount: "3"
  readyPodCount: "3"
  # replicaSetCount and oldReplicaSets absent again
```

During the rollout: **3 API calls** (pod-list + replicaset-list × the anyOf gate). After: **1 API call**.

---

## Step 6 — Debug mode enrichment

Enable replicaset enrichment without triggering a rollout:

```bash
kubectl patch microservice web-frontend --type=merge -p '{"spec":{"debug":"true"}}'
```

```bash
kubectl get microservice web-frontend -o yaml | grep "debug\|replicaSet"
```

Expected:
```yaml
status:
  debugReplicaSetCount: "1"   # written when debug is on
```

The `debugReplicaSetCount` field appears. Other CRs without `debug: "true"` continue with 1 API call per reconcile.

Disable debug:
```bash
kubectl patch microservice web-frontend --type=merge -p '{"spec":{"debug":"false"}}'
```

`debugReplicaSetCount` disappears. Back to 1 API call.

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies the CRD, starts the operator, applies the CR, asserts no replicaset data in steady state and that debug mode surfaces it, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: No replicaset data in steady state (anyOf gate held)
    after: cr-applied
    timeout: 60s
    commands:
      - run: kubectl get microservice web-frontend -o jsonpath='{.status.replicaSetCount}'
        outputContains: ""

  - name: Debug mode surfaces replicaset data
    after: cr-applied
    timeout: 60s
    commands:
      - run: kubectl patch microservice web-frontend --type=merge -p '{"spec":{"debug":"true"}}'
        exitCode: 0
      - run: kubectl get microservice web-frontend -o jsonpath='{.status.replicaSetCount}'
        outputContains: "1"
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
