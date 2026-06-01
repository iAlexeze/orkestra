# External 02 — Config Inject

On every reconcile the operator fetches a JSON config blob from an external service and embeds it into a ConfigMap. The Deployment mounts the ConfigMap — the app always has current config without a restart or a redeploy. If the config service is down the ConfigMap is left unchanged and the Deployment keeps running.

**What you learn:** `continueOnError: true` for optional calls, conditional ConfigMap writes via `when: + external.*`, the live-config-without-redeploy pattern.

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

`--dev-server` starts a mock HTTP server on `:9999` — no real config service needed. It responds to `GET /config/:name` with a static JSON config blob:

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

Open [http://localhost:8081](http://localhost:8081). Select **webapp-config-inject**, then select the **WebApp** CRD.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

Wait one reconcile (~30s). The operator:
1. Calls `http://localhost:9999/config/my-app` (dev) — returns a static JSON config blob
2. Creates a ConfigMap with the response body
3. Creates the Deployment mounted to that ConfigMap

```bash
kubectl get webapp my-app -o yaml | grep -A10 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  lastExternalStatus: "200"
  configFresh: "true"
```

Inspect the ConfigMap:

```bash
kubectl get configmap my-app-config -o jsonpath='{.data.app\.json}' | jq .
```

---

## Step 5 — Watch live config update

Every 30s the operator fetches the config URL again and updates the ConfigMap if the response changes. Any process watching the mounted file sees the update immediately — no pod restart needed.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## E2E

Run the full lifecycle — deploys the mock dev server, starts the operator, applies the CR, asserts the ConfigMap contains the injected config and status is Ready, then tears down:

```bash
ork e2e --dev-server
```

CRs use the in-cluster address defined in [cr-e2e.yaml](./cr-e2e.yaml). This runs everything in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: ConfigMap created and contains injected config
    after: cr-applied
    commands:
      - run: "kubectl get configmap my-app-config -o jsonpath='{.data.app\\.json}'"
        outputContains: "production"
```
