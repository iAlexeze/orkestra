# 02 — Website with Service

Two resources, drift correction, and your first status fields. This is the
pattern most operators implement — a Deployment and a Service — in twelve
lines of Katalog YAML.

**What you learn:** Multiple resources, `reconcile: true`, status Layer 1 and Layer 2.

**Builds on:** [01 — Hello Website](../01-hello-website/README.md)

---

## What is new

**`reconcile: true`** on the Deployment and Service means Orkestra re-applies
the desired state on every reconcile cycle, not just on creation. Delete the
Deployment manually and it reappears within one resync interval (default 15s).
This is drift correction.

**Status fields** declared in the Katalog are written to the CR's `/status`
subresource after every successful reconcile. The `Ready` condition (Layer 1)
is written automatically — you declared the status subresource in the CRD and
Orkestra does the rest. The `phase`, `observedReplicas`, and `endpoint` fields
(Layer 2) are declared in the Katalog.

---

## Steps

### 1. Install the CRD

```bash
kubectl apply -f crd.yaml
```

### 2. Start the operator

```bash
ork run --katalog katalog.yaml
```

### 3. Apply the CR

```bash
kubectl apply -f cr.yaml
```

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

### 5. Verify status

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

Delete the Deployment and watch it come back:

```bash
kubectl delete deployment my-site
```

Wait up to 15 seconds (the resync interval):

```bash
kubectl get deployments
# my-site reappears
```

The Service selector always routes to pods owned by the same CR — the
`orkestra-owner: my-site` label is set automatically by Orkestra.

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
kubectl get website my-site -o jsonpath='{.status.observedReplicas}'
# "3"
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
