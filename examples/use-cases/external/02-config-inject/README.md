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

Open [http://localhost:8081](http://localhost:8081). Select **webapp-config-inject**, then select the **WebApp** CRD.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

Wait one reconcile (~30s). The operator:
1. Calls `https://httpbin.org/config/my-app` — returns JSON
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

Every 30s the operator fetches the config URL again and updates the ConfigMap if the response changed. Any process watching the mounted file sees the update immediately — no pod restart needed.

To simulate a config change, point the serviceUrl at a different endpoint:

```bash
kubectl patch webapp my-app --type=merge \
  -p '{"spec":{"serviceUrl":"https://httpbin.org"}}'
```

Wait 30s. The ConfigMap is updated with the new response body.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
