# 04 — Idempotent Secret Generation (once:)

`once: true` on a Secret means Orkestra generates it exactly once — when it does not exist. Every subsequent reconcile skips generation entirely. The Deployment mounts the Secret via `secretKeyRef` and gets stable credentials across its entire lifetime.

**What you learn:** the `once:` guard, `randomAlphanumeric` / `randomHex` / `randomBase64` generation functions, and how to consume a generated Secret from a Deployment without writing Go code for the idempotency check.

---

## Step 1 — Apply the CRD

```bash
kubectl apply -f crd.yaml
```

---

## Step 2 — Validate

```bash
ork validate
```

Expected:
```
✓ secure-app
    kind: SecureApp
    group: advanced.orkestra.io / version: v1alpha1 / plural: secureapps
    mode: dynamic / workers: 2 / resync: 30s
```

---

## Step 3 — Start the operator

```bash
ork run
```

---

## Step 4 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **secure-app**, then select the **SecureApp** CRD.

---

## Step 5 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

Wait one reconcile (~30s). The Secret is generated and the Deployment is created.

```bash
kubectl get secureapp my-secure-app -o yaml | grep -A6 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  credentialsSecret: my-secure-app-credentials
```

```bash
kubectl get secret my-secure-app-credentials
kubectl get deploy my-secure-app
```

---

## Step 6 — Verify idempotency

Delete the Secret manually and watch the next reconcile recreate it:

```bash
kubectl delete secret my-secure-app-credentials
```

Wait one reconcile (~30s). The Secret is back — new random values, same name.

Now delete and re-apply the CR itself:

```bash
kubectl delete secureapp my-secure-app
kubectl apply -f cr.yaml
```

New CR, new Secret with a fresh password. The `once:` guard is per Secret existence — not per CR, not per operator restart.

---

## Step 7 — Check the Deployment mounts

```bash
kubectl describe deploy my-secure-app | grep -A10 "Environment:"
```

`PASSWORD`, `API_KEY`, and `JWT_SECRET` are injected via `secretKeyRef`. The Deployment never sees the raw values — only the Secret does.

---

## Cleanup

```bash
kubectl delete secureapp my-secure-app --ignore-not-found
kubectl delete secret my-secure-app-credentials --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
```
