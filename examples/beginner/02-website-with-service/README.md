# 02 — Website with Service

Two resources, drift correction, and your first status fields. This is the
pattern most operators implement — a Deployment and a Service — in twelve
lines of Katalog YAML.

**What you learn:** Multiple resources, `reconcile: true`, status Layer 1 and Layer 2.

**Builds on:** [01 — Hello Website](../01-hello-website/README.md)

---

## What is new

**`reconcile: true`** on the Deployment and Service means Orkestra re-applies
the desired state from the CR on every reconcile cycle. In example 01, updating
the CR's image left the Deployment unchanged — because `onCreate` only fires at
creation. Here, updating the CR to `nginx:1.26` and reapplying immediately
updates the Deployment. That is drift correction: the live resource always
converges to what the CR declares.

**Status fields** declared in the Katalog are written to the CR's `/status`
subresource after every successful reconcile. The `Ready` condition (Layer 1)
is written automatically — you declared the status subresource in the CRD and
Orkestra does the rest. The `phase`, `observedReplicas`, and `endpoint` fields
(Layer 2) are declared in the Katalog.

---

## Steps

### 1. Start the operator

```bash
ork run -f katalog.yaml
```

Orkestra reads `crdFile: ./crd.yaml`, applies the CRD to the cluster, and starts the operator.

### 2. Apply the CR

```bash
kubectl apply -f cr.yaml
```

### 3. Open the Control Center

In a third terminal:

```bash
ork control start
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081) to see the live operator.

### 4. Verify resources

```bash
kubectl get websites
kubectl get deployments
kubectl get services
```

Expected:
```
NAME      READY   UP-TO-DATE   AVAILABLE
my-site   2/2     2            2

NAME          TYPE        CLUSTER-IP     PORT(S)
my-site-svc   ClusterIP   10.96.x.x      80/TCP
```

### 4. Verify status

```bash
kubectl get website my-site -o yaml | grep -A20 "status:"
```

Expected:
```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
      lastTransitionTime: "..."
      observedGeneration: 1
  observedGeneration: 1
  phase: Running
  observedReplicas: "2"
  endpoint: my-site.default.svc.cluster.local
```

### 6. Test drift correction

Update [cr.yaml](cr.yaml) to change the image to `nginx:1.26` and reapply:

```bash
kubectl apply -f cr.yaml
```

Check the Deployment image — Orkestra has already corrected it:

```bash
kubectl get deployment my-site -o jsonpath='{.spec.template.spec.containers[0].image}' && echo
# nginx:1.26
```

This works because `reconcile: true` is set on the Deployment in this Katalog. Every reconcile cycle, Orkestra re-applies the desired state from the CR. If anything drifts — a manual edit, a rollback, anything — Orkestra corrects it back.

This is the difference from example 01: there, updating the CR left the Deployment unchanged. Here, it updates immediately.

### 7. Update the replica count

```bash
kubectl patch website my-site --type=merge -p '{"spec":{"replicas":3}}'
```

Watch the Deployment scale:

```bash
kubectl get deployment my-site
# READY: 3/3
```

The status updates too:

```bash
kubectl get website my-site -o jsonpath='{.status.observedReplicas}' && echo
# "3"
```

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies the CRD, starts the operator, applies the CR, asserts every expectation, then tears down:

```bash
ork e2e -f e2e.yaml
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Deployment and Service created
    after: cr-applied
    timeout: 60s
    resources:
      - kind: Deployment
        namespace: default
        ready: true
      - kind: Service
        name: my-site
        namespace: default

  - name: Cleanup verified
    after: cr-deleted
    timeout: 30s
    resources:
      - kind: Deployment
        name: my-site
        namespace: default
        count: 0
      - kind: Service
        name: my-site
        namespace: default
        count: 0
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
