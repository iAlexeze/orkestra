# 09 — Kubernetes Notes

Kubernetes notes safely navigate the unstructured objects that Orkestra exposes through the template context — especially child resources available under `.children.*` and cross-CRD observations under `.cross.*`.

## What these notes operate on

The template context exposes Kubernetes objects as `map[string]interface{}` — the unstructured representation. Navigating them directly with Go template dot-notation works for top-level fields but becomes verbose for nested paths or optional structures. These notes provide typed, nil-safe access.

```yaml
# Direct dot-notation — panics if metadata is absent
# value: "{{ .children.deployment.metadata.labels.app }}"

# With kubernetes notes — nil-safe, returns "" if anything is absent
# value: "{{ mapGet (labels .children.deployment) \"app\" }}"
```

Notes work in both `value:` and `field:` positions:

```yaml
# value: field
- path: ready
  value: "{{ allReplicasReady.children.deployment }}"

# field: condition (template syntax)
when:
  - field: "{{ allReplicasReady.children.deployment }}"
    equals: "true"
```

---

## Metadata access

### `meta`

Return the `metadata` map of a Kubernetes object. Returns an empty map if metadata is absent — never nil.

Keywords: kubernetes, metadata, object, navigate, map

```yaml
# value: "{{ meta .children.cronjob }}"
# Useful as an intermediate value when chaining with mapGet:
# value: "{{ mapGet (meta .children.cronjob) \"resourceVersion\" }}"
```

---

### `labels`

Return the `metadata.labels` map. Returns an empty map if absent.

Keywords: kubernetes, metadata, labels, map, selector, tags

```yaml
# value: "{{ mapGet (labels .children.deployment) \"app\" }}"
# {app: frontend, tier: web} → "frontend"
```

---

### `annotations`

Return the `metadata.annotations` map. Returns an empty map if absent.

Keywords: kubernetes, metadata, annotations, map, tags

```yaml
# value: "{{ mapGet (annotations .children.deployment) \"orkestra.io/phase\" }}"
```

---

## Spec and status

### `spec`

Return the `spec` map of a Kubernetes object. Returns an empty map if absent.

Keywords: kubernetes, spec, object, navigate, map

```yaml
# value: "{{ mapGet (spec .children.cronjob) \"schedule\" }}"
# Equivalent to: .children.cronjob.spec.schedule
```

---

### `status`

Return the `status` map of a Kubernetes object. Returns an empty map if absent.

Keywords: kubernetes, status, object, navigate, map

```yaml
# value: "{{ mapGet (status .children.deployment) \"readyReplicas\" }}"
# Equivalent to: .children.deployment.status.readyReplicas
```

---

### `phase`

Return `status.phase` as a string. Returns `""` when absent.

Keywords: kubernetes, status, phase, string, pod, lifecycle

```yaml
# value: "{{ phase .children.pod }}"
# → "Running", "Pending", "Succeeded", "Failed", or ""

# when:
#   - field: "{{ phase .children.pod }}"
#     equals: "Running"
```

---

## Safe nested field lookup

### `get`

Navigate a nested path through a Kubernetes object using variadic string segments. Returns `nil` if any segment is missing. Nil-safe at every level.

Keywords: kubernetes, navigate, nested, path, safe, access, deep

```yaml
# value: "{{ get .children.cronjob \"status\" \"lastScheduleTime\" }}"
# value: "{{ get .children.deployment \"spec\" \"template\" \"spec\" \"containers\" }}"
```

---

## Owner references

### `ownerKind`

Return the `kind` of the first `ownerReference`.

Keywords: kubernetes, owner, reference, kind, controller

```yaml
# value: "{{ ownerKind .children.replicaset }}"
# → "Deployment"
```

---

### `ownerName`

Return the `name` of the first `ownerReference`.

Keywords: kubernetes, owner, reference, name, controller

```yaml
# value: "{{ ownerName .children.replicaset }}"
# → "my-app"
```

---

## Conditions

### `hasCondition`

Return `true` if `status.conditions` contains a condition of the given type with `status: "True"`.

Keywords: kubernetes, condition, status, check, boolean, ready, available

```yaml
# value: "{{ hasCondition .children.deployment \"Available\" }}"
# Standard types: Available, Progressing, Degraded, Ready, Complete
```

---

### `conditionReason`

Return the `reason` field of a named condition. Returns `""` when absent.

Keywords: kubernetes, condition, reason, status, string, message

```yaml
# value: "{{ conditionReason .children.deployment \"Available\" }}"
# → "MinimumReplicasAvailable" or ""
```

---

### `conditionMessage`

Return the `message` field of a named condition. Returns `""` when absent.

Keywords: kubernetes, condition, message, status, string, detail

```yaml
# value: "{{ conditionMessage .children.deployment \"Progressing\" }}"
# → "ReplicaSet has successfully progressed." or ""
```

---

## Existence and lifecycle

### `resourceExists`

Return `true` when the object is a non-nil `map[string]interface{}`. Use to check whether a child resource has been created.

Keywords: kubernetes, exists, check, boolean, nil, created, present

```yaml
# value: "{{ resourceExists .children.deployment }}"

# Gate dependent resources on child existence:
# when:
#   - field: "{{ resourceExists .children.secret }}"
#     equals: "true"
```

---

### `isTerminating`

Return `true` when `metadata.deletionTimestamp` is set — the object exists but is being deleted.

Keywords: kubernetes, terminating, deleting, boolean, lifecycle, deletion

```yaml
# Gate traffic routing away from terminating pods:
# when:
#   - field: "{{ isTerminating .children.deployment }}"
#     equals: "false"
```

---

## Generation tracking

### `generation`

Return `metadata.generation` as int64. Increments on every spec change. Returns `0` when absent.

Keywords: kubernetes, generation, metadata, revision, int

```yaml
# value: "{{ generation .children.deployment }}"
# → 3
```

---

### `observedGeneration`

Return `status.observedGeneration` as int64. This is the generation the controller last acted on.

Keywords: kubernetes, generation, status, observed, revision, int

```yaml
# value: "{{ observedGeneration .children.deployment }}"
# → 3
```

---

### `isSynced`

Return `true` when `metadata.generation == status.observedGeneration`, meaning the controller has fully processed the current spec. Returns `true` for resources without generation tracking (both 0).

Keywords: kubernetes, synced, generation, rollout, boolean, ready, reconciled

```yaml
# Gate dependent resources on rollout completion:
# when:
#   - field: "{{ isSynced .children.deployment }}"
#     equals: "true"
```

---

## Using `.children.*` context

Child resources are injected under `.children.<lowercaseKind>`. Each value is a full `map[string]interface{}` — metadata, spec, status, and all.

| Child type | Template path |
|-----------|--------------|
| Deployment | `.children.deployment` |
| Service | `.children.service` |
| CronJob | `.children.cronjob` |
| ConfigMap | `.children.configmap` |
| Secret | `.children.secret` |

Use notes instead of raw dot-notation in both `value:` and `field:` positions:

```yaml
status:
  fields:
    - path: lastScheduleTime
      value: "{{ get .children.cronjob \"status\" \"lastScheduleTime\" }}"
    - path: deploymentReady
      value: "{{ hasCondition .children.deployment \"Available\" }}"
    - path: phase
      value: "{{ phase .children.pod }}"

when:
  - field: "{{ isSynced .children.deployment }}"
    equals: "true"
  - field: "{{ resourceExists .children.secret }}"
    equals: "true"
```

---

## Quick reference

| Note | Signature | Returns |
|------|-----------|---------|
| `meta` | `(obj any)` | `map[string]interface{}` |
| `labels` | `(obj any)` | `map[string]interface{}` |
| `annotations` | `(obj any)` | `map[string]interface{}` |
| `spec` | `(obj any)` | `map[string]interface{}` |
| `status` | `(obj any)` | `map[string]interface{}` |
| `phase` | `(obj any)` | `string` |
| `get` | `(obj any, path ...string)` | `any` |
| `ownerKind` | `(obj any)` | `string` |
| `ownerName` | `(obj any)` | `string` |
| `hasCondition` | `(obj any, condType string)` | `bool` |
| `conditionReason` | `(obj any, condType string)` | `string` |
| `conditionMessage` | `(obj any, condType string)` | `string` |
| `resourceExists` | `(obj any)` | `bool` |
| `isTerminating` | `(obj any)` | `bool` |
| `generation` | `(obj any)` | `int64` |
| `observedGeneration` | `(obj any)` | `int64` |
| `isSynced` | `(obj any)` | `bool` |

---

**Next →** [10 — Container Notes](10-container.md)
