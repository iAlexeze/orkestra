# Registry Schema

The Registry Schema defines the structure of all resource operations available in a Katalog.  
These fields map directly to the underlying Registry implementations.

This page documents the schema for:

- Deployments  
- Services  
- ConfigMaps  
- Secrets  
- Namespace provisioning  
- Multi‑namespace secret distribution  

---

# Deployments

```yaml
deployments:
  - name: string
    namespace: string
    image: string
    replicas: int
    labels: map[string]string
    annotations: map[string]string
    reconcile: bool
```

!!! tip
    `reconcile: true` enables drift correction — the Deployment is updated on every reconcile.

---

# Services

```yaml
services:
  - name: string
    namespace: string
    port: int
    targetPort: int
    type: ClusterIP|NodePort|LoadBalancer
    reconcile: bool
```

---

# ConfigMaps

```yaml
configMaps:
  - name: string
    namespace: string
    data: map[string]string
    reconcile: bool
```

---

# Secrets

## Create or Update

```yaml
secrets:
  - name: string
    namespace: string
    data: map[string]string
    reconcile: bool
```

## Copy Across Namespaces

```yaml
secrets:
  - name: string
    fromSecret: string
    fromNamespace: string
    toNamespaces:
      - string
    reconcile: bool
```

!!! note
    The Registry ensures all copies stay in sync with the source.

---

# Namespace Provisioning

```yaml
namespaces:
  - name: string
    labels: map[string]string
    annotations: map[string]string
```

---

# Common Fields

All resource types support:

```yaml
reconcile: true|false
```

- `true` → drift correction enabled  
- `false` → create once, never update  
