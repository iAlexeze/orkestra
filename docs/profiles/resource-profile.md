# Resource Profile
*CPU and memory presets for workloads.*

A resource profile is a named preset that expands into a complete Kubernetes `resources` block — requests and limits — at Katalog load time.

You write a name. Orkestra writes the numbers.

---

## Profiles

| Profile | CPU Request | CPU Limit | Memory Request | Memory Limit | Use Case |
|---|---|---|---|---|---|
| `tiny` | 25m | 100m | 64Mi | 128Mi | Sidecars, health endpoints, minimal utilities |
| `small` | 100m | 500m | 128Mi | 512Mi | Standard microservices |
| `medium` | 250m | 1 | 256Mi | 1Gi | Production web services |
| `large` | 500m | 2 | 512Mi | 2Gi | High-traffic or heavier workloads |
| `burst` | 200m | 2 | 256Mi | 2Gi | Services with sudden spikes; low request, high limit |
| `steady` | 300m | 600m | 256Mi | 512Mi | Predictable, stable workloads |
| `compute-heavy` | 1 | 2 | 512Mi | 1Gi | CPU-bound work — builds, ML inference, data pipelines |
| `memory-heavy` | 250m | 500m | 1Gi | 2Gi | Memory-bound work — JVMs, large caches, analytics |

---

## Usage

Set `resources.profile` on any Deployment, StatefulSet, ReplicaSet, or Pod:

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}-api"
      image: "{{ .spec.image }}"
      resources:
        profile: small
```

Or dynamically from the CR:

```yaml
resources:
  profile: '{{ .spec.resourceProfile | default "small" }}'
```

---

## Rules

**Profile or explicit — not both.**  
`resources.profile` cannot coexist with `resources.requests` or `resources.limits` on the same resource. If both are present, Orkestra rejects the Katalog at load time.

```yaml
# Valid — profile only
resources:
  profile: burst

# Valid — explicit only
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2
    memory: 2Gi

# Invalid — profile and explicit together
resources:
  profile: burst
  requests:
    cpu: 500m  # error: cannot mix profile and explicit fields
```

**Unknown profiles fail fast.**  
A typo in a profile name (`buurst`, `Large`) is a Katalog load error. You will see the error before the operator starts.

**Template expressions are allowed.**  
When `resourceProfile` is a template expression, the profile name is resolved at reconcile time and validated then. Unknown names at runtime cause a reconcile failure on that CR — other CRs are not affected.

---

## When to use each profile

**tiny** — only when you genuinely need minimal overhead. A health check sidecar. A token rotator that runs once per hour.

**small** — your default for anything that does not have unusual resource behavior. Start here.

**medium** — when `small` limits are hit in practice. Most production web services.

**large** — services with consistently high load. Not a catch-all for uncertainty.

**burst** — when your service is mostly idle but needs to absorb sudden spikes without being throttled. The wide gap between requests and limits is intentional.

**steady** — when burst is overkill but `small` feels too tight. Predictable workloads where you want the limit to be close to the request.

**compute-heavy** — batch jobs, build pipelines, ML inference, anything that maxes out CPU cores. Memory stays low.

**memory-heavy** — JVM services, large in-process caches, analytics engines. CPU stays moderate; memory is generous.

---

**Next →** [Autoscale Profile](./autoscale-profile.md)
