# 12 — Replica Notes

Replica notes read rollout state from Deployment, ReplicaSet, and StatefulSet objects. They replace verbose `get`/`mapGet` chains for the most commonly gated condition in async operator workflows: "is the rollout complete?"

---

## Reference

### `replicasReady`

Return `true` when `status.readyReplicas >= spec.replicas`. The canonical rollout-complete gate.

```yaml
# Gate dependent resources on a stable rollout:
when:
  - field: "{{ allReplicasReady.children.deployment }}"
    equals: "true"
```

---

### `readyReplicas`

Return `status.readyReplicas` as `int`. Returns `0` when absent.

```yaml
# value: "{{ readyReplicas .children.deployment }}"  → 3

# Surface in status:
- path: readyCount
  value: "{{ readyReplicas .children.deployment }}"
```

---

### `availableReplicas`

Return `status.availableReplicas` as `int`. Returns `0` when absent.

Available replicas are those that have passed the minReadySeconds threshold. Slightly lags `readyReplicas` during rapid rollouts.

```yaml
# value: "{{ availableReplicas .children.deployment }}"  → 3
```

---

### `updatedReplicas`

Return `status.updatedReplicas` as `int` — pods running the current pod template. When `updatedReplicas == desiredReplicas`, the rollout is complete.

```yaml
# value: "{{ updatedReplicas .children.deployment }}"  → 3

# Check rollout progress without waiting for readiness:
- path: rolloutComplete
  value: "{{ eq (updatedReplicas .children.deployment) (desiredReplicas .children.deployment) }}"
```

---

### `desiredReplicas`

Return `spec.replicas` as `int`. Returns `1` when not set (Kubernetes default).

```yaml
# value: "{{ desiredReplicas .children.deployment }}"  → 3
```

---

## Combining replica notes

```yaml
status:
  fields:
    # Expose rollout progress as a fraction
    - path: rolloutProgress
      value: "{{ div (toFloat (readyReplicas .children.deployment)) (toFloat (desiredReplicas .children.deployment)) }}"

    # True only when all pods are both updated and ready
    - path: fullyRolledOut
      value: "{{ and (replicasReady .children.deployment) (eq (updatedReplicas .children.deployment) (desiredReplicas .children.deployment)) }}"

when:
  - field: "{{ allReplicasReady.children.deployment }}"
    equals: "true"
```

---

## Quick reference

| Note | Signature | Returns |
|------|-----------|---------|
| `replicasReady` | `(obj any)` | `bool` |
| `readyReplicas` | `(obj any)` | `int` |
| `availableReplicas` | `(obj any)` | `int` |
| `updatedReplicas` | `(obj any)` | `int` |
| `desiredReplicas` | `(obj any)` | `int` |

---

**Next →** [13 — Job Notes](13-job.md)
