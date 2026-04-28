# Replica Notes

Inspect Deployment and ReplicaSet replica counts. All notes accept the unstructured child object from `.children.*`.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `replicasReady` | `obj` | `bool` — `readyReplicas == desiredReplicas` |
| `desiredReplicas` | `obj` | `int` — `spec.replicas` (1 when unset, 0 for explicit scale-to-zero) |
| `readyReplicas` | `obj` | `int` — `status.readyReplicas` |
| `availableReplicas` | `obj` | `int` — `status.availableReplicas` |
| `updatedReplicas` | `obj` | `int` — `status.updatedReplicas` (pods on current spec) |

## Scale-to-zero

`replicasReady` uses `ready == desired` — it returns `true` when both are 0. A deployment explicitly scaled to zero is considered ready once all pods have terminated.

`desiredReplicas` distinguishes between "field absent" (returns 1 — Kubernetes default) and "field explicitly set to 0" (returns 0 — scale-to-zero intent).

## Examples

```yaml
# Gate a dependent resource on rollout completion
when:
  - field: "{{ replicasReady .children.deployment }}"
    equals: "true"

# Status: how many replicas are running
- path: readyReplicas
  value: "{{ readyReplicas .children.deployment }}"

# Status: is the rollout complete?
- path: rolloutComplete
  value: "{{ eq (updatedReplicas .children.deployment) (desiredReplicas .children.deployment) }}"
```
