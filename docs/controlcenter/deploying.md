# Deploying the Orkestra Control Center

The Orkestra Control Center can be deployed in three ways depending on your environment and operational requirements.  
It is included by default in all Orkestra runtime deployments unless explicitly disabled.

---

## 1. Local Development (Quick Start)

The fastest way to run the Control Center is directly from the `ork` CLI:

```bash
ork control start
```

Common options:

```bash
# Custom port
ork control start --port 9090

# Multiple Orkestra runtimes
ork control start --urls "http://cluster1:8080,http://cluster2:8080"

# Start with no runtimes (add them later from the UI)
ork control start --ignore-default
```

This method is ideal for:

- Local development  
- Testing Katalogs  
- Debugging CRDs  
- Multi‑runtime experimentation  

The Control Center will be available at:

```
http://localhost:<port>
```

---

## 2. Kubernetes Deployment (YAML)

The Control Center is included in the standard Orkestra installation bundle:

```bash
kubectl apply -f orkestra/install.yaml
```

This deploys:

- The Orkestra runtime  
- The Control Center  
- All required RBAC, services, and configuration  

You can expose the Control Center using a simple ingress:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: orkestra
  namespace: orkestra-system
  annotations:
    nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
spec:
  ingressClassName: nginx
  rules:
  - host: o.orkestra.sh
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: orkestra
            port:
              number: 8080
```

This method is suitable for:

- Staging environments  
- Internal clusters  
- Lightweight deployments  

---

## 3. Helm Chart (Recommended for Production)

The recommended production deployment method is Helm.  
The Control Center is deployed **automatically** with the Orkestra runtime unless disabled:

```yaml
controlCenter:
  enabled: true
```

### Example Helm installation

```bash
helm repo add orkestra https://charts.orkestra.sh
helm install ork orkestra/orkestra \
  --set controlCenter.enabled=true \
  --set controlCenter.config.port=8081 \
  --set controlCenter.config.refreshInterval=10s \
  --set controlCenter.config.orkestraURLs="{http://orkestra-runtime-1:8080,http://orkestra-runtime-2:8080}" \
```

!!! important "Security Note"

    Set ADMIN_UNSERNAME, ADMIN_PASSWORD and SESSION_SECRET as environment variables using which is the sam as kubernetes `envFrom`.

    Example:
    ```bash
    extraEnvFrom:
      - secretRef:
          name: orkestra-secret
        defaultMode: 420
        items:
          - key: ADMIN_USERNAME
            path: ADMIN_USERNAME
          - key: ADMIN_PASSWORD
            path: ADMIN_PASSWORD
          - key: SESSION_SECRET
            path: SESSION_SECRET
    ```



### Relevant Helm Values (Control Center)

```yaml
controlCenter:
  enabled: true

  image:
    repository: ghcr.io/orkspace/orkestra-cc
    tag: ""
    pullPolicy: IfNotPresent

  replicaCount: 1

  config:
    orkestraURLs:
      - http://orkestra-runtime-1:8080
      - http://orkestra-runtime-2:8080
    port: 8081
    refreshInterval: 10s
    logLevel: info

  service:
    type: ClusterIP
    port: 8081

  ingress:
    enabled: true
    hosts:
      - host: orkestra-cc.local
        paths:
          - path: /
            pathType: Prefix
```

### Why Helm is recommended

- Versioned, repeatable deployments  
- Built‑in configuration for Control Center and runtime  
- Ingress, TLS, and resource limits included  
- Easy upgrades and rollbacks  
- Works across all Kubernetes distributions  

---

## Automatic Deployment with Orkestra Runtime

When deploying Orkestra via Helm, the Control Center is enabled by default:

```yaml
controlCenter:
  enabled: true
```

To disable it:

```yaml
controlCenter:
  enabled: false
```

This allows:

- Single‑binary deployments (runtime + control center)  
- Consistent rollout across clusters  
- Unified configuration via Helm  

---

## Summary

| Method | Use Case | Notes |
|--------|----------|-------|
| `ork control start` | Local development | Fastest way to start; supports multi‑runtime |
| `kubectl apply -f install.yaml` | Simple cluster deployment | Control Center included automatically |
| Helm Chart | Production | Recommended; full configuration, ingress, TLS, scaling |
