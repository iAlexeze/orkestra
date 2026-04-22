# 09 — Kubernetes Notes

Kubernetes notes safely navigate the unstructured objects that Orkestra exposes through the template context — especially child resources available under `.children.*` and cross-CRD observations under `.cross.*`.

## What these notes operate on

The template context exposes Kubernetes objects as `map[string]interface{}` — the unstructured representation. Navigating them directly with Go template dot-notation works for top-level fields but becomes verbose for nested paths or optional structures. These notes provide typed, nil-safe access.

```yaml
# Direct dot-notation — works but panics if metadata is absent
# value: "{{ .children.deployment.metadata.labels.app }}"

# With kubernetes notes — nil-safe, returns "" if anything is absent
# value: "{{ mapGet (labels .children.deployment) \"app\" }}"
```

## Reference

### `meta`

Return the `metadata` map of a Kubernetes object. Returns an empty map if metadata is absent — never nil.

```yaml
# value: "{{ meta .children.cronjob }}"
# Useful as an intermediate value when chaining with mapGet:
# value: "{{ mapGet (meta .children.cronjob) \"resourceVersion\" }}"
```

---

### `labels`

Return the `metadata.labels` map. Returns an empty map if absent.

```yaml
# Check if a child has a specific label:
# value: "{{ mapGet (labels .children.deployment) \"app\" }}"
# {app: frontend, tier: web} → "frontend"
```

---

### `annotations`

Return the `metadata.annotations` map. Returns an empty map if absent.

```yaml
# value: "{{ mapGet (annotations .children.deployment) \"orkestra.io/phase\" }}"
```

---

### `spec`

Return the `spec` map of a Kubernetes object. Returns an empty map if absent.

```yaml
# value: "{{ mapGet (spec .children.cronjob) \"schedule\" }}"
# Equivalent to: .children.cronjob.spec.schedule
```

---

### `status`

Return the `status` map of a Kubernetes object. Returns an empty map if absent.

```yaml
# value: "{{ mapGet (status .children.deployment) \"readyReplicas\" }}"
# Equivalent to: .children.deployment.status.readyReplicas
```

---

### `get`

Navigate a nested path through a Kubernetes object using variadic string segments. Returns `nil` if any segment is missing. The most flexible of the navigation notes.

```yaml
# value: "{{ get .children.cronjob \"status\" \"lastScheduleTime\" }}"
# Equivalent to: .children.cronjob.status.lastScheduleTime

# value: "{{ get .children.deployment \"spec\" \"template\" \"spec\" \"containers\" }}"
# Returns the containers array
```

Unlike dot-notation, `get` is nil-safe at every segment — it will not panic if an intermediate key is absent.

---

### `ownerKind`

Return the `kind` of the first `ownerReference`. Useful for debugging controller relationships between child resources.

```yaml
# value: "{{ ownerKind .children.replicaset }}"
# → "Deployment"  (if owned by a Deployment)
```

---

### `ownerName`

Return the `name` of the first `ownerReference`.

```yaml
# value: "{{ ownerName .children.replicaset }}"
# → "my-app"
```

---

### `hasCondition`

Return `true` if the object's `status.conditions` array contains a condition of the given type with `status: "True"`.

```yaml
# Check if a Deployment's Available condition is true:
# value: "{{ hasCondition .children.deployment \"Available\" }}"

# Use in a when: block via status field:
# status:
#   fields:
#     - path: deploymentAvailable
#       value: "{{ hasCondition .children.deployment \"Available\" }}"
# Then condition:
# when:
#   - field: status.deploymentAvailable
#     equals: "true"
```

Standard Kubernetes condition types: `Available`, `Progressing`, `Degraded`, `Ready`, `Complete`.

---

## Using `.children.*` context

Child resources are injected into the resolver under `.children.<lowercaseKind>`. The kind is lowercased and singularized:

| Child type | Template path |
|-----------|--------------|
| Deployment | `.children.deployment` |
| Service | `.children.service` |
| CronJob | `.children.cronjob` |
| ConfigMap | `.children.configmap` |
| Secret | `.children.secret` |

Each value is a `map[string]interface{}` containing the full Kubernetes object — metadata, spec, status, and all.

```yaml
status:
  fields:
    - path: lastScheduleTime
      value: "{{ get .children.cronjob \"status\" \"lastScheduleTime\" }}"

    - path: deploymentReady
      value: "{{ hasCondition .children.deployment \"Available\" }}"
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
| `get` | `(obj any, path ...string)` | `any` |
| `ownerKind` | `(obj any)` | `string` |
| `ownerName` | `(obj any)` | `string` |
| `hasCondition` | `(obj any, condType string)` | `bool` |

---

**Next →** [10 — Container Notes](10-container.md)
