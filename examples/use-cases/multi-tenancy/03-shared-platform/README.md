# Multi-tenancy 03 — Shared platform

`platform` manages shared infrastructure CRDs (Cache, Queue). `team-a` manages an Api CRD that reads cache and queue endpoints via `cross:` and injects them as environment variables into its Deployment. The Deployment is only created once both platform CRs are healthy.

**What you learn:** cross reads between namespaces, readiness gating on shared infrastructure, `eqTernary` for string-based status branching.

---

## Step 1 — Validate

```bash
ork validate
```

Expected:
```
● cache   kind: Cache / group: multi-tenancy.orkestra.io / version: v1alpha1
● queue   kind: Queue / group: multi-tenancy.orkestra.io / version: v1alpha1
● api     kind: Api   / group: multi-tenancy.orkestra.io / version: v1alpha1

3 CRDs valid
```

---

## Step 2 — Apply the CRDs

```bash
kubectl apply -f crd.yaml
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081).

---

## Step 4 — Start the operator

```bash
ork run
```

Two namespace panels appear: **platform** (cache, queue) and **team-a** (api).

---

## Step 5 — Apply the CRs

```bash
kubectl apply -f cr.yaml
```

Watch the Api status as the platform CRs reconcile:

```bash
kubectl get apis,caches,queues -w
```

While `app-cache` and `app-queue` are reconciling, `my-api` shows `phase: waiting`. Once both platform CRs reach `phase: ready`, the Api Deployment is created and status flips to `running`.

---

## Step 6 — Verify env injection

```bash
kubectl get deployment my-api -o jsonpath='{.spec.template.spec.containers[0].env}' | jq .
```

Expected:
```json
[
  { "name": "REDIS_URL",  "value": "app-cache.default.svc.cluster.local:6379" },
  { "name": "AMQP_URL",   "value": "amqp://app-queue.default.svc.cluster.local" }
]
```

Orkestra injected the platform endpoints from the informer cache — zero API server calls.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
