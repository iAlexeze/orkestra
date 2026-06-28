# Namespace Provisioner 01 — Explicit

One `NamespaceClaim` CR provisions a fully isolated tenant namespace. Resource limits are declared field-by-field in the CR spec — pods, CPU, and memory ceilings are explicit values you can read and audit at a glance.

**What you learn:** how to wire `namespaces`, `networkPolicies`, `resourceQuotas`, `clusterRoles`, and `clusterRoleBindings` together from a single CR; how motifs extract reusable cross-cutting concerns.

---

## What gets created

| Resource | Name | Where |
|---|---|---|
| Namespace | `team-alpha` | cluster |
| ServiceAccount | `alpha-operator` | `team-alpha` |
| ResourceQuota | `alpha-quota` | `team-alpha` |
| NetworkPolicy | `alpha-deny-all` | `team-alpha` |
| NetworkPolicy | `alpha-allow-dns` | `team-alpha` |
| ClusterRole | `alpha-ns-admin` | cluster |
| ClusterRoleBinding | `alpha-ns-admin-binding` | cluster |

The two NetworkPolicies come from the `tenant-isolation` motif. The ClusterRole and ClusterRoleBinding come from the `tenant-rbac` motif. Only the ResourceQuota is declared inline in the katalog — with explicit `hard` limits read from `spec.limits`.

---

## Step 1 — Validate

```bash
ork validate
```

---

## Step 2 — Simulate

No cluster needed. Verify all seven resources are created in cycle 1:

```bash
ork simulate
```

---

## Step 3 — Start the runtime

```bash
ork run
```

This applies the CRD, `setup.yaml`, and CR, then starts the runtime. No cluster yet? Add `--dev` to provision a local kind cluster first.

---

## Step 4 — Verify Resources

Check that the namespace and quota appeared:

```bash
kubectl get namespace team-alpha
kubectl get resourcequota alpha-quota -n team-alpha -o yaml
```

Verify the deny-all NetworkPolicy blocks all traffic:

```bash
kubectl get networkpolicies -n team-alpha
```

Verify the ClusterRoleBinding gives `alpha-operator` admin access:

```bash
kubectl get clusterrolebinding alpha-ns-admin-binding -o yaml
```

---

## CR fields used by this example

```yaml
spec:
  targetNamespace: team-alpha   # namespace to provision
  team: alpha                   # labels and RBAC naming
  owner: alpha-operator         # ServiceAccount to bind ClusterRole to
  ownerNamespace: default
  limits:
    maxPods: 20                 # ResourceQuota: max pods
    cpu: "4"                    # ResourceQuota: CPU ceiling
    memory: "8Gi"               # ResourceQuota: memory ceiling
```

Compare with [02-profiles](../02-profiles/README.md) — the same outcome with just `tier: medium` instead of explicit limits.

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies setup fixtures, starts the operator, applies the CR, asserts every expectation, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Namespace provisioned
    after: cr-applied
    resources:
      - kind: Namespace
        name: team-alpha

  - name: ResourceQuota has correct pod limit
    after: cr-applied
    commands:
      - run: kubectl get resourcequota alpha-quota -n team-alpha -o jsonpath='{.spec.hard.pods}'
        outputContains: "20"

  - name: Namespace removed
    after: cr-deleted
    resources:
      - kind: Namespace
        name: team-alpha
        count: 0
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
