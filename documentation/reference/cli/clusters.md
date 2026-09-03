# ork clusters

Manage and verify gateway cluster routing configuration.

## Commands

| Command | Description |
|---------|-------------|
| [`ork clusters`](#ork-clusters-1) | List all registered gateway clusters |
| [`validate`](#ork-clusters-validate) | Validate `gateway.clusters` configuration offline |
| [`check`](#ork-clusters-check) | Connect to each cluster and verify CRD presence |
| [`bootstrap`](#ork-clusters-bootstrap) | Provision least-privilege access on a target cluster |

---

## `ork clusters`

List all clusters registered in `gateway.clusters`, showing the endpoint and credential form for each.

```bash
ork clusters
```

### Output

```text
  gateway.clusters (2 registered)

  →  prod
     https://prod.internal:6443
     kubeconfig  secretRef: prod-credentials[kubeconfig]

  →  staging
     https://staging.internal:6443
     token + CA  tokenRef: staging-sa-token[token]
```

---

## `ork clusters validate`

Validate the `gateway.clusters` block offline — no cluster connections required.

Checks each entry for structural validity: endpoint required, exactly one credential form, all required secret ref fields present. Also checks that every static `serve.cluster` and `target.cluster` reference resolves to a registered cluster name. Template expressions are validated against the full user-defined funcMap.

```bash
ork clusters validate
ork clusters validate --full
```

### Flags

| Flag | Description |
|------|-------------|
| `--full` | Show which CRDs route to each cluster. |
| `--file` | Path to a specific katalog file. Defaults to `katalog.yaml` in the current directory. |

### Examples

```bash
# Validate cluster configuration
ork clusters validate

# Show CRD routing per cluster
ork clusters validate --full
```

### Output

```text
⎈  ork clusters validate

  gateway.clusters (2 registered)

  →  prod
     ✓ endpoint: https://prod.internal:6443
     ✓ credential: kubeconfig (secretRef: prod-credentials[kubeconfig])
     ○ routes: widget.serve.target.prod-only.cluster

  →  staging
     ✓ endpoint: https://staging.internal:6443
     ✓ credential: bearer token + CA (tokenRef: staging-sa-token[token], caRef: staging-ca[ca.crt])
     ○ routes: widget.serve.cluster

────────────────────────────────────────────────────────────
✓ 2 cluster(s) valid
```

---

## `ork clusters check`

Go online: read each cluster's credential Secret from the management cluster, connect to the remote cluster, and verify the katalog's CRDs are installed.

```bash
ork clusters check
ork clusters check --clusters prod
ork clusters check --context my-mgmt-context
```

### Flags

| Flag | Description |
|------|-------------|
| `--context` | kubectl context for reading credential Secrets from the management cluster. Defaults to the current context. |
| `--clusters` | Comma-separated list of cluster names to check. Defaults to all registered clusters. |
| `--config` | Path to a credentials file emitted by `ork clusters bootstrap --out`. Skips the katalog — checks connectivity only. |
| `--file` | Path to a specific katalog file. |

### Examples

```bash
# Check all clusters using the current context
ork clusters check

# Check a single cluster
ork clusters check --clusters prod

# Use a specific kubectl context to reach the management cluster
ork clusters check --context my-mgmt-context

# Check a subset using a specific management context
ork clusters check --clusters prod,staging --context my-mgmt-context

# Check connectivity using credentials written by bootstrap --out (no katalog needed)
ork clusters check --config clusters-creds.yaml
```

### Output

```text
⎈  ork clusters check

  →  prod  https://prod.internal:6443
     ✓ credentials: read ok
     ✓ connect: reachable
     ✓ crd Widget: installed

  →  staging  https://staging.internal:6443
     ✓ credentials: read ok
     ✓ connect: reachable
     ✓ crd Widget: installed

────────────────────────────────────────────────────────────
✓ all clusters reachable
```

---

## `ork clusters bootstrap`

Provision the access the gateway needs on a target cluster. Connects to the target
cluster, creates a ServiceAccount and ClusterRole scoped to the katalog's
serve-enabled CRDs, and stores the resulting credentials as a Secret in the gateway
cluster. Prints a `gateway.clusters` YAML block ready to paste into the katalog.

The tool also works without a katalog — for ArgoCD, Flux, or any other system that
needs a least-privilege ServiceAccount and token on a remote cluster.

```bash
# Single cluster (Orkestra)
ork clusters bootstrap --context <target-context> --name <cluster-name>

# Multiple clusters from a config file
ork clusters bootstrap --config cluster-config.yaml

# Validate a config file without connecting to any cluster
ork clusters bootstrap --validate cluster-config.yaml
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--context` | — | kubectl context for the **target** cluster |
| `--name` | — | Name for this cluster in `gateway.clusters` |
| `--namespace` | `default` | Namespace in the **gateway** cluster for the credential Secret |
| `--sa-namespace` | `kube-system` | Namespace on the **target** cluster for the ServiceAccount and token Secret |
| `--config` | — | Path to a `cluster-config.yaml` or `.json` to bootstrap multiple clusters |
| `--out` | `-o` | Write cluster credentials to this file after bootstrap (`gateway.clusters` format; use with `check --config`) |
| `--validate` | — | Validate a config file without connecting to any cluster |
| `--dry-run` | `false` | Print what would be applied without making any changes |
| `--emit-rbac` | `false` | Print only the ClusterRole YAML for review, then exit |
| `--file` | — | Path to a katalog file (required for Orkestra path, not needed with `--config`) |

### What it creates

**On the target cluster** (`--context`):

| Resource | Name | Namespace |
|----------|------|-----------|
| ServiceAccount | `orkestra-gateway` | `kube-system` |
| ClusterRole | `orkestra-gateway` | — |
| ClusterRoleBinding | `orkestra-gateway` | — |
| Secret (SA token) | `orkestra-gateway-token` | `kube-system` |

The ClusterRole is scoped to exactly the serve-enabled CRDs in the katalog (Orkestra
path) or the `rules:` field in the config file (generic path). When neither is
present, only the ServiceAccount and token Secret are created.

**On the gateway cluster** (current context):

| Resource | Name | Namespace |
|----------|------|-----------|
| Secret (token + CA) | `orkestra-<name>` | `--namespace` |

### Single cluster example

```bash
ork clusters bootstrap --context kind-prod --name prod
```

```text
⎈  ork clusters bootstrap
  → cluster name:   prod
  → target context: kind-prod
  → namespace:      default
  → sa-namespace:   kube-system

→  target cluster (kind-prod)
   ✓ ServiceAccount kube-system/orkestra-gateway: created
   ✓ ClusterRole orkestra-gateway: created
   ✓ ClusterRoleBinding orkestra-gateway: created
   ✓ Secret kube-system/orkestra-gateway-token: token ready

→  gateway cluster (current context)
   ✓ Secret default/orkestra-prod: created

⎈  Add to your katalog:

gateway:
  clusters:
    prod:
      endpoint: https://127.0.0.1:6443
      tokenRef:
        name: orkestra-prod
        namespace: default
        key: token
      caRef:
        name: orkestra-prod
        namespace: default
        key: ca.crt
```

### Config file (multiple clusters)

```yaml
# cluster-config.yaml
clusters:
  - name: staging
    context: kind-ork-multi-2

  - name: prod
    context: kind-ork-multi-3
    sa-namespace: restricted-ns      # optional, default: kube-system
    sa-name: argocd-ork-generated    # optional, default: orkestra-gateway
    rules:                           # optional — generic (non-Orkestra) path
      - apiGroups: ["apps"]
        resources: ["deployments", "deployments/status"]
        verbs: ["get", "list", "create", "update", "patch", "delete"]
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `name` | yes | — | Logical cluster name; used in `gateway.clusters` and Secret names |
| `context` | yes | — | kubeconfig context for the target cluster |
| `sa-namespace` | no | `kube-system` | Namespace for SA and token Secret on the target cluster |
| `sa-name` | no | `orkestra-gateway` | SA name override for non-Orkestra consumers |
| `rules` | no | — | ClusterRole rules; absent → SA + token only, no ClusterRole |

YAML and JSON are both accepted.

```bash
# Bootstrap all clusters and emit a credentials file
ork clusters bootstrap --config cluster-config.yaml --out clusters-creds.yaml

# Then verify connectivity without a katalog
ork clusters check --config clusters-creds.yaml
```

### Validate only

```bash
ork clusters bootstrap --validate cluster-config.yaml
```

```text
✓ bootstrap config valid (2 clusters)
  staging  →  kind-ork-multi-2
  prod     →  kind-ork-multi-3
```

No cluster connections are made. Invalid verbs and missing required fields are caught here.

### Re-running bootstrap

Bootstrap is idempotent. Re-running updates the ClusterRole to reflect the current
katalog (useful after adding a new CRD to `serve:`) and reuses the existing token
Secret.

---

## Where to go next

- [`gateway.clusters` schema](../schema/02-katalog/24-gateway-clusters.md)
- [Multi-cluster routing](../../concepts/self-service/10-multi-cluster-routing.md)
- [`ork create cluster`](create.md) — create local kind clusters for testing
