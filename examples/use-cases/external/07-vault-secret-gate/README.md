# External 07 — Vault Secret Gate

On every reconcile the operator reads a secret from Vault KV v2 before creating the Deployment. A missing secret (404) or expired lease (403) blocks the Deployment and surfaces the reason in `status.phase`. Rotation recovery is automatic — the next reconcile retries without any manual intervention.

**What you learn:** gating a Deployment on an external secret store, distinguishing missing vs expired secrets in status, and how the reconcile loop handles rotation recovery without custom polling.

---

## Step 1 — Validate

```bash
ork validate
```

---

## Step 2 — Start the operator

```bash
ork run --dev-server
```

Dev server route: `GET /vault/v1/secret/data/:path`
- Path contains `expired` → 403 lease expired
- Path contains `missing` → 404 not found
- Anything else → 200 with secret data

No `VAULT_TOKEN` needed — the mock server ignores the header.

---

## Step 3 — Open the Control Center

```bash
ork control   # http://localhost:8081 → orkestra/orkestra
```

---

## Step 4 — Apply the expired CR

```bash
kubectl apply -f cr-expired.yaml
```

The CR name is `my-app-expired` — the operator constructs the path `apps/my-app-expired`. The dev server sees `expired` in the path and returns 403.

Wait one reconcile (~30s):

```bash
kubectl get webapp my-app-expired -o yaml | grep -A6 "status:"
```

Expected:
```yaml
status:
  phase: SecretExpired
  secretStatus: "403"
```

No Deployment created.

---

## Step 5 — Apply the healthy CR

```bash
kubectl apply -f cr.yaml
```

Path is `apps/my-app` — no special segment, secret found and valid.

```bash
kubectl get webapp my-app -o yaml | grep -A6 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  secretStatus: "200"
```

---

## Step 6 — Simulate rotation recovery

Rename the expired CR (or delete and re-create with a clean name). The next reconcile hits a 200 — the Deployment appears without any operator restart.

This is the rotation recovery pattern: fix the secret path (or rotate in Vault), wait one reconcile, the Deployment is created. No finalizers, no annotations, no custom controller logic.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
