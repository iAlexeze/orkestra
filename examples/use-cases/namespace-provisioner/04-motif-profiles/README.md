# Namespace Provisioner 04 — Profiles via Motif

Same outcome as [03-user-defined-profiles](../03-user-defined-profiles/README.md) — all six profile classes exercised — but the profile definitions live in the `tenant-policies` motif, not in this Katalog. The Katalog imports the motif and references profile names; it holds no `profiles:` block at all.

**What you learn:** how to move profile definitions into a shared motif so multiple Katalogs can reference the same names; that imported profiles merge into the Katalog's registry at load time and resolve exactly like locally declared ones.

**Contrast with 03:** example 03 owns its profiles inline. This example delegates the entire registry to a motif — the Katalog only says `profile: org-conservative`; the definition lives in `tenant-policies`, owned and versioned by whoever maintains the motif.

---

## What gets created

| Resource | Name | Where | Profile source |
|---|---|---|---|
| Namespace | `team-delta` | cluster | — |
| ServiceAccount | `delta-operator` | `team-delta` | — |
| NetworkPolicy | `delta-deny-all` | `team-delta` | motif resource using `org-deny-all` |
| NetworkPolicy | `delta-allow-dns` | `team-delta` | motif resource using `org-allow-dns-egress` |
| NetworkPolicy | `delta-allow-monitoring` | `team-delta` | motif resource using `org-allow-monitoring` |
| ResourceQuota | `delta-quota` | `team-delta` | `org-medium` (from motif registry) |
| LimitRange | `delta-container-limits` | `team-delta` | `org-container-defaults` (from motif registry) |
| Deployment | `delta-ns-agent` | `team-delta` | rollingUpdate: `org-safe` (from motif registry) |
| HorizontalPodAutoscaler | `delta-ns-agent-hpa` | `team-delta` | behavior: `org-conservative` (from motif registry) |
| PodDisruptionBudget | `delta-ns-agent-pdb` | `team-delta` | behavior: `org-at-least-one` (from motif registry) |
| ClusterRole | `delta-ns-admin` | cluster | — (from `tenant-rbac` motif) |
| ClusterRoleBinding | `delta-ns-admin-binding` | cluster | — (from `tenant-rbac` motif) |

---

## Motifs imported

| Motif | What it provides |
|---|---|
| [`tenant-policies`](../motifs/tenant-policies/motif.yaml) | All six profile classes + three NetworkPolicies |
| [`tenant-rbac`](../motifs/tenant-rbac/motif.yaml) | ClusterRole + ClusterRoleBinding |

---

## Step 1 — Validate

```bash
ork validate
```

---

## Step 2 — Simulate

No cluster needed. Verify all twelve resources are created in cycle 1:

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

Confirm the namespace and all profiles expanded correctly — every value comes from the motif:

```bash
kubectl get namespace team-delta
kubectl get networkpolicies -n team-delta#

kubectl get resourcequota delta-quota -n team-delta -o jsonpath='{.spec.hard.pods}' && echo
# 25 — from org-medium in tenant-policies motif

kubectl get limitrange delta-container-limits -n team-delta -o yaml

kubectl get deployment delta-ns-agent -n team-delta -o jsonpath='{.spec.strategy.rollingUpdate}' && echo
# {"maxSurge":1,"maxUnavailable":0} — from org-safe

kubectl get hpa delta-ns-agent-hpa -n team-delta
kubectl get pdb delta-ns-agent-pdb -n team-delta
kubectl get clusterrolebinding delta-ns-admin-binding -o yaml
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
  - name: Three NetworkPolicies from tenant-policies motif
    after: cr-applied
    resources:
      - kind: NetworkPolicy
        name: delta-deny-all
        namespace: team-delta
      - kind: NetworkPolicy
        name: delta-allow-dns
        namespace: team-delta
      - kind: NetworkPolicy
        name: delta-allow-monitoring
        namespace: team-delta

  - name: org-medium expands to 25 pods
    after: cr-applied
    commands:
      - run: kubectl get resourcequota delta-quota -n team-delta -o jsonpath='{.spec.hard.pods}'
        outputContains: "25"
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
