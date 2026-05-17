# Deploying Orkestra

Deploy Orkestra — the declarative Kubernetes operator runtime — along with its Control Center for multi-instance observability.

---

## Prerequisites

- Kubernetes 1.28+
- Helm 3.10+
- `kubectl` configured for your cluster
- `ork` CLI installed

---

## Before you begin

**1. Install the Orkestra CLI**

```bash
# macOS
brew install orkspace/tap/ork

# Linux / macOS (curl)
curl -sSL https://get.orkestra.sh | bash
```

**2. Create a minimal Katalog** (save as `katalog.yaml`)

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: my-first-katalog
spec:
  crds:
    website:
      enabled: true
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites
      operatorBox:
        default: true
        onCreate:
          deployments:
            - image: nginx
              replicas: 1
```

> **Tip:** This assumes you have the Website CRD installed in your cluster. If not, generate one from your Katalog:
> ```bash
> ork generate crd -f katalog.yaml -o my-website-crd.yaml
> kubectl apply -f my-website-crd.yaml
> ```

---

## Security-First Installation (Recommended)

Orkestra generates **minimal RBAC** from your Katalog — no wildcards, no excess permissions. The entire flow from zero to running is four steps.

### Step 1 — Generate a bundle

```bash
ork generate bundle -f katalog.yaml -o bundle.yaml
```

This produces a single YAML file containing:

- A `Namespace` named `orkestra-system` (use `-n <my-namespace>` to override)
- `ServiceAccounts` for Runtime (`orkestra`) and Control Center (`orkestra-cc`)
- A `ClusterRole` with **only** the permissions your Katalog needs
- A `ClusterRoleBinding`
- A `ConfigMap` named `orkestra-katalog` with your Katalog data

### Step 2 — Apply the bundle

```bash
kubectl apply -f bundle.yaml
```

### Step 3 — Deploy with Helm

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm install orkestra orkestra/orkestra --namespace orkestra-system
```

Replace `orkestra-system` if you used a different namespace in Step 1.

### Step 4 — Verify everything works

```bash
kubectl get pods -n orkestra-system
kubectl get websites -A   # if your Katalog defines the Website CRD
```

> **Why this matters**: Traditional operators are massively over-permissioned. Orkestra generates RBAC from your **declared intent**, giving you least-privilege security by default.

---

## Customising service accounts

If you renamed the `ServiceAccounts` in the generated bundle, pass the custom names to Helm:

```bash
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --set runtime.serviceAccount=my-runtime \
  --set controlCenter.serviceAccount=my-cc
```

---

## Install runtime only (without Control Center)

```yaml
# runtime-only.yaml
controlCenter:
  enabled: false
```

```bash
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --values runtime-only.yaml
```

---

## Install with your own Katalog

You can provide the Katalog directly to Helm without using a ConfigMap bundle:

```yaml
# my-values.yaml
runtime:
  katalog:
    inline: |
      apiVersion: orkestra.orkspace.io/v1
      kind: Katalog
      metadata:
        name: my-katalog
      spec:
        crds:
          # ... rest of your Katalog
```

```bash
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --values my-values.yaml
```

Or use an existing ConfigMap managed by GitOps:

```yaml
runtime:
  katalog:
    existingConfigMap: my-katalog-configmap
    configMapKey: katalog.yaml
```

---

## Control Center multi-runtime monitoring

To monitor multiple Orkestra runtimes, set the list of runtime URLs:

```yaml
# control-center-values.yaml
controlCenter:
  config:
    orkestraURLs:
      - http://orkestra-prod:8080
      - http://orkestra-staging:8080
    refreshInterval: 30s
  ingress:
    enabled: true
    hosts:
      - host: control-center.orkestra.io
```

```bash
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --values control-center-values.yaml
```

---

## Upgrade

```bash
helm repo update
helm upgrade orkestra orkestra/orkestra --namespace orkestra-system
```

---

## Uninstall

```bash
helm uninstall orkestra --namespace orkestra-system
```

Uninstalling removes the Orkestra Runtime and Control Center Deployments and all chart resources. CRDs and custom resources that Orkestra was managing are **not** deleted — they remain in the cluster and must be cleaned up separately if desired.

---

## Configuration Reference

For the full Helm values reference — all runtime, Control Center, HPA, NetworkPolicy, webhook, and per-component options — see the [Orkestra Helm chart documentation](https://orkspace.github.io/orkestra/).

---

## Observability

After installing, Orkestra exposes:

### Runtime Endpoints

```bash
kubectl port-forward svc/orkestra-runtime 8080:8080 -n orkestra-system

curl localhost:8080/health          # liveness
curl localhost:8080/ready           # readiness
curl localhost:8080/metrics         # Prometheus metrics
curl localhost:8080/katalog | jq    # all CRDs
```

### Control Center Endpoints

```bash
kubectl port-forward svc/orkestra-cc 8081:8081 -n orkestra-system

open http://localhost:8081/controlcenter
```

### Prometheus Scrape Configuration

```yaml
- job_name: orkestra-runtime
  static_configs:
    - targets: ['orkestra-runtime.orkestra-system.svc.cluster.local:8080']
  metrics_path: /metrics

- job_name: orkestra-control-center
  static_configs:
    - targets: ['orkestra-cc.orkestra-system.svc.cluster.local:8081']
  metrics_path: /metrics
```

---

## Troubleshooting

### Control Center cannot connect to Runtime

```bash
kubectl get svc -n orkestra-system
kubectl logs -n orkestra-system deployment/orkestra-cc
```

Ensure `orkestraURLs` is set correctly.

### Katalog not loaded

```bash
kubectl get configmap -n orkestra-system
kubectl logs -n orkestra-system deployment/orkestra-runtime | grep -i katalog
```

### RBAC permission denied

Regenerate the bundle and re-apply:

```bash
ork generate bundle -f katalog.yaml -o bundle.yaml
kubectl apply -f bundle.yaml
```

### Webhooks not working

Verify TLS secret exists and webhooks are enabled:

```bash
kubectl get secret -n orkestra-system
kubectl logs -n orkestra-system deployment/orkestra-runtime | grep -i webhook
```

---

## Security Defaults

Out of the box, the chart applies:

**Runtime**
- `runAsNonRoot: true` with uid/gid 1000
- `readOnlyRootFilesystem: true` (with `/tmp` as emptyDir)
- `allowPrivilegeEscalation: false`
- `capabilities.drop: [ALL]`
- `seccompProfile: RuntimeDefault`
- Pod anti-affinity spreads replicas across nodes

**Control Center**
- `runAsNonRoot: true` with uid/gid 1001
- `readOnlyRootFilesystem: true`
- `allowPrivilegeEscalation: false`
- `capabilities.drop: [ALL]`
- No RBAC permissions (read-only access to Runtime API only)

---

## Why Security-First?

| Traditional operators | Orkestra |
|----------------------|----------|
| Wildcard permissions (`*/*/*`) | Minimal, derived from your Katalog |
| RBAC written by hand | Generated automatically |
| Drifts over time | Always in sync |
| Over-permissioned by default | Least privilege by default |

The generated bundle gives you exactly what your Katalog declares — nothing more.
