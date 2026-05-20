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

## Step 1 — Validate the Katalog

```bash
ork validate
```

Expected output:

```
✓ website
    kind: Website
    group: demo.orkestra.io / version: v1alpha1 / plural: websites
    mode: dynamic / workers: 3 / resync: 15s
```

`ork validate` resolves `crdFile`, reads the CRD, and reports what Orkestra found — without touching the cluster.

---

## Step 2 — Start the operator

```bash
ork run
```

Orkestra reads `crdFile: ./crd.yaml`, applies the CRD to the cluster, and starts the operator. You will see the health server start and the informer sync:

```
{"level":"info","message":"health server listening on :8080"}
{"level":"info","message":"conversion https server: disabled"}
{"level":"info","crd":"demo.orkestra.io/v1alpha1, Kind=Website","message":"informer synced"}
{"level":"info","message":"✅ All komponents started successfully"}
```

---

## Step 3 — Apply the CR

Open a second terminal:

```bash
kubectl apply -f cr.yaml
```

Watch the operator terminal. You will see the reconcile event arrive and the
Deployment being created.

---

## Step 4 — Open the Control Center

In a third terminal:

```bash
ork control
# username:password → orkestra
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081) to see the live operator — CRD health, worker state, reconcile metrics, and the `Website` CR you just created.

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

## Step 6 — Understand onCreate

**Deletion is always recovered.** Delete the Deployment manually — owner references mean Kubernetes garbage-collects it, and Orkestra recreates it on the next reconcile:

```bash
kubectl delete deployment hello-deployment
kubectl get deployments
# hello-deployment reappears
```

**Updating the CR does not update the Deployment.** Edit [cr.yaml](cr.yaml), change the image to `nginx:1.26`, and reapply:

```bash
kubectl apply -f cr.yaml
```

Check the Deployment image:

```bash
kubectl get deployment hello-website-deployment -o jsonpath='{.spec.template.spec.containers[0].image}' && echo
# nginx:1.25  — unchanged
```

This is intentional. `onCreate` only fires when the CR is first created. To make Orkestra update existing resources when the CR changes, add `reconcile: true` — which is what example 02 introduces.

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies the CRD, starts the operator, applies the CR, asserts every expectation, then tears down:

```bash
ork e2e -f e2e.yaml
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Deployment created
    after: cr-applied
    timeout: 60s
    resources:
      - kind: Deployment
        namespace: default
        ready: true

  - name: Cleanup verified
    after: cr-deleted
    timeout: 30s
    resources:
      - kind: Deployment
        name: hello-website
        namespace: default
        count: 0
```

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
