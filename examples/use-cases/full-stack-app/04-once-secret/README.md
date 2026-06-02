# 04 — Idempotent Secret Generation (once:)

`once: true` on a Secret means Orkestra generates it exactly once — when it does not exist. Every subsequent reconcile skips generation entirely. The Deployment mounts the Secret via `secretKeyRef` and gets stable credentials across its entire lifetime.

**What you learn:** the `once:` guard, `randomAlphanumeric` / `randomHex` / `randomBase64` generation functions, and how to consume a generated Secret from a Deployment without writing Go code for the idempotency check.

---

## Step 1 — Validate

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

## Step 2 — Start the operator

`ork run` reads the `crdFile` declared in `katalog.yaml`, applies the CRD to the cluster, and starts the runtime:

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

Open [http://localhost:8081](http://localhost:8081). Select **secure-app**, then select the **SecureApp** CRD.

---

## Step 4 — Apply the CR

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

## Step 5 — Verify idempotency

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

## Step 6 — Check the Deployment mounts

```bash
kubectl describe deploy my-secure-app | grep -A10 "Environment:"
```

`PASSWORD`, `API_KEY`, and `JWT_SECRET` are injected via `secretKeyRef`. The Deployment never sees the raw values — only the Secret does.

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies the CRD, starts the operator, applies the CR, asserts the Secret is created exactly once and not regenerated on re-apply, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Secret created on first reconcile
    after: cr-applied
    timeout: 60s
    resources:
      - kind: Secret
        name: my-secure-app-credentials
        namespace: default

  - name: Secret is not recreated on re-apply (once: semantics)
    after: cr-applied
    timeout: 30s
    commands:
      - run: >
          ORIG=$(kubectl get secret my-secure-app-credentials -o jsonpath='{.metadata.resourceVersion}') &&
          kubectl apply -f cr.yaml &&
          sleep 5 &&
          NEW=$(kubectl get secret my-secure-app-credentials -o jsonpath='{.metadata.resourceVersion}') &&
          [ "$ORIG" = "$NEW" ] && echo "unchanged"
        outputContains: unchanged
```

---

## Cleanup

```bash
kubectl delete secureapp my-secure-app --ignore-not-found
kubectl delete secret my-secure-app-credentials --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
```
