# pkg/runtime/runners

Resource runners — one file per Kubernetes resource type.

Each runner takes a resolved list of template sources and applies them to the cluster: creating, updating, or deleting the corresponding Kubernetes objects according to the CR's declared state.

## What lives here

| File | Resource |
|------|----------|
| `secrets.go` | Secret (includes `once:`, `rotateAfter:`, `tls:`, `toNamespaces:`) |
| `configmaps.go` | ConfigMap (includes `toNamespaces:`, `fromConfigMap:`) |
| `serviceaccounts.go` | ServiceAccount |
| `roles.go` | Role |
| `rolebindings.go` | RoleBinding |
| `deployments.go` | Deployment |
| `statefulsets.go` | StatefulSet |
| `replicasets.go` | ReplicaSet |
| `services.go` | Service |
| `ingresses.go` | Ingress |
| `jobs.go` | Job |
| `cronjobs.go` | CronJob |
| `pods.go` | Pod |
| `pvcs.go` | PersistentVolumeClaim |
| `pvs.go` | PersistentVolume |
| `hpas.go` | HorizontalPodAutoscaler |
| `pdbs.go` | PodDisruptionBudget |
| `namespaces.go` | Namespace (create-only; no drift correction) |
| `secrets_once.go` | Helper: `once:` guard, `IsNotFoundErr` |
| `secret_tls.go` | Helper: TLS secret rotation |

## What does NOT live here

Runners that are specific to the reconciler's dispatch logic stay in `pkg/runtime/reconciler/`:

- `run_template_reconcile.go` — the dispatcher that calls each runner in sequence
- `run_admission.go`, `run_validations.go`, `run_mutations.go` — admission/validation/mutation pipeline
- `run_status.go` — status field writing
- `run_namespace_guard.go` — namespace allow/restrict enforcement
- `run_delete_ordered.go` — sequential staged deletion
- `run_external.go`, `run_git.go`, `run_docker.go` — external integration runners
- `run_providers.go`, `run_cross.go`, `run_customresource.go` — provider and cross-CRD runners

## Adding a new resource type

See [docs/01-runner-contract.md](docs/01-runner-contract.md) for the canonical shape every runner must follow.

For the full end-to-end walkthrough (types → resource package → resolver → runner → dispatcher), see [pkg/runtime/reconciler/docs/07-adding-a-resource.md](../reconciler/docs/07-adding-a-resource.md).
