# pkg/tools/cluster/bootstrap

Provisions least-privilege gateway access on remote clusters.

Given a target cluster's kubeconfig context, `Cluster()` creates a ServiceAccount,
ClusterRole scoped to the katalog's serve-enabled CRDs, ClusterRoleBinding, and a
long-lived token Secret on the target. It then stores the credential in the gateway
cluster so the gateway can route applies there.

The package is generic: Orkestra is the primary consumer, but any tool that needs a
scoped ServiceAccount + token on a remote cluster can use it by setting
`ClusterEntry.SAName` to something other than the default.

## Package structure

| File | Responsibility |
|------|----------------|
| `config.go` | `ClusterEntry`, `ConfigFile`, `LoadConfig()`, `ValidateConfig()` |
| `names.go` | Resource name derivation — SA, ClusterRole, CRB, Secret names |
| `bootstrap.go` | `Cluster()`, `Result`, `RunOptions` |
| `apply.go` | Kubernetes resource helpers — SA, ClusterRole, CRB, token Secret, credential Secret |
| `helper.go` | `waitForToken()` |

## Testing

Pure function tests (no cluster needed):

```bash
go test ./pkg/tools/cluster/bootstrap/...
```

For end-to-end testing against real clusters, see [fixture/README.md](fixture/README.md).

## Developer documentation

| I want to… | Go to |
|------------|-------|
| Understand the full bootstrap flow and config file format | [docs/01-bootstrap.md](docs/01-bootstrap.md) |
| Understand how the ClusterRole rules are derived from the katalog | [docs/02-rbac.md](docs/02-rbac.md) |
