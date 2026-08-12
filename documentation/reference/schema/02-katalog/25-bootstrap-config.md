# Bootstrap config

A `cluster-config.yaml` (or `.json`) file passed to `ork clusters bootstrap --config`
describes a set of remote clusters to bootstrap in one command. It is a plain CLI
config file — not a Kubernetes resource, no `apiVersion` or `kind`.

## Structure

```yaml
clusters:
  - name: staging
    context: kind-ork-multi-2

  - name: prod
    context: kind-ork-multi-3
    sa-namespace: restricted-ns
    sa-name: argocd-ork-generated
    rules:
      - apiGroups: ["demo.orkestra.io"]
        resources: ["websites", "websites/status"]
        verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

## Fields

### `clusters[]`

List of cluster entries. At least one entry is required.

### `clusters[].name`

**Required.** Logical name for the cluster. Used as:
- The key in `gateway.clusters` (printed in the Orkestra output snippet)
- The suffix of the credential Secret on the gateway cluster (`orkestra-<name>`)

Must be unique within the file.

### `clusters[].context`

**Required.** The kubeconfig context to use when connecting to the target cluster.

### `clusters[].sa-namespace`

Namespace on the **target** cluster where the ServiceAccount and token Secret are
created. Default: `kube-system`.

The namespace must already exist. Bootstrap does not create it.

### `clusters[].sa-name`

Name for the ServiceAccount (and the derived ClusterRole and ClusterRoleBinding) on
the target cluster. Default: `orkestra-gateway`.

Set this when using bootstrap for a non-Orkestra consumer:

```yaml
sa-name: argocd-ork-generated
```

All target-side resource names derive from this value:

| Resource | Name |
|----------|------|
| ServiceAccount | `<sa-name>` |
| ClusterRole | `<sa-name>` |
| ClusterRoleBinding | `<sa-name>` |
| Token Secret | `<sa-name>-token` |

### `clusters[].rules[]`

ClusterRole rules to apply on the target cluster. Used in the generic (non-Orkestra)
path. When absent, ClusterRole and ClusterRoleBinding are skipped — only the
ServiceAccount and token Secret are created.

Each rule follows the standard Kubernetes `rbacv1.PolicyRule` shape:

```yaml
rules:
  - apiGroups: ["apps"]
    resources: ["deployments", "deployments/status"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

**Valid verbs:** `get`, `list`, `watch`, `create`, `update`, `patch`, `delete`,
`deletecollection`, `use`, `bind`, `escalate`, `impersonate`, `approve`, `sign`, `*`

Unknown verbs are rejected by `ork clusters bootstrap --validate`.

## JSON equivalent

```json
{
  "clusters": [
    {
      "name": "staging",
      "context": "kind-ork-multi-2"
    },
    {
      "name": "prod",
      "context": "kind-ork-multi-3",
      "sa-namespace": "restricted-ns"
    }
  ]
}
```

## See also

- [`ork clusters bootstrap`](../../cli/clusters.md#ork-clusters-bootstrap)
- [Cluster bootstrap concept](../../../concepts/self-service/11-cluster-bootstrap.md)
