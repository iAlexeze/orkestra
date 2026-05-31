# External 01 — Health Gate

`external:` runs an HTTP call before every resource group. When the upstream health check returns 200 the Deployment is created and kept in sync. When it returns anything else the Deployment is untouched — no broken app, no reconcile error. The operator retries on the next resync.

**What you learn:** the primary use of `external:`, the `continueOnError: true` vs `false` choice, how `external.*` fields gate resources and drive the phase state machine.

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
    mode: dynamic / workers: 2 / resync: 15s
```

---

## Step 2 — Start the operator

`--dev-server` starts a mock HTTP server on `:9999` — no real upstream service needed. It serves `/health` (200) and `/status/503` (503) for the dev CRs:

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

Open [http://localhost:8081](http://localhost:8081). Select **webapp-health-gate**, then select the **WebApp** CRD.

---

## Step 4 — Apply both CRs

```bash
kubectl apply -f cr-dev-healthy.yaml
kubectl apply -f cr-dev-degraded.yaml
```

Wait one reconcile cycle (~15s). Both CRs appear in the Control Center.

```bash
kubectl get webapp -o wide
```

Expected:
```
NAME               PHASE     EXTERNAL   AGE
my-app-healthy     Ready     200        20s
my-app-degraded    Degraded  503        20s
```

Only `my-app-healthy` has a Deployment:

```bash
kubectl get deploy
# NAME               READY   UP-TO-DATE   AVAILABLE
# my-app-healthy     2/2     2            2
```

`my-app-degraded` has no Deployment. The operator never created it.

---

## Step 5 — Inspect status

```bash
kubectl get webapp my-app-healthy -o yaml | grep -A10 "status:"
kubectl get webapp my-app-degraded -o yaml | grep -A10 "status:"
```

Expected for `my-app-degraded`:
```yaml
status:
  phase: Degraded
  lastExternalStatus: "503"
  lastExternalError: "expected status 200, got 503"
```

---

## Step 6 — Fix the degraded app

Patch the CR to point at the healthy dev endpoint:

```bash
kubectl patch webapp my-app-degraded --type=merge \
  -p '{"spec":{"healthCheckUrl":"http://localhost:9999/health"}}'
```

Wait a reconcile cycle. The phase flips to `Ready` and the Deployment appears.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
