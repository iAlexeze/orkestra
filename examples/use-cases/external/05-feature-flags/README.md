# External 05 — Feature Flag Rollout Gate

On every reconcile the operator reads a live flag from an external service. When `v2Enabled` is `true` the Deployment runs at `spec.replicas` (full capacity). When `false` it falls back to 1 replica. The operator converges automatically on the next resync — no CR edit, no kubectl, no restart.

**What you learn:** an external call that drives a resource attribute (replicas), not just a gate condition. The difference between "block until ready" (`continueOnError: false`) and "degrade gracefully" (`continueOnError: true`). How flipping one flag in an external service ripples through the cluster within one reconcile cycle.

---

## Step 1 — Validate

```bash
ork validate
```

Expected:
```
✓ webapp
    kind: WebApp
    group: demo.orkestra.io / version: v1 / plural: webapps
    mode: dynamic / workers: 2 / resync: 30s
```

---

## Step 2 — Start the operator

`--dev-server` starts a mock server on `:9999`. `GET /flags/my-app/v2Enabled` returns `true` by default. The toggle endpoint lets you flip it live without restarting anything:

```bash
ork run --dev-server
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **webapp-feature-flags**, then select the **WebApp** CRD.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

`spec.replicas: 5` — this is the scaled-up count when `v2Enabled` is `true`.

Wait one reconcile (~30s). The operator reads the flag, gets `true`, creates the Deployment with 5 replicas.

```bash
kubectl get webapp my-app -o yaml | grep -A8 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  v2Enabled: "true"
  activeReplicas: "5"
```

```bash
kubectl get deploy my-app
# NAME      READY   UP-TO-DATE   AVAILABLE
# my-app    5/5     5            5
```

---

## Step 5 — Flip the flag off

In a separate terminal, toggle `v2Enabled` to `false`:

```bash
curl -X POST http://localhost:9999/flags/my-app/v2Enabled/toggle
# → false
```

Wait one reconcile (~30s). The operator reads the flag again, gets `false`, updates the Deployment to 1 replica.

```bash
kubectl get deploy my-app -w
# NAME      READY   UP-TO-DATE   AVAILABLE
# my-app    5/5     5            5
# my-app    1/1     1            1
```

```bash
kubectl get webapp my-app -o yaml | grep -A8 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  v2Enabled: "false"
  activeReplicas: "1"
```

No CR edit. No redeployment. The flag service changed — the cluster followed.

---

## Step 6 — Check the metrics

```bash
curl -s localhost:8080/metrics | grep external_call
```

The call counter increments on every reconcile _(confirm the number of reconciles in the Control Center)_ — unlike the image signing example, this call is intentionally made every cycle because the flag can change at any time. The `continueOnError: true` setting means a flag service outage degrades to baseline (1 replica) rather than halting reconciliation.

---

## Step 7 — Flip the flag back on

```bash
curl -X POST http://localhost:9999/flags/my-app/v2Enabled/toggle
# → true
```

Wait one reconcile. Deployment scales back to 5.

---

## How the replica switching works

Two deployment entries target the same name with different replicas. Exactly one fires per reconcile:

```yaml
onCreate:
  # Full capacity — fires when flag is true
  deployments:
    - name: "{{ .metadata.name }}"
      replicas: "{{ .spec.replicas }}"    # 5
      reconcile: true
      when:
        - field: external.flags.body
          equals: "true"

    # Baseline — fires when flag is false or flag service is down
    - name: "{{ .metadata.name }}"
      replicas: "1"
      reconcile: true
      when:
        - field: external.flags.body
          notEquals: "true"
```

`reconcile: true` is required here because these resources are under `onCreate`. Without it, `onCreate` would create the Deployment once and never update it when the flag changes. `reconcile: true` tells Orkestra to apply the desired state on every reconcile — drift correction for `onCreate` resources. The second entry also catches flag service outages — if the call fails, `flags.body` is empty, `notEquals: "true"` passes, and the operator falls back to baseline safely.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## E2E

Run the full lifecycle — deploys the mock dev server, starts the operator, applies the CR, asserts the Deployment runs at full replicas (`v2Enabled=true` default), then tears down:

```bash
ork e2e --dev-server
```

CRs use the in-cluster address defined in [cr-e2e.yaml](./cr-e2e.yaml). This runs everything in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Deployment at full replicas when v2Enabled is true
    after: cr-applied
    resources:
      - kind: Deployment
        name: my-app
        ready: true
  - name: Status v2Enabled is true and activeReplicas is 5
    after: cr-applied
    commands:
      - run: kubectl get webapp my-app -o jsonpath='{.status.v2Enabled}'
        outputContains: "true"
      - run: kubectl get deploy my-app -o jsonpath='{.spec.replicas}'
        outputContains: "5"
```
