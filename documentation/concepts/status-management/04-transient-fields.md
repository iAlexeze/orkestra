# Transient Status Fields

Some status fields only make sense while a condition is active — a crash reason while pods are crashing, a warning message while events are firing, old replica counts during a rollout. When the condition clears, the field should disappear too.

By default it doesn't. A field with a `when:` condition is simply skipped when the condition is false — the previous value stays in etcd unchanged. This is correct for fields like `phase`, where you want the last known phase to persist. For transient diagnostic fields it leaves stale data.

`clearOnFalse: true` changes that behaviour for a single field.

---

## Declaration

```yaml
status:
  fields:
    - path: crashReason
      value: "{{ podContainerReasons .children.deployment }}"
      clearOnFalse: true
      when:
        - field: "{{ hasCrashingPod .children.deployment }}"
          equals: "true"
```

When `hasCrashingPod` is `true`, `crashReason` is written with the container reason string.  
When `hasCrashingPod` is `false`, `crashReason` is written as `""`, clearing the previous value.

Without `clearOnFalse`, the condition failing means the field is left untouched — `ImagePullBackOff` would still be visible in status after the pods recovered.

---

## When to use it

Add `clearOnFalse: true` to any field whose value is only meaningful while a condition holds:

| Field type | Example |
|---|---|
| Crash or error reason | `crashReason`, `errorMessage` |
| Diagnostic counts | `warningCount`, `oldReplicaSets` |
| Ephemeral messages | `firstWarning`, `degradedReason` |
| Debug-mode data | `debugReplicaSetCount` |

Do **not** use it on `phase` or other fields that describe the last stable state — leaving the previous value is the correct behaviour there.

---

## Effect on always-written fields

`clearOnFalse` has no effect when no `when:` or `anyOf:` conditions are declared. Fields written unconditionally are not affected.

---

## Example: warning events

```yaml
- path: firstWarning
  value: "{{ firstWarning .children.deployment }}"
  clearOnFalse: true
  when:
    - field: "{{ hasWarnings .children.deployment }}"
      equals: "true"
```

While the deployment has warnings → `firstWarning` is populated.  
When the deployment recovers → `firstWarning` is cleared to `""` on the next reconcile.
