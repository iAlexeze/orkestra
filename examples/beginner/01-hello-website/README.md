# 01 — Hello Website

Your first Orkestra operator. You will declare a `Website` CRD, write a
twelve-line Katalog, and watch Orkestra create a Deployment from a CR — with
no Go code written.

**What you learn:** CRD declaration, apiTypes, Katalog template expressions, `reconcile: true`, status fields.

---

## How it works

When you apply a `Website` CR with `spec.image: nginx:1.25`, Orkestra:

1. Receives the watch event from the informer
2. Reads the CR from the informer cache
3. Resolves template expressions — `{{ .spec.image }}` → `"nginx:1.25"`, `{{ .metadata.name }}` → `"hello-website"`, etc.
4. Creates a Deployment named `hello-website` in the `default` namespace
5. Creates a Service named `hello-website-svc` exposing port 80
6. Writes `phase: Running` and `endpoint: hello-website.default.svc.cluster.local` to the CR's status
7. Sets owner references on the Deployment and Service pointing to the `Website` CR
8. Emits a `WebsiteReconciled` Kubernetes event on the CR

Delete the `Website` CR and Kubernetes automatically deletes both the Deployment and
Service — because of the owner references. No `onDelete` logic needed.

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

## Step 2 — Start the runtime

```bash
ork run       # add --dev if you don't have a cluster; Orkestra will create a kind cluster
```

Orkestra reads `crdFile: ./crd.yaml`, applies the CRD and `cr.yaml` to the cluster, and starts the operator. You will see the health server start and the informer sync:

```
{"level":"info","message":"health server listening on :8080"}
{"level":"info","message":"conversion https server: disabled"}
{"level":"info","crd":"demo.orkestra.io/v1alpha1, Kind=Website","message":"informer synced"}
{"level":"info","message":"✅ All komponents started successfully"}
```

---

## Step 3 — Watch the CR reconcile

`cr.yaml` is applied automatically by `ork run` before the runtime starts.
Watch the operator terminal — you will see the reconcile event arrive and the
Deployment and Service being created.

---

## Step 4 — Open the Control Center

In a second terminal:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081) to see the live operator — CRD health, worker state, reconcile metrics, and the `Website` CR you just created.

---

## Step 5 — Verify

```bash
# The CR exists
kubectl get websites

# The Deployment and Service were created
kubectl get deployments
kubectl get services

# Status was written
kubectl get website hello-website -o yaml | grep -A10 "status:"

# Owner references are set
kubectl get deployment hello-website -o yaml | grep -A5 ownerReferences

# The reconcile event was emitted
kubectl describe website hello-website | tail -10
```

Expected:

```
NAME           READY   UP-TO-DATE   AVAILABLE
hello-website  1/1     1            1

NAME               TYPE        CLUSTER-IP     PORT(S)
hello-website-svc  ClusterIP   10.96.x.x      80/TCP
```

Status:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
  phase: Running
  endpoint: hello-website.default.svc.cluster.local
```

---

## Step 6 — Drift correction and reconcile

**`reconcile: true`** is set on both the Deployment and Service. This means Orkestra
re-applies the desired state from the CR on every reconcile cycle — not just at
creation. If anything drifts, Orkestra corrects it back.

**Delete the Deployment manually** — Orkestra recreates it on the next reconcile:

```bash
kubectl delete deployment hello-website
kubectl get deployments
# hello-website reappears
```

**Update the image** — edit [cr.yaml](cr.yaml), change to `nginx:1.26`, and reapply:

```bash
kubectl apply -f cr.yaml
kubectl get deployment hello-website -o jsonpath='{.spec.template.spec.containers[0].image}' && echo
# nginx:1.26
```

**Scale the replicas:**

```bash
kubectl patch website hello-website --type=merge -p '{"spec":{"replicas":2}}'
kubectl get deployment hello-website
# READY: 2/2
```

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies the CRD, starts the operator, applies the CR, asserts every expectation, then tears down:

```bash
ork e2e
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
      - kind: Service
        namespace: default

  - name: Cleanup verified
    after: cr-deleted
    timeout: 30s
    resources:
      - kind: Deployment
        name: hello-website
        namespace: default
        count: 0
      - kind: Service
        name: hello-website-svc
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
