# Orkestra Helm Chart

Declarative Kubernetes Operator Runtime • Security‑First • GitOps‑Native

Orkestra is a **declarative operator runtime**: a platform for building Kubernetes operators using pure YAML. This Helm chart deploys:

- **Orkestra Runtime** — the operator engine  
- **Orkestra Control Center** — multi‑instance observability UI  

```bash
helm repo add orkestra https://ialexeze.github.io/orkestra
helm repo update

helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace
```

---

> [!IMPORTANT]
> **RBAC and ConfigMap are not created by the chart**
>
> Orkestra follows a **security‑first** model:  
> RBAC and ConfigMap **must be generated from your intent**, not shipped as static YAML.
>
> **You MUST generate and apply RBAC (and optionally the Katalog ConfigMap) using the Ork CLI before installing this chart.**

This ensures:

- least‑privilege RBAC  
- zero surprises  
- full GitOps compatibility  
- transparent, reviewable manifests  

---

### 1. Generate RBAC and ConfigMap (required)

From your Katalog file:

```bash
# Minimal ServiceAccounts + RBAC only
ork generate rbac --katalog my-katalog.yaml -o rbac.yaml

# Or full bundle: ServiceAccounts + RBAC + Katalog ConfigMap
ork generate bundle --katalog my-katalog.yaml -o bundle.yaml

# Override namespace if needed
ork generate bundle --katalog my-katalog.yaml -n orkestra-system -o bundle.yaml
```

This produces:

- `ServiceAccount` for runtime  
- `ServiceAccount` for control center  
- `ClusterRole` + `ClusterRoleBinding` (derived from your Katalog)  
- `ConfigMap` containing your Katalog (if using `bundle`)  

---

### 2. Apply the generated manifests

```bash
kubectl apply -f rbac.yaml
# or
kubectl apply -f bundle.yaml
```

Everything is explicit and reviewable.

---

### 3. Configure the Helm chart to use your generated resources

In `values.yaml`:

```yaml
runtime:
  serviceAccount: orkestra
  katalog:
    existingConfigMap: orkestra-katalog
    configMapKey: katalog.yaml

controlCenter:
  serviceAccount: orkestra-cc
```

Then install:

```bash
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --values values.yaml
```

---

### GitOps‑First Workflow (Recommended)

In CI:

```bash
ork validate --katalog my-katalog.yaml
ork template --katalog my-katalog.yaml
ork generate bundle --katalog my-katalog.yaml -n orkestra-system -o orkestra-bundle.yaml

# Commit orkestra-bundle.yaml to your GitOps repo
```

ArgoCD / Flux syncs:

- the bundle (RBAC + ConfigMap)  
- the Helm release  

This is the cleanest, safest, most auditable way to run Orkestra.

---

## Runtime Configuration

```yaml
runtime:
  enabled: true
  ...
  serviceAccount: "orkestra"   # must match generated SA

  config:
  ...
  katalog:
    inline: |        # starter Katalog for demos only
      apiVersion: orkestra.konductor.io/v1Alpha
      kind: Katalog
      metadata:
        name: default-katalog
      spec:
        crds: []
    existingConfigMap: ""        # set to your generated ConfigMap
    configMapKey: katalog.yaml
    mountPath: /etc/orkestra/katalog
```

---

## Control Center Configuration

```yaml
controlCenter:
  enabled: true
  ...
  serviceAccount: "orkestra-cc"   # must match generated SA
```

---

## Shared Settings

- `imagePullSecrets`
- `registry.enabled` / `registry.url`
- `hpa.enabled`
- `networkPolicy.enabled`
- `pdb.runtime` / `pdb.controlCenter`
- `nodeSelector`, `tolerations`, `affinity`
- `extraEnv`, `extraVolumes`, `extraVolumeMounts`
- `podAnnotations`, `podLabels`

---

## Upgrade

```bash
helm repo update
helm upgrade orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --values values.yaml
```

---

## Uninstall

```bash
helm uninstall orkestra --namespace orkestra-system
```

This removes deployments and services.  
CRDs and CRs remain unless manually removed.