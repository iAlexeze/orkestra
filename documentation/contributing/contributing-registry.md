# Contributing to pkg/orkestra-registry

The registry is the library of built-in Kubernetes resource handlers Orkestra knows how to create, update, and delete on behalf of an operator. Each resource type lives in its own subdirectory.

---

## What exists

| Directory | Resource | Status |
|-----------|----------|--------|
| `deployments/` | `apps/v1 Deployment` | Complete |
| `services/` | `v1 Service` | Complete (minor TODOs in edge cases) |
| `configmaps/` | `v1 ConfigMap` | Complete |
| `secrets/` | `v1 Secret` | Complete |
| `statefulsets/` | `apps/v1 StatefulSet` | Complete |
| `jobs/` | `batch/v1 Job` | Complete |
| `cronjobs/` | `batch/v1 CronJob` | Complete |
| `replicasets/` | `apps/v1 ReplicaSet` | Complete |
| `ingresses/` | `networking.k8s.io/v1 Ingress` | Implemented — needs tests |
| `hpas/` | `autoscaling/v2 HPA` | Implemented — needs tests |
| `pdbs/` | `policy/v1 PodDisruptionBudget` | Implemented — needs tests |
| `pvcs/` | `v1 PersistentVolumeClaim` | Implemented — needs tests |
| `pvs/` | `v1 PersistentVolume` | Implemented — needs tests |
| `namespaces/` | `v1 Namespace` | Implemented — cleanup on delete not yet working |
| `roles/` | `rbac/v1 Role` | Implemented — needs tests |
| `rolebindings/` | `rbac/v1 RoleBinding` | Implemented — needs tests |
| `serviceaccounts/` | `v1 ServiceAccount` | Implemented — needs tests |
| `customresources/` | Dynamic CR via `dynamic` client | Implemented |
| `pods/` | `v1 Pod` | Implemented |

---

## What needs implementing

The following resource types are declared in `pkg/types/types.go` as `PlaceholderSource` — they are accepted in YAML but not yet executed. Implementing one means building the full handler package.

**Workloads:**
- `DaemonSets`
- `PodTemplates`

**RBAC:**
- `ClusterRoles`
- `ClusterRoleBindings`

**Scheduling:**
- `PriorityClasses`
- `PriorityLevelConfigurations`
- `LimitRanges`
- `ResourceQuotas`
- `RuntimeClasses`

**Networking:**
- `NetworkPolicies`

**Storage:**
- `StorageClasses`
- `StorageLocations`
- `StoragePools`
- `StorageBackups`
- `StorageSnapshots`
- `StorageVolumes`

**Monitoring:**
- `ServiceMonitors` (Prometheus Operator CRD)

**Other:**
- `Volumes` (injected into pod specs)
- `VolumeMounts` (injected into pod specs)
- `PodSecurityPolicies`

When you implement one, remove it from the `PlaceholderSource` block in `pkg/types/types.go` and replace it with a proper typed slice.

---

## How to add a resource type

Follow the pattern of any complete resource (e.g., `deployments/`):

### 1. Create the directory

```
pkg/orkestra-registry/<resourcename>/
  <resourcename>.go   — Create, Update, Delete, Resolve functions
  types.go            — ResolvedSpec struct
```

### 2. Implement four functions

```go
func Create(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedSpec) error
func Update(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedSpec) error
func Delete(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedSpec) error
```

All three are idempotent. `Create` does nothing if the resource exists. `Delete` does nothing if it does not.

### 3. Add a resolver in `template/resolver.go`

```go
func (r *Resolver) ResolveMyResourceTemplate(src *orktypes.MyResourceSource) (*myresource.ResolvedSpec, error)
```

Use `r.Resolve(expr)` for any field that may contain a `{{ .spec.something }}` expression.

### 4. Wire the runner in `pkg/reconciler/generic.go`

Find `runResources` and add your resource type alongside the existing ones.

### 5. Write tests

Every resource needs tests covering:
- `Create` — resource created; already exists (no-op)
- `Update` — drift detected and applied; no drift (no-op)
- `Delete` — resource exists (deleted); does not exist (no-op)

Use the `pkg/simulate` harness and a fake clientset (`kubeclient.NewFakeClientset()`).

---

## Code conventions

- Use `kubeclient.KubeClient` for all API calls — never raw client-go.
- Return wrapped errors: `fmt.Errorf("creating daemonset: %w", err)`.
- Owner references are set via `common.SetOwnerReference(owner, resource)`.
- Labels are merged with `labels.ManagedLabels(owner)`.
- Namespace resolution uses `common.ResolveNamespace(owner, spec.Namespace)`.
