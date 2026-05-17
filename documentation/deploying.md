# Deploying Orkestra

Run Orkestra in a Kubernetes cluster using the official Helm chart.

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm repo update
```

---

## Install

```bash
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --set runtime.katalog.existingConfigMap=my-katalog-configmap
```

Your Katalog must be stored in a ConfigMap and referenced via `runtime.katalog.existingConfigMap`.

---

## Control Center

The Control Center is deployed alongside the runtime by default. It connects to the runtime automatically.

```yaml
# values.yaml
controlCenter:
  enabled: true          # default: true
  config:
    port: 8081
    refreshInterval: 10s
    orkestraURLs:
      - http://orkestra-runtime:8080
```

To disable it:

```yaml
controlCenter:
  enabled: false
```

### Authentication

Set these via a Kubernetes Secret and `envFrom` in your values:

```yaml
extraEnvFrom:
  - secretRef:
      name: orkestra-secret
```

The secret should contain:

| Key | Description |
|-----|-------------|
| `ADMIN_USERNAME` | Login username |
| `ADMIN_PASSWORD` | Login password |
| `SESSION_SECRET` | Cookie signing secret (use a random value in production) |

---

## Production Values

```yaml
runtime:
  replicaCount: 2          # enable leader election
  katalog:
    existingConfigMap: my-katalog-configmap

controlCenter:
  enabled: true
  replicaCount: 1
  config:
    port: 8081
    refreshInterval: 10s
    orkestraURLs:
      - http://orkestra-runtime:8080
  ingress:
    enabled: true
    className: nginx
    hosts:
      - host: orkestra.myorg.internal
        paths:
          - path: /
            pathType: Prefix
  resources:
    requests:
      cpu: 50m
      memory: 64Mi
    limits:
      cpu: 200m
      memory: 256Mi
```

---

## Upgrade

```bash
helm upgrade orkestra orkestra/orkestra \
  --namespace orkestra-system \
  -f values.yaml
```
