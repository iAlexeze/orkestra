# Orkestra Helm Chart

Deploy Orkestra — the declarative Kubernetes operator runtime — along with its Control Center for multi-instance observability.

```bash
helm repo add orkestra https://ialexeze.github.io/orkestra
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace
```

---

## Prerequisites

- Kubernetes 1.28+
- Helm 3.10+
- `kubectl` configured to point at your cluster

---

## Components

The chart installs two separate components:

| Component | Description | Default |
|-----------|-------------|---------|
| **Runtime** | The Orkestra operator runtime that manages CRDs | Enabled |
| **Control Center** | Web UI for monitoring multiple Orkestra instances | Enabled |

---

## Install

### Quick install with defaults

```bash
helm repo add orkestra https://ialexeze.github.io/orkestra
helm repo update

helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace
```

### Install runtime only (without Control Center)

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

### Install with your own Katalog

The Katalog is mounted as a ConfigMap volume. Pass it inline in your values:

```yaml
# my-values.yaml
runtime:
  katalog:
    inline: |
      apiVersion: orkestra.konductor.io/v1Alpha
      kind: Katalog
      metadata:
        name: my-katalog
      spec:
        crds:
          - name: website
            enabled: true
            apiTypes:
              group: demo.orkestra.io
              version: v1alpha1
              kind: Website
              plural: websites
            reconciler:
              default: true
              onCreate:
                deployments:
                  - image: "{{ .spec.image }}"
                    replicas: "{{ .spec.replicas }}"
                    reconcile: true
```

```bash
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --values my-values.yaml
```

### Use an existing ConfigMap for the Katalog

If you manage the Katalog separately (e.g. with Flux or ArgoCD):

```yaml
runtime:
  katalog:
    existingConfigMap: my-katalog-configmap
    configMapKey: katalog.yaml
```

### Configure Control Center with multiple runtimes

```yaml
# control-center-values.yaml
controlCenter:
  enabled: true
  config:
    orkestraURLs:
      - http://orkestra-prod:8080
      - http://orkestra-staging:8080
      - http://orkestra-dev:8080
    refreshInterval: 30s
  ingress:
    enabled: true
    className: nginx
    hosts:
      - host: control-center.orkestra.io
        paths:
          - path: /
            pathType: Prefix
```

---

## Upgrade

```bash
helm repo update
helm upgrade orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --values my-values.yaml
```

---

## Uninstall

```bash
helm uninstall orkestra --namespace orkestra-system
```

Uninstalling removes the Orkestra Runtime and Control Center Deployments and all chart resources.
CRs and CRDs that Orkestra was managing are **not** deleted — they
remain in the cluster and must be cleaned up separately if desired.

---

## Configuration

### Runtime Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `runtime.enabled` | Deploy the Orkestra runtime | `true` |
| `runtime.image.repository` | Runtime container image | `ghcr.io/orkspace/orkestra` |
| `runtime.image.tag` | Image tag | Chart `appVersion` |
| `runtime.replicaCount` | Number of replicas (leader election) | `2` |
| `runtime.service.type` | Service type | `ClusterIP` |
| `runtime.service.port` | Service port | `8080` |
| `runtime.resources` | Resource limits and requests | See values.yaml |
| `runtime.serviceAccount.create` | Create ServiceAccount | `true` |
| `runtime.server.readTimeout` | HTTP server read timeout (seconds) | `30` |
| `runtime.server.writeTimeout` | HTTP server write timeout (seconds) | `60` |
| `runtime.config.logLevel` | Log level: debug, info, warn, error | `info` |
| `runtime.config.defaultWorkers` | Default reconcile workers per CRD | `2` |
| `runtime.config.defaultResync` | Default resync interval | `30s` |
| `runtime.config.maxQueueDepth` | Max workqueue depth per CRD | `500` |
| `runtime.config.degradeThreshold` | Consecutive failures before degraded | `10` |
| `runtime.config.environment` | Deployment environment | `development` |
| `runtime.config.watchNamespace` | Restrict to single namespace (empty = all) | `""` |
| `runtime.leaderElection.enabled` | Enable leader election for HA | `true` |
| `runtime.leaderElection.leaseDuration` | Leader lease duration | `15s` |
| `runtime.leaderElection.renewDeadline` | Renew deadline | `10s` |
| `runtime.leaderElection.retryPeriod` | Retry period | `5s` |
| `runtime.webhooks.enabled` | Enable admission/conversion webhooks | `false` |
| `runtime.webhooks.admission` | Enable admission webhook | `false` |
| `runtime.webhooks.conversion` | Enable conversion webhook | `false` |
| `runtime.webhooks.existingSecret` | TLS secret name (required if enabled) | `""` |
| `runtime.katalog.inline` | Inline Katalog YAML | Starter Katalog |
| `runtime.katalog.existingConfigMap` | Use existing ConfigMap | `""` |
| `runtime.katalog.configMapKey` | Key in ConfigMap | `katalog.yaml` |
| `runtime.katalog.mountPath` | Mount path inside container | `/etc/orkestra/katalog` |

### Control Center Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `controlCenter.enabled` | Deploy the Control Center | `true` |
| `controlCenter.image.repository` | Control Center image | `ghcr.io/orkspace/orkestra-cc` |
| `controlCenter.image.tag` | Image tag | Chart `appVersion` |
| `controlCenter.replicaCount` | Number of replicas | `1` |
| `controlCenter.resources` | Resource limits and requests | See values.yaml |
| `controlCenter.serviceAccount.create` | Create ServiceAccount | `true` |
| `controlCenter.config.orkestraURLs` | List of runtime URLs to monitor | `[]` |
| `controlCenter.config.port` | Control Center port | `8090` |
| `controlCenter.config.refreshInterval` | Katalog refresh interval | `10s` |
| `controlCenter.config.logLevel` | Log level | `info` |
| `controlCenter.service.type` | Service type | `ClusterIP` |
| `controlCenter.service.port` | Service port | `8081` |
| `controlCenter.ingress.enabled` | Enable Ingress | `true` |
| `controlCenter.ingress.hosts` | Ingress hosts | See values.yaml |
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
| `rbac.create` | Create RBAC resources | `true` |
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
| `extraEnv` | Extra environment variables | `[]` |
| `extraVolumes` | Extra volumes | `[]` |
| `extraVolumeMounts` | Extra volume mounts | `[]` |
| `podAnnotations` | Annotations for all pods | `{}` |
| `podLabels` | Labels for all pods | `{}` |

---

## Production Example

```yaml
# production-values.yaml
runtime:
  replicaCount: 3
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
  leaderElection:
    enabled: true
    leaseDuration: 15s
    renewDeadline: 10s
    retryPeriod: 2s
  katalog:
    existingConfigMap: platform-katalog   # managed by GitOps

controlCenter:
  enabled: true
  replicaCount: 2
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

pdb:
  runtime:
    enabled: true
    minAvailable: 2
  controlCenter:
    enabled: true
    minAvailable: 1

networkPolicy:
  enabled: true
  ingressFrom:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: monitoring

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/component
              operator: In
              values:
                - runtime
        topologyKey: kubernetes.io/hostname
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

## RBAC

The chart creates a `ClusterRole` granting Orkestra Runtime the permissions it
needs to watch and manage arbitrary CRDs. The permissions include:

- `get/list/watch` CRDs from any API group (required for dynamic CRD discovery)
- Full CRUD on resources declared in Katalogs (Deployments, Services, etc.)
- Leader election leases in the `coordination.k8s.io` API group
- Event creation for reconciliation tracking

The Control Center receives **no RBAC permissions** — it only reads from the
Runtime API and requires no direct cluster access.

If your security policy does not permit wildcard permissions, set `rbac.create: false` and
create a more restrictive ClusterRole yourself.

---

## Security Defaults

Out of the box, the chart applies:

### Runtime
- `runAsNonRoot: true` with uid/gid 1000
- `readOnlyRootFilesystem: true` (with `/tmp` as an emptyDir)
- `allowPrivilegeEscalation: false`
- `capabilities.drop: [ALL]`
- `seccompProfile: RuntimeDefault`
- Pod anti-affinity spreads replicas across nodes

### Control Center
- `runAsNonRoot: true` with uid/gid 1001
- `readOnlyRootFilesystem: true`
- `allowPrivilegeEscalation: false`
- `capabilities.drop: [ALL]`
- No RBAC permissions (read-only access to Runtime API only)

---

## Troubleshooting

### Control Center cannot connect to Runtime

Check that the runtime service is accessible:

```bash
kubectl get svc -n orkestra-system
kubectl logs -n orkestra-system deployment/orkestra-cc
```

Ensure the `orkestraURLs` in your values point to the correct service names.

### Webhooks not working

Verify TLS secret exists and webhooks are enabled:

```bash
kubectl get secret -n orkestra-system
kubectl logs -n orkestra-system deployment/orkestra-runtime | grep -i webhook
```

### Katalog not loaded

Check the ConfigMap and runtime logs:

```bash
kubectl get configmap -n orkestra-system
kubectl logs -n orkestra-system deployment/orkestra-runtime | grep -i katalog
```

---

## License

[MIT](../../LICENSE)