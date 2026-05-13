# Orkestra Helm Chart

Declarative Kubernetes Operator Runtime • Security‑First • GitOps‑Native

Orkestra is a **declarative operator runtime**: a platform for building Kubernetes operators using pure YAML. This Helm chart deploys:

- **Orkestra Runtime** — the operator engine  
- **Orkestra Control Center** — multi‑instance observability UI  

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm repo update

helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace
```

---

> [!IMPORTANT]
> ## Before you install: generate RBAC and ConfigMap
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

## Global Settings

Only four keys are shared across all components:

```yaml
global:
  namespace: orkestra-system
  nameOverride: ""
  fullnameOverride: ""
```

`imagePullSecrets` is intentionally **not** global — set it per component so each image can pull from a different registry if needed.

---

## Per-Component Settings

Every tuning knob lives under `runtime:` or `controlCenter:`. Both components expose the same set of optional keys:

```yaml
runtime:            # or controlCenter:
  imagePullSecrets: []

  registry:         # runtime only
    enabled: false
    url: ""

  hpa:
    enabled: false
    minReplicas: 2
    maxReplicas: 5
    targetCPUUtilizationPercentage: 80
    targetMemoryUtilizationPercentage: 80

  networkPolicy:
    enabled: false
    ingressFrom: []

  pdb:
    enabled: true
    minAvailable: 1
    # maxUnavailable: 5

  nodeSelector: {}
  tolerations: []
  affinity: {}
  topologySpreadConstraints: []

  extraEnv: []
  extraEnvFrom: []
  extraVolumes: []
  extraVolumeMounts: []
  podAnnotations: {}
  podLabels: {}
```

> `registry` is runtime-only — the Control Center has no Orkestra registry integration.

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

---

## Security Posture

### Minimal production binary

The Orkestra CLI ships in two forms: a **full CLI** (used by developers and CI) and a **runtime binary** (what runs in your cluster).

The runtime binary is built with the `runtime` tag, which strips every command except `run` and `version`:

| Command | Developer CLI | Runtime binary |
|---------|:---:|:---:|
| `ork run` | ✓ | ✓ |
| `ork version` | ✓ | ✓ |
| `ork generate` | ✓ | — |
| `ork validate` | ✓ | — |
| `ork init` | ✓ | — |
| `ork template` | ✓ | — |
| `ork diff` | ✓ | — |

**Why this matters:** a compromised container cannot use the binary to generate RBAC bundles, enumerate registered CRDs, scaffold new operators, or exfiltrate Katalog definitions. The attack surface of the in-cluster binary is limited to what the operator actually does at runtime. There is no code generation surface to exploit.

### Explicit, auditable RBAC

Orkestra never auto-creates `ServiceAccount`, `ClusterRole`, or `ClusterRoleBinding` resources. You generate them from your Katalog using `ork generate bundle`, review the output, commit it, and apply it explicitly. Nothing is hidden inside the Helm chart. Every permission your operator has is visible in source control before it reaches the cluster.

### Deletion protection

Every resource the Helm chart creates carries the label `orkestra.io/deletion-protection: "true"`. With `security.deletionProtection.enabled: true` in your Katalog, a `ValidatingWebhookConfiguration` intercepts every DELETE request for any resource bearing that label — including the operator's own Deployment, Service, ServiceAccount, and TLS Secret. Accidental or malicious deletion is blocked at the API server before it reaches etcd.

### Webhook self-healing

The webhook controller watches its own `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` objects. If either is deleted — accidentally or by an attacker who gained API server access — Orkestra recreates it within the configured sync interval (default 30 seconds). The protection gap is bounded and logged.

> _Event-driven protection - planned_

### TLS — automatic rotation

Orkestra generates its own TLS certificate for webhook traffic and rotates it automatically. To supply your own certificate authority: `--set tls.certFile=/path/to/tls.crt --set tls.keyFile=/path/to/tls.key`.