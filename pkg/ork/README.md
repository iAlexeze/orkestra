# pkg/ork

The `ork` package provides the cluster-facing primitives that the Orkestra runtime needs to operate: Helm chart installation, kind cluster management, health checking, and prerequisite detection.

This is the boundary between Orkestra's core packages and the cluster. It does not detect projects, generate Katalogs, parse compose files, or manage deploy state — those concerns live in `ork-doctor`.

## What the package provides

| Area | Functions | Used by |
|------|-----------|---------|
| **Helm** | `InstallOrUpgradeOrkestra`, `BuildControlCenterValues` | `pkg/e2e`, `cmd/cli` |
| **Cluster health** | `OrkestraInstalled`, `RuntimeDeployed`, `CheckRuntimeHealth`, `SyncRuntime`, `FetchRuntimeLogs`, `KatalogChanged` | `pkg/e2e`, `cmd/cli` |
| **Kind** | `EnsureKindCluster`, `DeleteKindCluster` | `pkg/e2e`, `cmd/cli` |
| **Dependencies** | `EnsureDependencies`, `KubectlAvailable`, `HelmAvailable`, `ClusterReachable` | `pkg/e2e`, `cmd/cli` |
| **Constants** | `Orkestra`, `OrkestraRuntime`, `OrkestraNamespace`, `OrkestraChartRepo`, `OrkestraChartName`, `KindClusterName` | everywhere |

## Package structure

| File | Responsibility |
|------|---------------|
| `constants.go` | All `Orkestra*` and chart constants |
| `helm.go` | `InstallOrUpgradeOrkestra`, `BuildControlCenterValues` |
| `health.go` | `CheckRuntimeHealth`, `SyncRuntime`, `FetchRuntimeLogs`, `OrkestraInstalled`, `RuntimeDeployed`, `KatalogChanged` |
| `kind.go` | `EnsureKindCluster`, `DeleteKindCluster`, binary download |
| `deps.go` | `EnsureDependencies`, `KubectlAvailable`, `HelmAvailable`, `ClusterReachable` |

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand cluster health checking and runtime sync | [docs/01-cluster.md](docs/01-cluster.md) |
| Understand Helm chart installation | [docs/02-helm.md](docs/02-helm.md) |
