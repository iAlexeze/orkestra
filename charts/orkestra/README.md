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
> ## Before you install: generate RBAC and Katalog
>
> Orkestra separates the runtime from security configuration. This keeps your permissions explicit, minimal, and reviewable – no hidden RBAC inside the Helm chart.
>
> Because of this design, the chart does **not** create:
>
> - `ServiceAccount`s
> - `ClusterRole`s or `ClusterRoleBinding`s
> - the Katalog `ConfigMap`
> 
> Instead, you generate those resources from your Katalog file using the Ork CLI. Then you apply them before (or together with) the Helm chart.

### How to generate the required resources

From your Katalog file:

```bash
# Minimal RBAC only
ork generate rbac --katalog my-katalog.yaml -o rbac.yaml

# Full bundle: ServiceAccounts + RBAC + Katalog ConfigMap
ork generate bundle --katalog my-katalog.yaml -o bundle.yaml

# Override namespace if needed
ork generate bundle --katalog my-katalog.yaml -n orkestra-system -o bundle.yaml
```

### Apply the generated manifests

```bash
kubectl apply -f rbac.yaml
# or
kubectl apply -f bundle.yaml
```

Now everything is explicit, auditable, and ready for the Helm chart.

### Then configure the Helm chart to use your resources

In `values.yaml`:

```yaml
runtime:
  serviceAccount: orkestra
  katalog:
    existingConfigMap: orkestra-katalog   # if you used --bundle
    configMapKey: katalog.yaml

controlCenter:
  serviceAccount: orkestra-cc
```

Install the chart normally:

```bash
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --values values.yaml
```

> [!TIP]
> **Why this workflow?**  
> It guarantees least‑privilege RBAC, no surprises, and full GitOps compatibility. 
> The generated YAML can be reviewed, versioned, and audited – exactly as infrastructure should be.

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