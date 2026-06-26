# Resource types

Every `onCreate`, `onReconcile`, and `onDelete` block accepts the resource types listed on this page. Resources are applied in the order declared within each type slice.

---

## Supported

These resource types are fully implemented — Orkestra creates, updates, and deletes them on every reconcile.

### Workloads

| Field | Kubernetes kind | API version |
|-------|----------------|-------------|
| `deployments` | `Deployment` | `apps/v1` |
| `replicaSets` | `ReplicaSet` | `apps/v1` |
| `statefulSets` | `StatefulSet` | `apps/v1` |
| `pods` | `Pod` | `v1` |
| `jobs` | `Job` | `batch/v1` |
| `cronJobs` | `CronJob` | `batch/v1` |

### Networking

| Field | Kubernetes kind | API version |
|-------|----------------|-------------|
| `services` | `Service` | `v1` |
| `ingresses` | `Ingress` | `networking.k8s.io/v1` |
| `networkPolicies` | `NetworkPolicy` | `networking.k8s.io/v1` |

### Configuration

| Field | Kubernetes kind | API version |
|-------|----------------|-------------|
| `configMaps` | `ConfigMap` | `v1` |
| `secrets` | `Secret` | `v1` |
| `namespaces` | `Namespace` | `v1` |

### Storage

| Field | Kubernetes kind | API version |
|-------|----------------|-------------|
| `persistentVolumeClaims` | `PersistentVolumeClaim` | `v1` |
| `persistentVolumes` | `PersistentVolume` | `v1` |

### Identity

| Field | Kubernetes kind | API version |
|-------|----------------|-------------|
| `serviceAccounts` | `ServiceAccount` | `v1` |
| `roles` | `Role` | `rbac.authorization.k8s.io/v1` |
| `roleBindings` | `RoleBinding` | `rbac.authorization.k8s.io/v1` |
| `clusterRoles` | `ClusterRole` | `rbac.authorization.k8s.io/v1` |
| `clusterRoleBindings` | `ClusterRoleBinding` | `rbac.authorization.k8s.io/v1` |

### Policy

| Field | Kubernetes kind | API version |
|-------|----------------|-------------|
| `hpa` | `HorizontalPodAutoscaler` | `autoscaling/v2` |
| `pdb` | `PodDisruptionBudget` | `policy/v1` |
| `resourceQuotas` | `ResourceQuota` | `v1` |
| `limitRanges` | `LimitRange` | `v1` |

### Custom

| Field | Description |
|-------|-------------|
| `custom` | Any CRD — applied via the dynamic client. Accepts any YAML structure. |

---

## Not yet supported

The following fields are accepted in the YAML and parsed without error, but Orkestra does not act on them at reconcile time. They are placeholders — declaring them does nothing.

| Field | Kubernetes kind | Notes |
|-------|----------------|-------|
| `daemonSets` | `DaemonSet` | Planned |
| `podTemplates` | `PodTemplate` | Planned |
| `storageClasses` | `StorageClass` | Planned |
| `storageLocations` | Velero `BackupStorageLocation` | Planned |
| `storagePools` | Rook `CephBlockPool` | Planned |
| `storageBackups` | Velero `Backup` | Planned |
| `storageSnapshots` | `VolumeSnapshot` | Planned |
| `storageVolumes` | Longhorn `Volume` | Planned |
| `serviceMonitors` | Prometheus Operator `ServiceMonitor` | Planned — use `custom` in the meantime |
| `priorityClasses` | `PriorityClass` | Planned |
| `priorityLevelConfigurations` | `PriorityLevelConfiguration` | Planned |
| `runtimeClasses` | `RuntimeClass` | Planned |
| `podSecurityPolicies` | `PodSecurityPolicy` (deprecated) | No plan — PSP is removed in Kubernetes 1.25+ |
| `volumes` | Pod volume definitions | Future — injected into pod specs |
| `volumeMounts` | Container volume mounts | Future — injected into container specs |

!!! tip "Using an unsupported type now"
    For any resource type not in the supported list, use `custom:` with the full YAML structure. Orkestra applies it via the dynamic client against the cluster API. See [13-external.md](13-external.md) for details on combining external API calls with custom resources.

---

## ClusterRole and ClusterRoleBinding notes

`clusterRoles` and `clusterRoleBindings` are cluster-scoped. Kubernetes does not allow a namespace-scoped CR to own a cluster-scoped resource via `OwnerReferences` — setting one causes the garbage collector to treat the resource as orphaned and delete it immediately.

Orkestra tracks ownership via the `orkestra.io/owner` label instead. This means cluster-scoped resources are **not automatically deleted** when the CR is deleted. Declare explicit cleanup in `onDelete` if needed:

```yaml
onDelete:
  clusterRoles:
    - name: "{{ .metadata.name }}-cluster-role"
  clusterRoleBindings:
    - name: "{{ .metadata.name }}-crb"
```
