# ClusterRole rules and the katalog

## Orkestra path (with `-f katalog.yaml`)

When a katalog is provided, `buildClusterRoleRules` derives the ClusterRole rules
directly from it — one `PolicyRule` per API group, covering every serve-enabled
CRD and its `/status` subresource:

```yaml
rules:
  - apiGroups: ["demo.orkestra.io"]
    resources: ["websites", "websites/status", "functions", "functions/status"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

This is the least-privilege promise: the role is scoped to exactly what the
katalog declares. No wildcards, no cluster-admin.

## Generic path (no katalog, `--config` only)

When bootstrapping without a katalog, no ClusterRole or ClusterRoleBinding is
created. Only the ServiceAccount and token Secret are provisioned. The caller is
responsible for applying appropriate RBAC on the target cluster separately.

Use `--emit-rbac -f katalog.yaml` to preview the ClusterRole YAML the Orkestra
path would generate, then apply or adapt it manually:

```bash
ork clusters bootstrap --emit-rbac -f katalog.yaml --name staging --context kind-staging
```

## Resource name derivation

All resource names are derived from `ClusterEntry` via `names.go`:

| Resource | Name source | Default |
|----------|-------------|---------|
| ServiceAccount | `SAName(entry)` | `orkestra-gateway` |
| ClusterRole | `ClusterRoleName(entry)` — always matches SA | `orkestra-gateway` |
| ClusterRoleBinding | `CRBName(entry)` — always matches SA | `orkestra-gateway` |
| Token Secret (target) | `TokenSecretName(entry)` | `orkestra-gateway-token` |
| Credential Secret (gateway) | `GatewaySecretName(entry)` | `orkestra-<name>` |

Setting `sa-name: argocd-ork-generated` in the config file changes all four
target-side names consistently.
