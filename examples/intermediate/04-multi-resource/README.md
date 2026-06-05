# 04 — Multi-Resource with Status

Three resources from one CR. Status fields that reflect both the declared spec
and the live state of a child resource — Layer 3 child status propagation.

**What you learn:** ConfigMap, all three status layers, `readyReplicas` from the
live Deployment status, multi-field status updates.

**Builds on:** [02 — Website with Service](../../beginner/02-website-with-service/README.md)

---

## What is new

**ConfigMap** — The `Application` CR produces a ConfigMap alongside the Deployment
and Service. The ConfigMap contains configuration values resolved from the CR spec.
With `reconcile: true`, if you change `spec.logLevel`, the ConfigMap is updated
within one resync interval.

**Layer 3 status** — After the Deployment is created, Orkestra reads it back and
makes its status available in the template resolver. The expression
`{{ .children.deployment.status.readyReplicas }}` resolves to the actual number
of ready replicas from Kubernetes — not the desired count from the spec.

This distinction matters. `observedReplicas: "{{ .spec.replicas }}"` tells you
what was declared. `readyReplicas: "{{ .children.deployment.status.readyReplicas }}"`
tells you what Kubernetes has actually achieved.

---

## Steps

### 1. Start the runtime

```bash
ork run
# Orkestra reads katalog.yaml, applies the CRD and cr.yaml, and starts the operator.
```

### 2. Verify all three resources

```bash
kubectl get deployments,services,configmaps | grep my-app
```

Expected:
```
deployment.apps/my-app       2/2   2   2
service/my-app-svc           ClusterIP   ...
configmap/my-app-config      2
```

### 5. Verify the ConfigMap data

```bash
kubectl get configmap my-app-config -o yaml
```

```yaml
data:
  LOG_LEVEL: info
  MAX_CONNECTIONS: "100"
```

### 6. Check status — Layer 2 and Layer 3

```bash
kubectl get application my-app -o yaml | grep -A20 "status:"
```
> [!IMPORTANT]  
> If you get the error: 'Error from server (NotFound): applications.argoproj.io "my-app" not found', Use `kubectl get application.demo.orkestra.io my-app -o yaml` instead of `application` to avoid conflict with ArgoCD application if installed in your cluster.

On first reconcile, `readyReplicas` may be empty (Deployment still starting).
Wait a few seconds and re-check — it populates as soon as the Deployment becomes ready:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
  phase: Running
  observedReplicas: "2"
  readyReplicas: "2"          ← from the live Deployment status
  availableReplicas: "2"      ← from the live Deployment status
  endpoint: my-app.default.svc.cluster.local
```

### 7. Update configuration live

Change the log level and watch the ConfigMap update:

```bash
kubectl patch application my-app --type=merge \
  -p '{"spec":{"logLevel":"debug","maxConnections":500}}'
```

Within 15 seconds:

```bash
kubectl get configmap my-app-config -o jsonpath='{.data.LOG_LEVEL}'
# debug
```

No Deployment restart needed — the ConfigMap update is live. If your
application reads configuration at startup rather than dynamically, you
would add a Deployment annotation trigger — that is covered in example 07.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
