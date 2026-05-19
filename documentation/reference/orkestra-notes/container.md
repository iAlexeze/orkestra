# Container Notes

Inspect containers within a Deployment, StatefulSet, or DaemonSet pod spec (`.spec.template.spec.containers`).

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `containerImage` | `obj, index int` | `string` — image of containers[index] |
| `containerEnv` | `obj, index int, key string` | `string` — env var value, `""` if not found |
| `containerPort` | `obj, index int, port int` | `bool` — true if port is declared |
| `containerPortByName` | `obj, index int, name string` | `int` — port number, `0` if not found |
| `containerPorts` | `obj, index int` | `string` — comma-separated port numbers |
| `containerCount` | `obj` | `int` — total container count |

All notes use index 0 for the primary (first) container.

## Examples

```yaml
# Expose the primary container's image in status
- path: currentImage
  value: "{{ containerImage .children.deployment 0 }}"

# Read an env var from the running container
- path: appEnv
  value: "{{ containerEnv .children.deployment 0 \"APP_ENV\" }}"

# Named port lookup — for Ingress or NetworkPolicy construction
- path: httpPort
  value: "{{ containerPortByName .children.deployment 0 \"http\" }}"

# Gate on port presence
when:
  - field: "{{ containerPort .children.deployment 0 8080 }}"
    equals: "true"

# Surface all exposed ports
- path: exposedPorts
  value: "{{ containerPorts .children.deployment 0 }}"

# Multi-container: count sidecar containers
- path: sidecarCount
  value: "{{ sub (containerCount .children.deployment) 1 }}"
```
