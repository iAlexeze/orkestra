# CRD Detail View

Click "View details" on any CRD card to open the CRD Detail view. This is the deepest level of visibility in the Control Center.

---

## Worker Pool

Every worker goroutine is shown individually:

| State | Meaning |
|-------|---------|
| Processing (pulsing blue) | Currently reconciling an item |
| Idle (green) | Waiting for work |
| Stopped (red) | Worker terminated — CRD is deactivated |

If all workers are idle and the queue has items, a worker may be stuck. Check "Last Reconcile" and "Last Error" below.

For CRDs with more than 10 workers, the first 10 are shown with a "Show all N workers" toggle.

---

## Queue

Queue depth is shown as a progress bar with exact numbers. Large values are formatted (e.g., `15.2K`). Hover for the exact count.

---

## Runtime Health

| Field | Description |
|-------|-------------|
| Uptime | How long this CRD has been registered |
| Started | When its workers last started |
| Last Reconcile | When the last reconcile completed |
| Consecutive Failures | Count of back-to-back reconcile errors |
| Last Error | The most recent error message |

---

## Dependencies

If this CRD depends on others, each dependency is shown with its current state. A dependency in Pending or Degraded state will block this CRD from reconciling. Fix the dependency first.

---

## RBAC Permissions

Orkestra derives RBAC permissions directly from the Katalog — from the resources declared in `onCreate`, `onUpdate`, and `onDelete` blocks. The CRD Detail view shows exactly what permissions this operator requires.

This lets you audit least-privilege without reading ClusterRole YAML.

---

## Admission Webhooks

If validation or mutation rules are configured for this CRD, this section shows:

| Metric | Description |
|--------|-------------|
| Total requests | All admission requests received |
| Allowed / Denied | Validation outcomes |
| Applied / Skipped | Mutation outcomes |
| Avg / P95 latency | Response time |

---

## Version Conversion

If the CRD declares multiple API versions, conversion metrics appear here:

| Metric | Description |
|--------|-------------|
| Total requests | Conversion requests received |
| Success / Failure | Outcomes |
| Avg / P95 latency | Conversion time |

---

## Reconciler Configuration

| Field | Description |
|-------|-------------|
| Reconciler type | Generic (declarative) or Custom (Go hooks) |
| Resync | How often to re-reconcile even without a change event |
| Finalizers | Cleanup finalizers configured for this CRD |
