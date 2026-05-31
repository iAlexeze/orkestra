# KI-002 — Transient resource update conflict on fast reconciles

**Status:** Mitigated — pre-v1 architectural resolution planned (SSA)
**Affects:** Deployments, StatefulSets, ReplicaSets, Pods — any resource updated via `reconcile: true`

---

## What you see

A log line like:

```
level=error error="deployment.Update: updating deployment \"my-app\": Operation cannot be fulfilled
on deployments.apps \"my-app\": the object has been modified; please apply your changes to the
latest version and try again"
```

followed immediately by a successful reconcile on the next cycle. No resources are lost or corrupted.

---

## Why it happens

Orkestra's reconcile path uses the classic Get → modify → Update pattern. A status patch at the end of each reconcile increments the resource version of the CR, which re-queues an immediate reconcile. In fast-resync or high-churn scenarios, a second reconcile can begin before the first Update has fully committed, producing a version conflict.

This was most pronounced when `SyncContainerSpec` and `SyncPodSpec` compared all fields unconditionally — Kubernetes defaults like `ServiceAccountName: "default"` and `Protocol: "TCP"` differed from the zero values in the desired spec on every cycle, triggering unnecessary Updates that collided with status patches. The declared-intent guards introduced in `sync.go` eliminated the practical conditions under which this conflict occurred.

---

## Workaround

No action required. The reconciler retries on the next resync and converges correctly. If you see repeated conflict errors rather than a single transient one, check whether a field is being compared without a declared-intent guard in the relevant `Sync*` function.

---

## Resolution plan

Server Side Apply (SSA) is the architectural solution. Rather than Get → modify → Update, the operator sends only the fields it owns and Kubernetes merges them server-side using field ownership tracking. SSA eliminates the version conflict entirely because the server handles the merge — no resource version check is required.

Scheduled for resolution before v1.
