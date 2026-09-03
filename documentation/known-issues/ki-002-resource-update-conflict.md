# KI-002 — Transient resource update conflict on fast reconciles

**Status:** Resolved in v0.7.12
**Affects:** Deployments, StatefulSets, ReplicaSets, Pods — any resource updated via `reconcile: true`

---

## Resolution

All reconcilable resource packages were migrated to Kubernetes Server-Side Apply in v0.7.12. The runtime sends only the fields it owns (`fieldManager: orkestra-runtime`, `force: true`); Kubernetes merges them server-side using field ownership tracking. No resource version check is required, so the Get → Update race that caused this conflict is eliminated entirely.

---

## Historical context

Orkestra's reconcile path previously used the classic Get → modify → Update pattern. A status patch at the end of each reconcile incremented the resource version of the CR, re-queuing an immediate reconcile. In fast-resync or high-churn scenarios a second reconcile could begin before the first Update had committed, producing:

```text
level=error error="deployment.Update: updating deployment \"my-app\": Operation cannot be fulfilled
on deployments.apps \"my-app\": the object has been modified; please apply your changes to the
latest version and try again"
```

The conflict was transient — no resources were lost or corrupted — and was mitigated by declared-intent guards that eliminated unnecessary Updates from k8s-injected defaults. SSA closes it permanently.
