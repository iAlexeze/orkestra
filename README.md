# Orkestra Helm Chart

Deploy Orkestra — the declarative Kubernetes operator runtime — along with its Control Center for multi‑instance observability.

---

## Prerequisites

- Kubernetes 1.28+
- Helm 3.10+
- `kubectl` configured for your cluster
- `ork` CLI installed (see [Orkestra CLI installation](https://github.com/orkspace/orkestra))

---

## Before you begin

1. **Install the Orkestra CLI**

```bash
curl -sSL https://raw.githubusercontent.com/orkspace/orkestra/main/install.sh | bash
```

2. **Create a minimal Katalog** (save as `katalog.yaml`)

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

> [!TIP]
> This assumes you have a website CRD installed in your cluster. If not, generate one from your Katalog:
> ```bash
> ork generate crd -k katalog.yaml -o my-website-crd.yaml
> kubectl apply -f my-website-crd.yaml
> ```

---
## Security‑First Installation (Recommended)

Orkestra generates **minimal RBAC** from your Katalog – no wildcards, no excess permissions.  
The entire flow from zero to running is **four steps**:

### Step 1 – Generate a bundle

```bash
ork generate bundle --katalog katalog.yaml -o bundle.yaml
```

This produces a single YAML file containing:
- `ServiceAccounts` for Runtime (`orkestra`) and Control Center (`orkestra-cc`)
- A `ClusterRole` with **only** the permissions your Katalog needs
- A `ClusterRoleBinding`
- A `ConfigMap` named `orkestra-katalog` with your Katalog data

### Step 2 – Apply the bundle

```bash
kubectl create namespace orkestra-system   # if not already present
kubectl apply -f bundle.yaml
```

### Step 3 – Deploy with Helm

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm install orkestra orkestra/orkestra --namespace orkestra-system
```

That’s it. The Helm chart automatically uses the `ServiceAccount` and `ConfigMap` you just applied.

### Step 4 – Verify everything works

```bash
kubectl get pods -n orkestra-system
kubectl get websites -A   # (if your Katalog defines the Website CRD)
```

> [!IMPORTANT]
> **Why this matters**: Traditional operators are massively over‑permissioned. Orkestra generates RBAC from your **declared intent**, giving you least‑privilege security by default.

---

## Customising service accounts (advanced)

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

## Install with your own Katalog (advanced)

You can also provide the Katalog directly to Helm without using a ConfigMap bundle:

```yaml
# my-values.yaml
runtime:
  katalog:
    inline: |
      apiVersion: orkestra.orkspace.io/v1Alpha
      kind: Katalog
      metadata:
        name: my-katalog
      spec:
        crds:
          - name: website
            enabled: true
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

## Control Center multi‑runtime monitoring

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

Uninstalling removes the Orkestra Runtime and Control Center Deployments and all chart resources.  
CRDs and custom resources that Orkestra was managing are **not** deleted – they remain in the cluster and must be cleaned up separately if desired.

---

## Configuration Reference

<details>
<summary>Click to expand full configuration options</summary>


### Runtime Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `runtime.enabled` | Deploy the Orkestra runtime | `true` |
| `runtime.image.repository` | Runtime container image | `ghcr.io/orkspace/orkestra` |
| `runtime.image.tag` | Image tag | Chart `appVersion` |
| `runtime.image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `runtime.replicaCount` | Number of replicas (leader election) | `2` |
| `runtime.service.type` | Service type | `ClusterIP` |
| `runtime.service.port` | Service port | `8080` |
| `runtime.service.annotations` | Service annotations | `{}` |
| `runtime.resources` | Resource limits and requests | See [values.yaml](https://github.com/orkspace/orkestra/blob/main/charts/orkestra/values.yaml) |
| `runtime.serviceAccount` | ServiceAccount name (must match generated RBAC) | `"orkestra"` |
| `runtime.server.readTimeout` | HTTP server read timeout (seconds) | `30` |
| `runtime.server.writeTimeout` | HTTP server write timeout (seconds) | `60` |
| `runtime.server.port` | HTTP server port | `8080` |
| `runtime.config.logLevel` | Log level: debug, info, warn, error | `info` |
| `runtime.config.defaultWorkers` | Default reconcile workers per CRD | `2` |
| `runtime.config.defaultResync` | Default resync interval | `30s` |
| `runtime.config.maxQueueDepth` | Max workqueue depth per CRD | `500` |
| `runtime.config.degradeThreshold` | Consecutive failures before degraded | `10` |
| `runtime.config.environment` | Deployment environment | `development` |
| `runtime.config.watchNamespace` | Restrict to single namespace (empty = all) | `""` |
| `runtime.startupProbe` | Startup probe configuration | See [values.yaml](https://github.com/orkspace/orkestra/blob/main/charts/orkestra/values.yaml) |
| `runtime.livenessProbe` | Liveness probe configuration | See values.yaml |
| `runtime.readinessProbe` | Readiness probe configuration | See values.yaml |
| `runtime.podSecurityContext` | Pod security context | Non-root (1000) |
| `runtime.securityContext` | Container security context | Non-root, read-only root |
| `runtime.leaderElection.enabled` | Enable leader election for HA | `true` |
| `runtime.leaderElection.leaseDuration` | Leader lease duration | `15s` |
| `runtime.leaderElection.renewDeadline` | Renew deadline | `10s` |
| `runtime.leaderElection.retryPeriod` | Retry period | `5s` |
| `runtime.webhooks.enabled` | Master switch for webhooks | `false` |
| `runtime.webhooks.admission` | Enable admission webhook | `false` |
| `runtime.webhooks.conversion` | Enable conversion webhook | `false` |
| `runtime.webhooks.existingSecret` | TLS secret name (required if enabled) | `""` |
| `runtime.webhooks.certSecretKey` | Key in secret for certificate | `tls.crt` |
| `runtime.webhooks.keySecretKey` | Key in secret for private key | `tls.key` |
| `runtime.katalog.inline` | Inline Katalog YAML | (starter Katalog) |
| `runtime.katalog.existingConfigMap` | Use existing ConfigMap | `""` |
| `runtime.katalog.configMapKey` | Key in ConfigMap | `katalog.yaml` |
| `runtime.katalog.mountPath` | Mount path inside container | `/etc/orkestra/katalog` |

### Control Center Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `controlCenter.enabled` | Deploy the Control Center | `true` |
| `controlCenter.image.repository` | Control Center image | `ghcr.io/orkspace/orkestra-cc` |
| `controlCenter.image.tag` | Image tag | Chart `appVersion` |
| `controlCenter.image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `controlCenter.replicaCount` | Number of replicas | `1` |
| `controlCenter.resources` | Resource limits and requests | See values.yaml |
| `controlCenter.serviceAccount` | ServiceAccount name | `"orkestra-cc"` |
| `controlCenter.config.orkestraURLs` | List of runtime URLs to monitor | `[]` (must be set in production) |
| `controlCenter.config.port` | Control Center port | `8090` |
| `controlCenter.config.refreshInterval` | Katalog refresh interval | `10s` |
| `controlCenter.config.logLevel` | Log level | `info` |
| `controlCenter.service.type` | Service type | `ClusterIP` |
| `controlCenter.service.port` | Service port | `8081` |
| `controlCenter.service.annotations` | Service annotations | `{}` |
| `controlCenter.ingress.enabled` | Enable Ingress | `false` |
| `controlCenter.ingress.className` | Ingress class name | `""` |
| `controlCenter.ingress.annotations` | Ingress annotations | `{}` |
| `controlCenter.ingress.hosts` | Ingress hosts | `[]` |
| `controlCenter.ingress.tls` | TLS configuration | `[]` |
| `controlCenter.securityContext` | Container security context | Non-root (1001) |
| `controlCenter.podSecurityContext` | Pod security context | Non-root (1001) |
| `controlCenter.livenessProbe` | Liveness probe configuration | See values.yaml |
| `controlCenter.readinessProbe` | Readiness probe configuration | See values.yaml |

### Shared Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `imagePullSecrets` | Image pull secrets | `[]` |
| `nameOverride` | Override chart name | `""` |
| `fullnameOverride` | Override full name | `""` |
| `registry.enabled` | Enable registry support | `true` |
| `registry.url` | Global registry URL | `""` |
| `hpa.enabled` | Enable HorizontalPodAutoscaler | `false` |
| `hpa.runtime` | HPA configuration for runtime | See values.yaml |
| `hpa.controlCenter` | HPA configuration for control center | See values.yaml |
| `networkPolicy.enabled` | Enable NetworkPolicy | `false` |
| `networkPolicy.ingressFrom` | Allowed ingress sources | `[]` |
| `pdb.runtime.enabled` | Enable PDB for runtime | `true` |
| `pdb.runtime.minAvailable` | Min available runtime pods | `1` |
| `pdb.controlCenter.enabled` | Enable PDB for control center | `true` |
| `pdb.controlCenter.minAvailable` | Min available control center pods | `1` |
| `nodeSelector` | Node selector for all pods | `{}` |
| `tolerations` | Tolerations for all pods | `[]` |
| `affinity` | Affinity for all pods | `{}` |
| `topologySpreadConstraints` | Topology spread constraints | `[]` |
| `extraEnv` | Extra environment variables | `[]` |
| `extraEnvFrom` | Extra environment variables from sources | `[]` |
| `extraVolumes` | Extra volumes | `[]` |
| `extraVolumeMounts` | Extra volume mounts | `[]` |
| `podAnnotations` | Annotations for all pods | `{}` |
| `podLabels` | Labels for all pods | `{}` |

</details>

---

## Production Example

<details>
<summary>Click to expand full production values</summary>


```yaml
# production-values.yaml
runtime:
  replicaCount: 3
  serviceAccount: orkestra
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 1Gi
  config:
    logLevel: warn
    defaultWorkers: 4
    defaultResync: 1m
  katalog:
    existingConfigMap: platform-katalog   # managed by GitOps

controlCenter:
  enabled: true
  replicaCount: 2
  serviceAccount: orkestra-cc
  config:
    orkestraURLs:
      - http://orkestra-runtime:8080
    refreshInterval: 30s
  ingress:
    enabled: true
    className: nginx
    annotations:
      cert-manager.io/cluster-issuer: letsencrypt-prod
    hosts:
      - host: control-center.platform.myorg.io
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: control-center-tls
        hosts:
          - control-center.platform.myorg.io

hpa:
  enabled: true
  runtime:
    minReplicas: 3
    maxReplicas: 8
    targetCPUUtilizationPercentage: 70
  controlCenter:
    minReplicas: 2
    maxReplicas: 5
    targetCPUUtilizationPercentage: 70

networkPolicy:
  enabled: true
  ingressFrom:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: monitoring
```
</details>

```bash
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system --create-namespace \
  --values production-values.yaml
```

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

# Open in browser
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

Regenerate the bundle and re‑apply:

```bash
ork generate bundle --katalog katalog.yaml -o bundle.yaml
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

- **Runtime**  
  - `runAsNonRoot: true` with uid/gid 1000  
  - `readOnlyRootFilesystem: true` (with `/tmp` as emptyDir)  
  - `allowPrivilegeEscalation: false`  
  - `capabilities.drop: [ALL]`  
  - `seccompProfile: RuntimeDefault`  
  - Pod anti‑affinity spreads replicas across nodes

- **Control Center**  
  - `runAsNonRoot: true` with uid/gid 1001  
  - `readOnlyRootFilesystem: true`  
  - `allowPrivilegeEscalation: false`  
  - `capabilities.drop: [ALL]`  
  - No RBAC permissions (read‑only access to Runtime API only)

---

## Why Security‑First?

| Traditional operators | Orkestra |
|----------------------|----------|
| Wildcard permissions (`*/*/*`) | Minimal, derived from your Katalog |
| RBAC written by hand | Generated automatically |
| Drifts over time | Always in sync |
| Over‑permissioned by default | Least privilege by default |

The generated bundle gives you exactly what your Katalog declares – nothing more.

---

## License

Same license as the Orkestra project.
