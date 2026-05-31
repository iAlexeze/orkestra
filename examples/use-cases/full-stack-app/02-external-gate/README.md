# 02 — External Health Gate (external:)

The Deployment is only created when an upstream health check returns 200. A second call fetches feature flags on every reconcile — it is non-blocking, so a flags service outage does not stop the operator. Both results land in `external.*` and are available to resource templates and status fields.

**What you learn:** `external:` as a gate on resource creation, `continueOnError: false` vs `true`, how `external.*` fields drive `when:` conditions and status.

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
✓ gated-app
    kind: GatedApp
    group: advanced.orkestra.io / version: v1alpha1 / plural: gatedapps
    mode: dynamic / workers: 2 / resync: 30s
```

---

## Step 3 — Start the operator

`--dev-server` starts a mock HTTP server on `:9999`. It serves `/health` (200) and `/flags/:name` for both calls — no real service needed:

```bash
ork run --dev-server
```

---

## Step 4 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **gated-app**, then select the **GatedApp** CRD.

---

## Step 5 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

Wait one reconcile (~30s). The health check fires and returns 200 — the Deployment appears.

```bash
kubectl get gatedapp my-gated-app -o yaml | grep -A8 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  lastHealthCheck: "200"
```

```bash
kubectl get deploy my-gated-app
# NAME            READY   UP-TO-DATE   AVAILABLE
# my-gated-app    1/1     1            1
```

---

## Step 6 — See the gate in action

Patch the CR to point at a URL that returns a non-200:

```bash
kubectl patch gatedapp my-gated-app --type=merge \
  -p '{"spec":{"serviceUrl":"http://localhost:9999/status/503"}}'
```

Wait one reconcile. Phase flips to `Degraded`. The Deployment is not deleted — but on the next reconcile it will not be updated. The gate prevents changes until health passes again.

Restore:

```bash
kubectl patch gatedapp my-gated-app --type=merge \
  -p '{"spec":{"serviceUrl":"http://localhost:9999"}}'
```

---

## Cleanup

```bash
kubectl delete gatedapp my-gated-app --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
```
