# Ordered Deletion

When a CR is deleted, Orkestra runs the `onDelete:` block before removing the finalizer. By default, all resource groups execute concurrently — deletion requests are submitted in parallel and the finalizer is removed immediately.

Orkestra provides two models when the sequence matters.

---

## When you need it

Most operators do not need ordered deletion. Owner references and Kubernetes garbage collection handle cascade deletion correctly for the common case.

Ordered deletion is for the cases where sequence matters:

- A cleanup Job must complete before its target is deleted
- Infrastructure must be deprovisioned before its credentials Secret is removed
- A final backup must complete before the workload is torn down

---

## Two models

| | Hard ordered | Condition-based |
|---|---|---|
| Mechanism | `ordered: true` | `when:` / `or:` conditions |
| Finalizer | Held until complete | Never held |
| CR can get stuck | Yes (on timeout) | Never |
| Guarantee | Sequential, enforced | Best-effort |
| Use case | Safety-critical cleanup | Optional sequencing |

---

## Where to go next

- [Hard Ordered Deletion](hard-ordered/) — `ordered: true`, groups, timeouts
- [Condition-Based Deletion](condition-based/) — `when:` / `or:` sequencing without blocking
