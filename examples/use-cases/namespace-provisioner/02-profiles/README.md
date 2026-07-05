# Namespace Provisioner 02 — Profiles

Same outcome as [01-explicit](../01-explicit/README.md) but the `ResourceQuota` is declared with `profile: "{{ .spec.tier }}"` instead of an explicit `hard` map. The CR only needs a tier name — Orkestra expands it to the correct pod count, CPU, and memory limits at reconcile time.

**What you learn:** how a profile reduces a CR from 8 fields to 4; that profiles and motifs compose cleanly — the NetworkPolicy and RBAC layers are unchanged from example 01.

---

## What gets created

| Resource | Name | Where |
|---|---|---|
| Namespace | `team-beta` | cluster |
| ServiceAccount | `beta-operator` | `team-beta` |
| ResourceQuota | `beta-quota` | `team-beta` |
| NetworkPolicy | `beta-deny-all` | `team-beta` |
| NetworkPolicy | `beta-allow-dns` | `team-beta` |
| ClusterRole | `beta-ns-admin` | cluster |
| ClusterRoleBinding | `beta-ns-admin-binding` | cluster |

---

## Profile reference

| Tier | Pods | CPU | Memory |
|---|---|---|---|
| `small` | 10 | 2 | 4Gi |
| `medium` | 20 | 4 | 8Gi |
| `large` | 50 | 8 | 16Gi |
| `xlarge` | 100 | 16 | 32Gi |

The CR uses `tier: medium` — the operator expands it to `pods: 20`, `requests.cpu: 4`, `requests.memory: 8Gi` without those values appearing anywhere in the katalog.

---

## Step 1 — Validate

```bash
ork validate
```

---

## Step 2 — Simulate

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

Confirm the `medium` preset expanded correctly:

```bash
kubectl get resourcequota beta-quota -n team-beta -o jsonpath='{.spec.hard}' | jq .
```

Expected:

```json
{
  "limits.cpu": "4",
  "limits.memory": "8Gi",
  "pods": "20",
  "requests.cpu": "4",
  "requests.memory": "8Gi"
}
```

Verify the deny-all NetworkPolicy blocks all traffic:

```bash
kubectl get networkpolicies -n team-beta
```

Verify the ClusterRoleBinding gives `beta-operator` admin access:

```bash
kubectl get clusterrolebinding beta-ns-admin-binding -o yaml
```

---

## CR fields used by this example

```yaml
spec:
  targetNamespace: team-beta
  team: beta
  owner: beta-operator
  ownerNamespace: default
  tier: medium          # replaces the entire spec.limits block from 01-explicit
```

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
        name: team-beta

  - name: medium profile expands to 20 pods
    after: cr-applied
    commands:
      - run: kubectl get resourcequota beta-quota -n team-beta -o jsonpath='{.spec.hard.pods}'
        outputContains: "20"

  - name: Namespace removed
    after: cr-deleted
    resources:
      - kind: Namespace
        name: team-beta
        count: 0
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
