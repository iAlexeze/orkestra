# Namespace Provisioner

A `NamespaceClaim` operator that fully provisions a tenant namespace — network isolation, resource quotas, and RBAC — from a single CR.

| Example | What it teaches |
|---|---|
| [01 — Explicit](01-explicit/README.md) | Namespace + NetworkPolicy + ResourceQuota (inline limits) + ClusterRole/Binding |
| [02 — Profiles](02-profiles/README.md) | Same outcome with `tier: medium` instead of explicit resource limits |
| [03 — User-defined profiles](03-user-defined-profiles/README.md) | All six profile classes declared in the Katalog's `profiles:` block |
| [04 — Profiles via motif](04-motif-profiles/README.md) | Same six classes, but definitions live in the `tenant-policies` motif — Katalog holds no `profiles:` block |

All examples share `crd.yaml` and three reusable motifs:

| Motif | What it provides |
|---|---|
| [`tenant-isolation`](motifs/tenant-isolation/motif.yaml) | Two NetworkPolicies: deny-all + allow-dns-egress |
| [`tenant-rbac`](motifs/tenant-rbac/motif.yaml) | ClusterRole + ClusterRoleBinding scoped to the provisioned namespace |
| [`tenant-policies`](motifs/tenant-policies/motif.yaml) | All six profile classes + three NetworkPolicies; used by example 04 |

---

## Quick start

```bash
# Run example 01 (explicit limits)
cd 01-explicit
ork validate && ork simulate && ork run

# In a separate terminal
kubectl apply -f ./cr.yaml
```

---

## E2E

```bash
ork e2e
```

---

## Simulate (no cluster needed)

```bash
ork simulate
```
