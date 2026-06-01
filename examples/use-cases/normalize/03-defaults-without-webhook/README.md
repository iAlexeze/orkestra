# Normalize 03 — Defaults Without a Webhook

`cr-minimal.yaml` declares only `image`. Everything else is omitted. `cr-full.yaml` declares every field explicitly. After reconcile, both produce a Deployment with the same resource requests, replica count, and concurrency policy.

This is mutation — `default` inside `normalize` — without deploying the Orkestra Gateway. No webhook. No admission configuration. Defaults are applied at reconcile time, in memory.

**What you learn:** `default` in `normalize.spec`, the difference between this and `mutation:` rules, when each is appropriate. Also: why nested optional fields like `spec.resources.requests.cpu` require `get` for safe navigation — direct dot-path access panics when an intermediate key is absent.

---

## Step 1 — Validate

```bash
ork validate
```

Expected:
```
✓ workload
    kind: Workload
    group: demo.orkestra.io / version: v1 / plural: workloads
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

Open [http://localhost:8081](http://localhost:8081).

Select **workload-operator**, then select the **Workload** CRD.

---

## Step 4 — Apply the minimal CR

```bash
kubectl apply -f cr-minimal.yaml
```

This CR declares only `spec.image`. Every other field is absent.

> `phase` transitions from `Pending` to `Ready` once all pods are running — allow ~10s after applying.

```bash
kubectl get workload api-server -o yaml | grep -A10 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  replicas: "1"           # ← defaulted by normalize
  cpu: 100m               # ← defaulted by normalize
  memory: 128Mi           # ← defaulted by normalize
  concurrencyPolicy: Allow # ← defaulted by normalize
```

In the Control Center, click `api-server` then **top-right** to see the child Deployment. Open it — inspect **Status**, **Labels**, and **Events** to confirm the Deployment was created with the defaulted resource requests.

Now check what the user wrote:
```bash
kubectl get workload api-server -o yaml | grep -A5 "spec:"
```

`replicas`, `resources`, `concurrencyPolicy` are absent from the stored spec. Normalize filled them in only for the operator's in-memory view. etcd is untouched.

---

## Step 5 — Apply the full CR

```bash
kubectl apply -f cr-full.yaml
```

This CR declares everything explicitly with different values.

> `phase` transitions to `Ready` once all pods are running — allow ~10s after applying.

```bash
kubectl get workload worker -o yaml | grep -A10 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  replicas: "3"
  cpu: 250m
  memory: 512Mi
  concurrencyPolicy: Forbid
```

---

## When to use `default` in normalize vs `mutation:` rules

| | `default` in normalize | `mutation:` rules |
|---|---|---|
| Gateway required | No | Yes |
| Stored in etcd | No | Yes |
| Visible at apply time | No — reconcile time | Yes — admission response |
| Can combine with reshape | Yes | No — defaults only |

Use `mutation:` when the default needs to be stored and visible to external tools reading the CR. Use `default` in normalize for everything else.

---

## E2E

Run the full lifecycle in one command — applies the minimal CR, asserts that omitted fields received their defaults (replicas, CPU request), then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Status has defaulted replicas
    after: cr-applied
    timeout: 60s
    commands:
      - run: kubectl get workload api-server -o jsonpath='{.status.replicas}'
        outputContains: "1"

  - name: Deployment has defaulted resource requests
    after: cr-applied
    timeout: 60s
    commands:
      - run: kubectl get deployment api-server -o jsonpath='{.spec.template.spec.containers[0].resources.requests.cpu}'
        outputContains: 100m
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
