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
| `podCount` | `(obj any)` | `int` |
| `readyPodCount` | `(obj any)` | `int` |
| `hasCrashingPod` | `(obj any)` | `bool` |

Requires `enrich: [pods]` or `enrichAll: true` on the CRD.

---

**Next →** [18 — Semver Notes](18-semver.md)
