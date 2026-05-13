# 01 — Basic Rollback (`rollBackOnError: true`)

Zero-config failure recovery. One field, no extra YAML blocks, no `.previous.spec.*` references.

**What you learn:** `rollBackOnError: true`, how derived rollback works, how to trigger and observe a rollback.

---

## How it works

The Katalog declares `rollBackOnError: true` on the `operatorBox`. That's the only rollback configuration.

When reconcile fails 3 times consecutively, Orkestra:

1. Captures the last known good spec (written to a compressed annotation before each spec change)
2. Marks rollback active on the CR (`orkestra.orkspace.io/rollback-at-generation`)
3. Blocks normal reconciliation
4. Re-applies the `reconcile: true` declarations from `onCreate` — but with `.spec.*` resolving to the **previous** spec values

The Deployment and Service are restored to the previous image and port. No `onRollback:` block. No `.previous.spec.image` template. The same declarations that run on every reconcile also run on rollback — against the previous spec.

Rollback exits when the spec generation changes (you apply a corrected CR).

---

## Step 1 — Apply the CRD

```bash
kubectl apply -f crd.yaml
```

---

## Step 2 — Start the operator

```bash
ork run -f katalog.yaml
```

---

## Step 3 — Apply the good CR

In a second terminal:

```bash
kubectl apply -f cr.yaml
```

Verify the Deployment is running and status is populated:

```bash
kubectl get webapp my-app -o yaml
```

Expected status:

```yaml
status:
  phase: Ready
  image: nginx:1.25
  replicas: 1
```

---

## Step 4 — Introduce a bad spec

Apply a CR with a non-existent image:

```bash
kubectl apply -f cr-bad.yaml
```

Watch the operator terminal. You will see reconcile errors accumulate. After 3 consecutive failures, rollback activates:

```
{"level":"warn","name":"my-app","message":"rollback: threshold reached — marking rollback active"}
{"level":"info","name":"my-app","derived":true,"message":"rollback: previous state re-applied"}
```

The Deployment is restored to `nginx:1.25`. The CR status shows:

```bash
kubectl get webapp my-app
```

```
NAME     PHASE       IMAGE        AGE
my-app   RolledBack  nginx:1.25   2m
```

---

## Step 5 — Exit rollback

Apply the corrected CR (the original good spec):

```bash
kubectl apply -f cr.yaml
```

The generation changes. On the next reconcile cycle, Orkestra detects the new generation, clears the rollback annotation, and resumes normal reconciliation.

```bash
kubectl get webapp my-app
```

```
NAME     PHASE   IMAGE        AGE
my-app   Ready   nginx:1.25   3m
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
