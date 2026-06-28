# Namespace Provisioner 03 — User-Defined Profiles

The built-in profiles (`deny-all`, `medium`, `safe`) exist so things work out of the box. But the reason profiles exist in Orkestra is so your organisation can name the things that matter to it.

When a Katalog says `profile: org-medium`, that name is a contract between the platform team and every app team using this operator. It means: this is what medium means here, in this organisation, right now. Not a Kubernetes detail. Not a number to look up. A name your teams agree on, enforced at load time, carried through every Motif that imports it.

This example declares all six profile classes in the Katalog's `profiles:` block. Every name is org-owned — prefixed `org-` to make that explicit. None of them are Orkestra built-ins.

**What you learn:** how to declare the profile registry your org actually needs; that `ork validate` enforces every reference against your definitions, not Orkestra's; that user profiles resolve before built-ins, so you are never constrained by what ships in the box.

**Contrast with 02:** example 02 borrows a name Orkestra ships (`medium`). This example owns the name and the definition. The distinction is the whole point.

---

## What gets created

| Resource | Name | Where | Profile used |
|---|---|---|---|
| Namespace | `team-gamma` | cluster | — |
| ServiceAccount | `gamma-operator` | `team-gamma` | — |
| NetworkPolicy | `gamma-deny-all` | `team-gamma` | `org-deny-all` |
| NetworkPolicy | `gamma-allow-dns` | `team-gamma` | `org-allow-dns-egress` |
| NetworkPolicy | `gamma-allow-monitoring` | `team-gamma` | `org-allow-monitoring` |
| ResourceQuota | `gamma-quota` | `team-gamma` | `org-medium` (via template) |
| LimitRange | `gamma-container-limits` | `team-gamma` | `org-container-defaults` |
| Deployment | `gamma-ns-agent` | `team-gamma` | rollingUpdate: `org-safe` |
| HorizontalPodAutoscaler | `gamma-ns-agent-hpa` | `team-gamma` | behavior: `org-conservative` |
| PodDisruptionBudget | `gamma-ns-agent-pdb` | `team-gamma` | behavior: `org-at-least-one` |
| ClusterRole | `gamma-ns-admin` | cluster | — (from `tenant-rbac` motif) |
| ClusterRoleBinding | `gamma-ns-admin-binding` | cluster | — (from `tenant-rbac` motif) |

---

## Profiles declared in this Katalog

| Class | Name | Purpose |
|---|---|---|
| `networkPolicies` | `org-deny-all` | Block all ingress and egress |
| `networkPolicies` | `org-allow-dns-egress` | Allow UDP/TCP 53 for DNS |
| `networkPolicies` | `org-allow-monitoring` | Allow ingress from `team: platform` namespace |
| `resourceQuotas` | `org-small` | 10 pods, 1 CPU, 2Gi |
| `resourceQuotas` | `org-medium` | 25 pods, 4 CPU, 8Gi |
| `resourceQuotas` | `org-large` | 60 pods, 12 CPU, 24Gi |
| `limitRanges` | `org-container-defaults` | Default 100m/128Mi request, 500m/512Mi limit |
| `hpa` | `org-conservative` | 70% CPU target, 5-min scale-down stabilization |
| `hpa` | `org-burst` | 60% CPU target, fast scale-down |
| `pdb` | `org-at-least-one` | minAvailable: 1 |
| `pdb` | `org-majority` | minAvailable: 51% |
| `rollingUpdate` | `org-safe` | maxSurge: 1, maxUnavailable: 0 |
| `rollingUpdate` | `org-fast` | maxSurge: 25%, maxUnavailable: 25% |

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

Confirm the namespace and all profiles expanded correctly:

```bash
kubectl get namespace team-gamma
kubectl get networkpolicies -n team-gamma

kubectl get resourcequota gamma-quota -n team-gamma -o jsonpath='{.spec.hard.pods}' && echo
# 25 — from org-medium

kubectl get limitrange gamma-container-limits -n team-gamma -o yaml

kubectl get deployment gamma-ns-agent -n team-gamma -o jsonpath='{.spec.strategy.rollingUpdate}' && echo
# {"maxSurge":1,"maxUnavailable":0} — from org-safe

kubectl get hpa gamma-ns-agent-hpa -n team-gamma
kubectl get pdb gamma-ns-agent-pdb -n team-gamma
kubectl get clusterrolebinding gamma-ns-admin-binding -o yaml
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
  - name: Three NetworkPolicies from user-defined profiles
    after: cr-applied
    resources:
      - kind: NetworkPolicy
        name: gamma-deny-all
        namespace: team-gamma
      - kind: NetworkPolicy
        name: gamma-allow-dns
        namespace: team-gamma
      - kind: NetworkPolicy
        name: gamma-allow-monitoring
        namespace: team-gamma

  - name: org-medium expands to 25 pods
    after: cr-applied
    commands:
      - run: kubectl get resourcequota gamma-quota -n team-gamma -o jsonpath='{.spec.hard.pods}'
        outputContains: "25"
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
