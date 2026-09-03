# pkg/children

`children` reads, enriches, and exposes the live Kubernetes state of every child resource that a Katalog operator manages. It is the bridge between a Katalog's YAML declarations and the running cluster — translating template source lists into a structured map that status field expressions can navigate.

The single public entry point is `ReadChildren`. Everything else in the package supports it.

```go
children := children.ReadChildren(ctx, kube, obj, resolver, crd)
// → map["deployments"]["my-api"] = { ... live object ... , "_pods": [...], "_warnings": [...] }
```

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand what `ReadChildren` produces and how to use it in status expressions | [docs/01-overview.md](docs/01-overview.md) |
| Understand how child resources are read from the cluster | [docs/02-reading.md](docs/02-reading.md) |
| Understand name resolution and forEach expansion | [docs/03-names-foreach.md](docs/03-names-foreach.md) |
| Understand enrichment layers (`_pods`, `_warnings`, `_endpoints`, `_pv`) | [docs/04-enrichment.md](docs/04-enrichment.md) |
| Understand the built-in kind registry (GVRs, metadata, RBAC detection) | [docs/05-builtins.md](docs/05-builtins.md) |
| Add a new enrichment layer | [docs/04-enrichment.md#adding-a-new-layer](docs/04-enrichment.md#adding-a-new-layer) |

## File layout

| File | Responsibility |
|------|---------------|
| `children.go` | Package doc and `ReadChildren` entry point |
| `read.go` | `readResourceGroup`, `firstValue`, `mergeTemplates` |
| `names.go` | Name resolution (`resolvedChildName`, `*Names` helpers) |
| `foreach.go` | `ExpandForEach*` — template expansion over list fields |
| `foreach_customresources.go` | `ExpandForEachCustomResources` for dynamic resource types |
| `enrich_pods.go` | `_pods` enrichment and pod summary building |
| `enrich_endpoints.go` | `_endpoints` enrichment via EndpointSlice |
| `enrich_warnings.go` | `_warnings` enrichment — workload + pod-level Warning events |
| `enrich_pvc.go` | `_pv` enrichment for PersistentVolumeClaims |
| `enrich_pv.go` | `_pvc` enrichment for PersistentVolumes |
| `enrich_replicasets.go` | `_owner` enrichment for ReplicaSets; `_replicaSets` for Deployments |
| `enrich_cronjobs.go` | `_activeJobs`, `_lastJob`, `_lastSuccessfulJob` for CronJobs |
| `enrich_statefulsets.go` | `_pvcs` enrichment for StatefulSets |
| `enrich_storageclass.go` | `_storageClass` enrichment for PersistentVolumeClaims |
| `enrich_service_pods.go` | `_backingPods` enrichment for Services |
| `enrich_ingress.go` | `_loadBalancerIPs`, `_tlsSecrets` enrichment for Ingresses |
| `enrich_node.go` | `_node` enrichment for Pods |
| `enrich_hpa.go` | `_currentMetrics`, `_scaleTarget` enrichment for HPAs |
| `gvr.go` | All built-in GVR variables and `ChildGVRs()` |
| `builtins.go` | Built-in kind registry — API metadata, RBAC detection, enrichment lookup |
