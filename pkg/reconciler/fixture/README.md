# pkg/reconciler/fixture

Living integration fixture for the reconciler's resource types.

## Why this exists

The runner functions in `pkg/runner/` apply Kubernetes resources from katalog
templates. There is no meaningful way to unit-test them: the logic that matters
— template rendering, server-side apply, status propagation, condition evaluation
— only works against a real API server. Mocking it tests the mock, not Orkestra.

This fixture is the right vehicle. It declares a `ReconcilerSuite` CRD (cluster-scoped)
and a `ReconcilerProbe` CRD. A single `ReconcilerSuite` CR is the test entry point:
its operator creates a `ReconcilerProbe` via `custom:`, and the probe's operator then
triggers every supported resource block in one reconcile cycle. This exercises the full
`custom:` child chain — creation, reconciliation, and cluster-scoped GC on delete —
alongside all other resource types. Failures surface as a missing resource, a wrong
status field, or an operator crash, all observable without reading code.

## What each block covers

| Block in `katalog.yaml`        | What it exercises                          |
|--------------------------------|--------------------------------------------|
| `custom:` (ReconcilerSuite)    | Cluster-scoped CR creating a child CR      |
| `validation.rules`             | Admission validation                       |
| `mutation.rules`               | Admission mutation (default injection)     |
| `namespaces`                   | Namespace creation                         |
| `secrets` (`once: true`)       | Secret generation, idempotency             |
| `configMaps`                   | ConfigMap with template values             |
| `networkPolicies`              | Inline deny-all NetworkPolicy              |
| `resourceQuotas`               | Inline ResourceQuota with hard limits      |
| `limitRanges`                  | Container default limits                   |
| `clusterRoles`                 | Cluster-scoped RBAC role                   |
| `clusterRoleBindings`          | Cluster-scoped RBAC binding                |
| `serviceAccounts`              | ServiceAccount creation                    |
| `roles`                        | Namespaced RBAC role                       |
| `roleBindings`                 | Namespaced RBAC binding                    |
| `replicaSets`                  | ReplicaSet creation                        |
| `deployments` (main)           | Deployment with env + serviceAccountName   |
| `deployments` with `when:`     | Conditional resource (`tier=premium` only) |
| `services`                     | Service with port mapping                  |
| `jobs`                         | One-shot Job                               |
| `cronJobs`                     | CronJob with schedule                      |
| `statefulSets`                 | StatefulSet creation                       |
| `persistentVolumes`            | Cluster-scoped PV with hostPath            |
| `persistentVolumeClaims`       | PVC creation                               |
| `ingresses`                    | Ingress with host routing                  |
| `hpa`                          | HorizontalPodAutoscaler targeting probe-app |
| `pdb`                          | PodDisruptionBudget targeting probe-app    |
| `pods`                         | Direct Pod creation                        |
| `status.fields`                | Status propagation and template evaluation |

## Running

Run simulate first — it exercises the real reconciler in-memory and is fast:

```bash
ork simulate -f pkg/reconciler/fixture/simulate.yaml
```

For full cluster verification:

```bash
ork e2e -f pkg/reconciler/fixture/e2e.yaml --workers 3
```

`ork e2e` creates the kind cluster, installs Orkestra, applies the CRD and CR,
runs all assertions, and tears down the cluster.

To reuse an existing cluster during iteration:

```bash
ork e2e -f pkg/reconciler/fixture/e2e.yaml --use-current
```

## Adding a new resource type

1. Add a block to the appropriate fixture motif in
   [pkg/children/fixtures/motifs/](../../children/fixtures/motifs/). Name resources
   `{{ .metadata.name }}-<type>` to avoid collisions.
2. Add a row to the table above.
3. Add a `create` op to [simulate.yaml](./simulate.yaml) for the new resource type.
4. Add a `resources:` assertion to the `All resources created` checkpoint in
   [e2e/01-resources.yaml](./e2e/01-resources.yaml).
5. Run `ork e2e -f pkg/reconciler/fixture/simulate.yaml` locally before opening the PR.
