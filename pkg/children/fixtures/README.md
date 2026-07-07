# pkg/children fixtures

Living integration fixture for the `forEach` expansion and child tracking paths in `pkg/children`.

## Why this exists

The `ExpandForEach*`, `*Names`, and child tracking blocks in `pkg/children/` only work against
a real API server — there is nothing meaningful to unit-test. The expansion is correct when the
right resources appear in the cluster with the right names, and wrong when they are missing or
named incorrectly. This fixture makes that check fast and repeatable without reading code.

## What each motif covers

| Motif | Resource types exercised |
|---|---|
| `motifs/01-namespaced/` | `Namespace`, `NetworkPolicy`, `ResourceQuota`, `LimitRange` |
| `motifs/02-rbac/` | `ClusterRole`, `ClusterRoleBinding`, `Role`, `RoleBinding`, `ServiceAccount` |
| `motifs/03-workloads/` | `Deployment`, `StatefulSet`, `ReplicaSet`, `Pod`, `Job`, `CronJob` |
| `motifs/04-network/` | `Service`, `Ingress`, `HPA`, `PDB` |
| `motifs/05-config/` | `ConfigMap`, `Secret` |
| `motifs/06-storage/` | `PersistentVolume`, `PersistentVolumeClaim` |

All templates use `forEach:` — the fixture exercises the full expand → name → track path for every type.

## Running locally

Requires: `kind`, `kubectl`, `ork` — all installed and on `$PATH`.

Commands must be run from inside this directory (motif paths resolve relative to the working directory).

```bash
cd pkg/children/fixtures

ork run --dev
```

`--dev` creates a local kind cluster named `orkestra-playground` if one does not exist.

After the `acme` Tenant CR reconciles, verify:

```bash
kubectl get networkpolicies,resourcequotas,limitranges -n acme-dev
kubectl get networkpolicies,resourcequotas,limitranges -n acme-staging
kubectl get clusterroles,clusterrolebindings | grep acme
kubectl get deployments,statefulsets,services,hpa,pdb -n default | grep acme
kubectl get pv,pvc -A | grep acme
```

## Adding a new child resource type

When you add a new resource type to `pkg/children/` (new `ExpandForEach*`, GVR, name helper, and tracking block):

1. **Add it to the appropriate motif** in `motifs/` — or create a new motif if it belongs to a new group.
2. **Run `ork validate`** — confirms the schema is correct.
3. **Run `ork run`** — confirms resources are created with the right names.
4. **Add a row to the table above** so the coverage map stays accurate.

This fixture is the final checkpoint before a children PR is ready.
