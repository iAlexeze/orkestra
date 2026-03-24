# Orkestra Helm Chart

Deploy Orkestra — the declarative Kubernetes operator runtime — using Helm.

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

## Install

### Quick install with defaults

```bash
helm repo add orkestra https://ialexeze.github.io/orkestra
helm repo update

helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace
```

### Install with your own Katalog

The Katalog is mounted as a ConfigMap volume. Pass it inline in your values:

```bash
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --values my-values.yaml
```

`my-values.yaml`:

```yaml
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

### Use an existing ConfigMap for the Katalog

If you manage the Katalog separately (e.g. with Flux or ArgoCD):

```yaml
katalog:
  existingConfigMap: my-katalog-configmap
  configMapKey: katalog.yaml
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

Uninstalling removes the Orkestra Deployment and all chart resources.
CRs and CRDs that Orkestra was managing are **not** deleted — they
remain in the cluster and must be cleaned up separately if desired.

---

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Container image repository | `ghcr.io/ialexeze/orkestra` |
| `image.tag` | Image tag | Chart `appVersion` |
| `replicaCount` | Number of replicas (only one leads) | `2` |
| `katalog.inline` | Katalog YAML embedded in a ConfigMap | starter Katalog |
| `katalog.existingConfigMap` | Use an existing ConfigMap instead | `""` |
| `config.logLevel` | Log level: debug, info, warn, error | `info` |
| `config.healthPort` | Port for health, metrics, and Katalog API | `8080` |
| `config.defaultWorkers` | Default reconcile workers per CRD | `2` |
| `config.defaultResync` | Default resync interval | `30s` |
| `leaderElection.enabled` | Enable leader election for HA | `true` |
| `leaderElection.leaseDuration` | Leader lease duration | `15s` |
| `service.type` | Service type | `ClusterIP` |
| `ingress.enabled` | Enable Ingress for the health API | `false` |
| `hpa.enabled` | Enable HorizontalPodAutoscaler | `false` |
| `pdb.enabled` | Enable PodDisruptionBudget | `true` |
| `pdb.minAvailable` | Minimum available pods during disruptions | `1` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `networkPolicy.enabled` | Enable NetworkPolicy | `false` |
| `rbac.create` | Create ClusterRole and ClusterRoleBinding | `true` |
| `serviceAccount.create` | Create ServiceAccount | `true` |

Full values reference: [`values.yaml`](./values.yaml)

---

## Production example

```yaml
# production-values.yaml
replicaCount: 3

katalog:
  existingConfigMap: platform-katalog   # managed by GitOps

config:
  logLevel: warn
  defaultWorkers: 4
  defaultResync: 1m

leaderElection:
  enabled: true
  leaseDuration: 15s
  renewDeadline: 10s
  retryPeriod: 2s

resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 1Gi

pdb:
  enabled: true
  minAvailable: 2

hpa:
  enabled: true
  minReplicas: 3
  maxReplicas: 8
  targetCPUUtilizationPercentage: 70

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: orkestra.platform.myorg.io
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: orkestra-tls
      hosts:
        - orkestra.platform.myorg.io

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
            - key: app.kubernetes.io/name
              operator: In
              values:
                - orkestra
        topologyKey: kubernetes.io/hostname
```

---

## Observability

After installing, Orkestra exposes:

```bash
# Health endpoints
kubectl port-forward svc/orkestra 8080:8080 -n orkestra-system

curl localhost:8080/health          # liveness
curl localhost:8080/ready           # readiness
curl localhost:8080/metrics          # Prometheus metrics
curl localhost:8080/katalog | jq     # all CRDs
```

**Prometheus scrape config:**

```yaml
- job_name: orkestra
  static_configs:
    - targets: ['orkestra.orkestra-system.svc.cluster.local:8080']
  metrics_path: /metrics
```

---

## RBAC

The chart creates a `ClusterRole` granting Orkestra the permissions it
needs to watch and manage arbitrary CRDs. The permissions are scoped to
what the OrkestraRegistry uses: Deployments, Services, Secrets, ConfigMaps,
ServiceAccounts, Jobs, CronJobs, Pods, Events, and Leases.

The wildcard `get/list/watch` on `*` groups is required because Orkestra
watches CRDs from any API group declared in the Katalog. If your security
policy does not permit wildcard permissions, set `rbac.create: false` and
create a more restrictive ClusterRole yourself.

---

## Security defaults

Out of the box the chart applies:

- `runAsNonRoot: true` with uid/gid 1000
- `readOnlyRootFilesystem: true` (with `/tmp` as an emptyDir)
- `allowPrivilegeEscalation: false`
- `capabilities.drop: [ALL]`
- `seccompProfile: RuntimeDefault`
- Pod anti-affinity spreads replicas across nodes

---

## License

[MIT](../../LICENSE)