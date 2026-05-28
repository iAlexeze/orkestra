# Resource Profile

A resource profile is a named preset that expands into a complete Kubernetes `resources` block — requests and limits — at Katalog load time.

You write a name. Orkestra writes the numbers.

---

## Profiles

| Profile | CPU Request | CPU Limit | Memory Request | Memory Limit | Use case |
|---------|-------------|-----------|----------------|--------------|----------|
| `tiny` | 25m | 100m | 64Mi | 128Mi | Sidecars, health endpoints, minimal utilities |
| `small` | 100m | 500m | 128Mi | 512Mi | Standard microservices (start here) |
| `medium` | 250m | 1 | 256Mi | 1Gi | Production web services |
| `large` | 500m | 2 | 512Mi | 2Gi | High-traffic or heavier workloads |
| `burst` | 200m | 2 | 256Mi | 2Gi | Services with sudden spikes — low request, high limit |
| `steady` | 300m | 600m | 256Mi | 512Mi | Predictable, stable workloads |
| `compute-heavy` | 1 | 2 | 512Mi | 1Gi | CPU-bound work — builds, ML inference, data pipelines |
| `memory-heavy` | 250m | 500m | 1Gi | 2Gi | Memory-bound work — JVMs, large caches, analytics |

---

## Usage

Set `resources.profile` on any Deployment, StatefulSet, ReplicaSet, Pod, Job, or CronJob:

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

`resources.profile` cannot coexist with `resources.requests` or `resources.limits` on the same resource. Orkestra rejects the Katalog at load time if both are present.

```yaml
# Valid
resources:
  profile: burst

# Valid
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2
    memory: 2Gi

# Invalid — rejected at load time
resources:
  profile: burst
  requests:
    cpu: 500m
```

**Unknown profiles fail fast.** A typo (`buurst`, `Large`) is a Katalog load error.

**Template expressions are allowed.** When the profile is a template expression, the name is resolved at reconcile time and validated then. An unknown name at runtime causes a reconcile failure on that CR only — other CRs are not affected.

---

## Choosing a profile

| Situation | Profile |
|-----------|---------|
| Not sure — default for new services | `small` |
| Small hits production, limits reached | `medium` |
| Heavy but predictable load | `large` |
| Mostly idle, occasional spikes | `burst` |
| Steady throughput, tight cost control | `steady` |
| Sidecar, health check, token rotator | `tiny` |
| Build jobs, ML inference, data pipelines | `compute-heavy` |
| JVM, large in-memory cache, analytics | `memory-heavy` |
