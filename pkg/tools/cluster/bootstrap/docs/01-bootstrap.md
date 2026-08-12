# Bootstrap flow and config file format

## Motivation

Bootstrapping a remote cluster — creating a scoped ServiceAccount, extracting its
token, and storing the credential somewhere the gateway can read it — is something
every platform engineer has done dozens of times in Terraform or bash. This package
automates it from a single point. It was built to support Orkestra's multi-cluster
gateway, but it works for any tool that needs the same pattern.

## What happens during a bootstrap run

```
Target cluster                          Gateway cluster
──────────────────────────────          ───────────────────────────
ServiceAccount  (kube-system)
ClusterRole     (scoped to CRDs)
ClusterRoleBinding
Secret          (long-lived token)  →   Secret (token + CA cert)
```

1. Connect to the target cluster using the kubeconfig context in `ClusterEntry.Context`.
2. Create (or update) a `ServiceAccount` in `ClusterEntry.SANamespace`.
3. Create (or update) a `ClusterRole` with rules derived from the katalog's
   serve-enabled CRDs (Orkestra path), or declared explicitly as `rules:` in the
   config file (generic path). If neither is provided, this step is skipped — only
   the SA and token are provisioned.
4. Create a `ClusterRoleBinding` tying the role to the SA.
5. Create a `kubernetes.io/service-account-token` Secret and wait for the token
   controller to populate it.
6. Connect to the current context (gateway cluster).
7. Write an Opaque Secret containing the token and CA cert so the gateway can
   authenticate to the target.

## Config file format

A config file bootstraps multiple clusters in one command:

```yaml
# cluster-config.yaml
clusters:
  - name: staging
    context: kind-ork-multi-2

  - name: prod
    context: kind-ork-multi-3
    sa-namespace: restricted-ns      # optional, default: kube-system
    sa-name: argocd-ork-generated    # optional, default: orkestra-gateway
```

JSON is also accepted:

```json
{
  "clusters": [
    { "name": "staging", "context": "kind-ork-multi-2" },
    { "name": "prod",    "context": "kind-ork-multi-3", "sa-namespace": "restricted-ns" }
  ]
}
```

### Fields

| Field          | Required | Default             | Description |
|----------------|----------|---------------------|-------------|
| `name`         | yes      | —                   | Logical name; used in `gateway.clusters` and the credential Secret name |
| `context`      | yes      | —                   | kubeconfig context for the target cluster |
| `sa-namespace` | no       | `kube-system`       | Namespace for the SA and token Secret on the target cluster |
| `sa-name`      | no       | `orkestra-gateway`  | SA name; override for non-Orkestra consumers |

`sa-namespace` is not created if it doesn't exist — the bootstrap will fail with
a clear error from Kubernetes. Create it beforehand if using a custom namespace.

## Output modes

### Orkestra (default)

Prints the `gateway.clusters` YAML block to paste into your katalog:

```
⎈  Add to your katalog:

gateway:
  clusters:
    staging:
      endpoint: https://127.0.0.1:6444
      tokenRef:
        name: orkestra-staging
        namespace: default
        key: token
      caRef:
        name: orkestra-staging
        namespace: default
        key: ca.crt
```

### Non-Orkestra (`--no-hint`)

Suppresses the Orkestra snippet and prints only the Secrets that were created:

```
⎈  Credentials stored:

  Secret default/orkestra-staging
    token key: token
    CA key:    ca.crt
```

## Dry run

`--dry-run` logs every resource that would be created without touching any cluster:

```
   ✓ dry-run: ServiceAccount kube-system/orkestra-gateway
   ✓ dry-run: ClusterRole orkestra-gateway
   ✓ dry-run: ClusterRoleBinding orkestra-gateway
   ✓ dry-run: Secret kube-system/orkestra-gateway-token (token)
```

## Validate only

`--validate <file>` checks the config file without connecting to any cluster:

```
✓ bootstrap config valid (2 clusters)
  staging  →  kind-ork-multi-2
  prod     →  kind-ork-multi-3
```
