# Known Issues

Pre-v1 issues tracked here. Each entry is a real limitation — not a bug in the core logic, but something that behaves unexpectedly in specific configurations. All are scheduled to be resolved before the v1 release.

---

| # | Title | Affects | Status |
|---|-------|---------|--------|
| [KI-001](./ki-001-gateway-stats-multi-replica.md) | Gateway webhook stats show zeros with multiple replicas | Gateway `/katalog` endpoint | Open — pre-v1 |
| [KI-002](./ki-002-resource-update-conflict.md) | Transient resource update conflict on fast reconciles | Deployments, StatefulSets, ReplicaSets, Pods | Mitigated — pre-v1 SSA planned |

---

> The core operator runtime, conversion, and deletion protection work correctly in all configurations. These known issues are observability and tooling gaps, not correctness issues.
