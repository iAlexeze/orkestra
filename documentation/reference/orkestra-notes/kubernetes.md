# Kubernetes Notes

Safe traversal and inspection of Kubernetes objects. All notes accept `map[string]interface{}` — the unstructured form that Orkestra injects for child resources via `.children.*`.

---

## Navigation

| Note | Signature | Returns |
|------|-----------|---------|
| `get` | `obj interface{}, keys ...string` | `interface{}` — safe deep traversal |
| `meta` | `obj` | `map[string]interface{}` — metadata block |
| `name` | `obj` | `string` |
| `namespace` | `obj` | `string` |
| `labels` | `obj` | `map[string]interface{}` |
| `annotations` | `obj` | `map[string]interface{}` |
| `spec` | `obj` | `map[string]interface{}` |
| `status` | `obj` | `map[string]interface{}` |
| `phase` | `obj` | `string` |

```yaml
- path: lastScheduleTime
  value: "{{ get .children.cronjob \"status\" \"lastScheduleTime\" }}"

- path: childLabels
  value: "{{ labels .children.deployment }}"
```

---

## Metadata fields

Direct accessors for the most common metadata fields.

| Note | Signature | Returns |
|------|-----------|---------|
| `resourceName` | `obj` | `string` |
| `resourceNamespace` | `obj` | `string` |
| `resourceUID` | `obj` | `string` |
| `resourceVersion` | `obj` | `string` (etcd revision) |
| `creationTimestamp` | `obj` | `string` (RFC3339) |

```yaml
- path: childUID
  value: "{{ resourceUID .children.deployment }}"
```

---

## Owner references

| Note | Signature | Returns |
|------|-----------|---------|
| `ownerKind` | `obj` | `string` — kind of the first owner |
| `ownerName` | `obj` | `string` — name of the first owner |

---

## Conditions

Kubernetes conditions follow the standard `type`/`status`/`reason`/`message` structure.

| Note | Signature | Returns |
|------|-----------|---------|
| `hasCondition` | `obj, type string` | `bool` — condition exists with `status: "True"` |
| `conditionReason` | `obj, type string` | `string` |
| `conditionMessage` | `obj, type string` | `string` |

```yaml
- path: deploymentReady
  value: "{{ hasCondition .children.deployment \"Available\" }}"

- path: deploymentMessage
  value: "{{ conditionMessage .children.deployment \"Progressing\" }}"

when:
  - field: "{{ hasCondition .children.deployment \"Available\" }}"
    equals: "true"
```

---

## Lifecycle

| Note | Signature | Returns |
|------|-----------|---------|
| `resourceExists` | `obj` | `bool` — object has been created |
| `isTerminating` | `obj` | `bool` — deletion timestamp is set |
| `generation` | `obj` | `int64` |
| `observedGeneration` | `obj` | `int64` |
| `isSynced` | `obj` | `bool` — `generation == observedGeneration` |

```yaml
when:
  - field: "{{ isSynced .children.deployment }}"
    equals: "true"
  - field: "{{ isTerminating .children.deployment }}"
    equals: "false"
```
