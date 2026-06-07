# 01 — Hello Website

Your first Orkestra operator. You will declare a `Website` CRD, write a
twelve-line Katalog, and watch Orkestra create a Deployment from a CR — with
no Go code written.

**What you learn:** CRD declaration, apiTypes, the minimal Katalog, template expressions.

---

## How it works

When you apply a `Website` CR with `spec.image: nginx:1.25`, Orkestra:

1. Receives the watch event from the informer
2. Reads the CR from the informer cache
3. Resolves `{{ .spec.image }}` → `"nginx:1.25"`
4. Creates a Deployment named `hello-deployment` in the `default` namespace
5. Sets an owner reference on the Deployment pointing to the `Website` CR
6. Emits a `Reconciled` Kubernetes event on the CR

Delete the `Website` CR and Kubernetes automatically deletes the Deployment —
because of the owner reference. No `onDelete` logic needed.

---

## Step 1 — Install the CRD

```bash
kubectl apply -f crd.yaml
```

Verify:

```bash
kubectl get crd websites.demo.orkestra.io
```

---

## Step 2 — Validate the Katalog

```bash
ork validate --file katalog.yaml
```

Expected output:

```
✓ website
    kind: Website
    group: demo.orkestra.io / version: v1alpha1 / plural: websites
    mode: dynamic / workers: 3 / resync: 15s
```

---

## Step 3 — Start the runtime

```bash
ork run --file katalog.yaml
```

You will see the health server start and the informer sync:

```
{"level":"info","message":"health server listening on :8080"}
{"level":"info","message":"conversion https server: disabled"}
{"level":"info","crd":"demo.orkestra.io/v1alpha1, Kind=Website","message":"informer synced"}
{"level":"info","message":"✅ All komponents started successfully"}
```

---

## Step 4 — Apply the CR

Open a second terminal:

```bash
kubectl apply -f cr.yaml
```

Watch the operator terminal. You will see the reconcile event arrive and the
Deployment being created.

---

## Step 5 — Verify

```bash
# The CR exists
kubectl get websites

# The Deployment was created
kubectl get deployments

# The owner reference is set
kubectl get deployment hello-deployment -o yaml | grep -A5 ownerReferences

# The reconcile event was emitted
kubectl describe website hello | tail -10
```

Expected deployment name: `hello-deployment`

---

## Step 6 — Check the health API

```bash
curl localhost:8080/katalog/website/health | jq '{
  healthy: .healthy,
  resourceCount: .resourceCount,
  totalReconciles: .totalReconciles
}'
```

Expected output:
```json
{
  "healthy": true,
  "resourceCount": 1,
  "totalReconciles": 1
}
```

> [!NOTE]
> Notice how there are no reconciles yet, — because `reconcile: true` is not set.
> This is intentional: this example shows `onCreate` only.

In example 02 you will add `reconcile: true` and see the difference.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

Or manually:

```bash
kubectl delete -f cr.yaml
kubectl delete -f crd.yaml
# Stop ork run with Ctrl+C
```
