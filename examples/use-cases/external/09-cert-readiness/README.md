# External 09 — Certificate Readiness Gate

On every reconcile the operator checks whether a TLS certificate has been issued for this CR. The Deployment is only created or kept running while the cert status is `issued`. Toggle the cert to `pending` and the Deployment disappears. Toggle it back and the next reconcile restores it — no CR edits, no kubectl restarts.

**What you learn:** gating a Deployment on an external readiness state that can change at any time, distinguishing pending from unavailable in status, and observing automatic recovery when the gate reopens.

---

## Step 1 — Validate

```bash
ork validate
```

---

## Step 2 — Start the runtime

```bash
ork run --dev-server
```

Dev server routes:
- `GET /certs/:name/status` — `200 issued` by default, `202 pending` after toggle
- `POST /certs/:name/toggle` — flip between issued and pending (stateful, in-memory)

---

## Step 3 — Open the Control Center

```bash
ork control   # http://localhost:8081 → orkestra/orkestra
```

---

## Step 4 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

Cert starts as `issued`. Wait one reconcile (~15s):

```bash
kubectl get webapp my-app -o yaml | grep -A6 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  certStatus: '{"name":"my-app","status":"issued","issuer":"dev-ca","expires_at":"2099-12-31T00:00:00Z"}'
```

```bash
kubectl get deploy
# NAME      READY   UP-TO-DATE   AVAILABLE
# my-app    2/2     2            2
```

---

## Step 5 — Toggle to pending

```bash
curl -X POST http://localhost:9999/certs/my-app/toggle
# → pending
```

Wait one reconcile (~15s). The cert check returns 202 — the Deployment `when:` condition fails. Orkestra removes the Deployment. Phase flips to `CertPending`.

```bash
kubectl get webapp my-app -o yaml | grep -A4 "status:"
```

Expected:
```yaml
status:
  phase: CertPending
```

---

## Step 6 — Toggle back to issued

```bash
curl -X POST http://localhost:9999/certs/my-app/toggle
# → issued
```

Wait one reconcile. Phase returns to `Ready`. Deployment is reconciled again.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## E2E

Run the full lifecycle — deploys the mock dev server, starts the operator, applies the CR, asserts the Deployment is created and `status.certStatus` contains `issued`, then tears down:

```bash
ork e2e --dev-server
```

CRs use the in-cluster address defined in [cr-e2e.yaml](./cr-e2e.yaml). This runs everything in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Deployment created when certificate is issued
    after: cr-applied
    resources:
      - kind: Deployment
        name: my-app
        ready: true
  - name: Status phase is Ready and certStatus contains issued
    after: cr-applied
    commands:
      - run: kubectl get webapp my-app -o jsonpath='{.status.phase}'
        outputContains: Ready
      - run: kubectl get webapp my-app -o jsonpath='{.status.certStatus}'
        outputContains: issued
```
