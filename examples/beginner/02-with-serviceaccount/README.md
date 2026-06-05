# 02 — Website with ServiceAccount

Three resources, drift correction, and a wired identity. This example adds a
`ServiceAccount` to the operator and passes it directly into the Deployment —
so the pod runs with a named identity from the moment it starts.

**What you learn:** ServiceAccount wiring, notes — reading live cluster state into status.

**Builds on:** [01 — Hello Website](../01-hello-website/README.md)

---

## What is new

**ServiceAccount created before the Deployment.** Orkestra processes `serviceAccounts`
before `deployments` in `onCreate`. The `serviceAccountName` field on the
Deployment template references the ServiceAccount by name, so the pod identity
is in place when the pod first schedules.

**Notes — live cluster state in status.** The `allReplicasReady` status field uses
a note: `{{ allReplicasReady .children.deployment }}`. Unlike the plain string fields
in example 01, this reads from the live Deployment object after every reconcile and
writes the result back to the CR's status. With 2 replicas declared in `cr.yaml`,
you can watch it flip from `"false"` to `"true"` as both pods come up — and see
the same in the Control Center's resources section.

---

## Step 1 — Validate the Katalog

```bash
ork validate
```

Expected output:

```
✓ website
    kind: Website
    group: demo.orkestra.io / version: v1alpha1 / plural: websites
    mode: dynamic / workers: 3 / resync: 15s
```

---

## Step 2 — Simulate (optional, no cluster needed)

Before starting against a real cluster, run the reconciler in memory to verify your templates produce the right resources:

```bash
ork simulate --cr cr.yaml
```

```text
Simulating website/my-site

  Cycle 1:
    + serviceaccounts/my-site-sa
    + deployments/my-site
    + services/my-site-svc
    ~ status/my-site
  Cycle 2:
    ~ status/my-site
  (cycles 3–10: identical)

  ✓ Steady state at cycle 3 in 215ms
```

**What this means:**
- `+` in Cycle 1 — these resources would be created: the ServiceAccount first, then the Deployment that references it, then the Service. The order matches `onCreate` processing order in the Katalog.
- `~ status/my-site` — the CR's status fields would be written back on every cycle. Notes like `allReplicasReady` re-evaluate each reconcile.
- Cycle 2 — no new resources. Status still updates because notes re-run unconditionally.
- **Steady state at cycle 3** — from cycle 3 onward, reconciling this CR produces no changes. A healthy operator converges and stops acting. If your operator never reaches steady state, simulate tells you which resources keep changing and why.
- 215ms — wall time for all 10 simulated cycles in memory. No cluster round-trips.

The example ships with [simulate.yaml](./simulate.yaml) which records these expectations so they are verified on every run:

```bash
ork simulate
```

```text
  ✓ steady
  ✓ noErrors
  ✓ create serviceaccounts/my-site-sa (cycle 1)
  ✓ create deployments/my-site (cycle 1)
  ✓ create services/my-site-svc (cycle 1)

  PASS
```

`ork simulate` with a `simulate.yaml` present is the fast inner check — no cluster, under a second. [`ork e2e`](#e2e) is the full integration gate.

---

## Step 3 — Start the runtime

```bash
ork run       # add --dev if you don't have a cluster; Orkestra will create a kind cluster
```

Orkestra applies the CRD, waits for it to be established, applies `cr.yaml`,
and starts the operator. Watch the terminal for the informer sync and reconcile event.

---

## Step 4 — Open the Control Center

In a separate terminal:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081) to see the live operator —
CRD health, worker state, reconcile metrics, and the `Website` CR.

---

## Step 5 — Verify resources

```bash
kubectl get websites
kubectl get deployments
kubectl get services
kubectl get serviceaccounts
```

Expected:

```
NAME      READY   UP-TO-DATE   AVAILABLE
my-site   1/1     1            1

NAME          TYPE        CLUSTER-IP     PORT(S)
my-site-svc   ClusterIP   10.96.x.x      80/TCP

NAME         SECRETS   AGE
my-site-sa   0         <age>
```

---

## Step 6 — Verify status

```bash
kubectl get website my-site -o yaml | grep -A20 "status:"
```

Expected once both pods are up:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
  phase: Running
  endpoint: my-site.default.svc.cluster.local
  allReplicasReady: "true"
```

While the pods are still starting, `allReplicasReady` will be `"false"`. Watch it
change:

```bash
kubectl get website my-site -o jsonpath='{.status.allReplicasReady}' && echo
```

---

## Step 7 — Test drift correction

Update [cr.yaml](cr.yaml) to change the image to `nginx:1.26` and reapply:

```bash
kubectl apply -f cr.yaml
kubectl get deployment my-site -o jsonpath='{.spec.template.spec.containers[0].image}' && echo
# nginx:1.26
```

---

## Step 8 — Scale and watch the note update

```bash
kubectl patch website my-site --type=merge -p '{"spec":{"replicas":4}}'
```

Watch the Deployment scale and `allReplicasReady` cycle through `"false"` back to `"true"`:

```bash
kubectl get deployment my-site
# READY: 4/4

kubectl get website my-site -o jsonpath='{.status.allReplicasReady}' && echo
# "true"
```

You will also see the value flip in real time in the Control Center's resources section.

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies the CRD,
starts the operator, applies the CR, asserts every expectation, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Resources created
    after: cr-applied
    timeout: 60s
    resources:
      - kind: Deployment
        namespace: default
        ready: true
      - kind: Service
        namespace: default
      - kind: ServiceAccount
        name: my-site-sa
        namespace: default

  - name: Cleanup verified
    after: cr-deleted
    timeout: 30s
    resources:
      - kind: Deployment
        name: my-site
        namespace: default
        count: 0
      - kind: Service
        name: my-site-svc
        namespace: default
        count: 0
      - kind: ServiceAccount
        name: my-site-sa
        namespace: default
        count: 0
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

Or manually:

```bash
kubectl delete -f cr.yaml
kubectl delete -f crd.yaml
# Stop ork run with Ctrl+C
```
