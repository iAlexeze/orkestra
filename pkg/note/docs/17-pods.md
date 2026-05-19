# 17 — Pod Notes

Pod notes navigate the `_pods` enrichment embedded by `children.go` into Deployment, StatefulSet, and ReplicaSet resource maps. They let templates surface pod names, IPs, counts, and health without hooks or custom Go code.

Pod enrichment is opt-in at the CRD level:

```yaml
spec:
  crds:
    my-operator:
      enrich: [pods]   # or enrichAll: true
      operatorBox:
        ...
```

Without enrichment, `_pods` is absent and all pod notes return their zero values (`""`, `0`, `false`) — safe to use in templates regardless.

---

## Reference

### `podNames`

Return a comma-separated string of pod names owned by the enriched resource.

```yaml
status:
  fields:
    - path: pods
      value: "{{ podNames .children.deployment }}"
# → "web-abc, web-def"
```

---

### `podIPs`

Return a comma-separated string of pod IP addresses. Returns `""` while pods are pending (IPs not yet assigned).

```yaml
- path: podIPs
  value: "{{ podIPs .children.statefulset }}"
# → "10.0.0.1, 10.0.0.2, 10.0.0.3"
```

---

### `podPhases`

Return a comma-separated string of pod phases in order. Useful for surfacing the phase distribution of a multi-replica Deployment or StatefulSet.

```yaml
- path: podPhases
  value: "{{ podPhases .children.statefulset }}"
# → "Running, Running, Pending"
```

---

### `podNodes`

Return a comma-separated list of node names the pods are scheduled on. Returns `""` while pods are Pending (not yet assigned to a node).

```yaml
- path: podNodes
  value: "{{ podNodes .children.deployment }}"
# → "node-1, node-2"
```

---

### `podCount`

Return the total number of pods as `int`.

```yaml
- path: podCount
  value: "{{ podCount .children.deployment }}"
# → 3
```

---

### `readyPodCount`

Return the number of pods whose `Ready` condition is `True`.

```yaml
- path: readyPods
  value: "{{ readyPodCount .children.deployment }}"
# → 2
```

---

### `podMaxRestarts`

Return the highest restart count across all pods as `int64`. Zero when no pods are present or none have restarted. Pairs with `hasCrashingPod` for full crash-loop visibility.

```yaml
- path: maxRestarts
  value: "{{ podMaxRestarts .children.deployment }}"
# → 3
```

---

### `hasCrashingPod`

Return `true` when any pod has restarted more than twice — the first declarative signal of a crash loop.

```yaml
when:
  - field: "{{ hasCrashingPod .children.deployment }}"
    equals: "false"

status:
  fields:
    - path: crashDetected
      value: "{{ hasCrashingPod .children.deployment }}"
    - path: maxRestarts
      value: "{{ podMaxRestarts .children.deployment }}"
```

---

### `podByOrdinal`

Return the pod summary map at the given ordinal index. Designed for StatefulSets whose pods are named `<name>-0`, `<name>-1`, etc. Returns `nil` when no pod with that ordinal exists.

```yaml
# Surface the primary StatefulSet member:
- path: primaryPod
  value: "{{ (podByOrdinal .children.statefulset 0).name }}"
- path: primaryIP
  value: "{{ (podByOrdinal .children.statefulset 0).ip }}"

# All fields available on the returned map:
# .name, .ip, .phase, .ready, .node, .restartCount, .ordinal
```

---

## Zero-code memcached operator

```yaml
spec:
  crds:
    memcached:
      enrich: [pods]
      operatorBox:
        default: true
        onCreate:
          deployments:
            - name: "{{ .metadata.name }}"
              image: "memcache:{{ .spec.version }}"
              replicas: "{{ .spec.size }}"

        status:
          fields:
            - path: pods
              value: "{{ podNames .children.deployment }}"
            - path: podIPs
              value: "{{ podIPs .children.deployment }}"
            - path: size
              value: "{{ podCount .children.deployment }}"
            - path: ready
              value: "{{ readyPodCount .children.deployment }}"
```

`kubectl get memcached <name> -o yaml` shows `status.pods` populated with running pod names — no hooks, no Go code.

---

## Quick reference

| Note | Signature | Returns |
|------|-----------|---------|
| `podNames` | `(obj any)` | `string` |
| `podIPs` | `(obj any)` | `string` |
| `podPhases` | `(obj any)` | `string` |
| `podNodes` | `(obj any)` | `string` |
| `podCount` | `(obj any)` | `int` |
| `readyPodCount` | `(obj any)` | `int` |
| `podMaxRestarts` | `(obj any)` | `int64` |
| `hasCrashingPod` | `(obj any)` | `bool` |
| `podByOrdinal` | `(obj any, ordinal int64)` | `any` |

Requires `enrich: [pods]` or `enrichAll: true` on the CRD.

Each pod in `_pods` carries: `name`, `ip`, `phase`, `ready`, `node`, `restartCount`, `ordinal` (-1 for non-StatefulSet pods).

---

**Next →** [18 — Semver Notes](18-semver.md)
