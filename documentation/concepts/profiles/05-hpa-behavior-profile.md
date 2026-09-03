# HPA Behavior Profile

An HPA behavior profile is a named preset that expands into a complete `HorizontalPodAutoscalerBehavior` block — scale-up and scale-down policies, stabilization windows, and a suggested CPU utilization target — at Katalog load time.

You write a name. Orkestra writes the behavior.

---

## Profiles

| Profile | CPU target | Scale-up window | Scale-up policy | Scale-down window | Scale-down policy | Use for |
|---------|-----------|-----------------|-----------------|-------------------|-------------------|---------|
| `web` | 70% | 0s | Max(100%/15s, 4 pods/15s) | 300s | 10%/60s (Min) | Frontend web services |
| `api` | 60% | 0s | Max(100%/15s, 4 pods/15s) | 600s | 5%/60s (Min) | Backend APIs |
| `latency-sensitive` | 50% | 0s | Max(200%/15s, 10 pods/15s) | 900s | 5%/120s (Min) | Real-time services |
| `batch` | 80% | 30s | 100%/60s (Max) | 120s | 50%/60s (Min) | Batch jobs |
| `cost-optimized` | 80% | 180s | 25%/60s (Min) | 60s | 50%/60s (Max) | Cost-sensitive workloads |

**Scale-up window**: Time the HPA waits after first seeing high utilization before adding pods. `0s` means instant reaction.

**Scale-down window**: Time the HPA waits after a sustained drop before removing pods. Longer windows prevent oscillation.

**SelectPolicy Max/Min**: `Max` scales to the policy that adds the most pods (fastest). `Min` is more conservative. Scale-down defaults to `Min` to avoid premature removal. `cost-optimized` uses `Min` on scale-up to grow slowly.

---

## Usage

Set `behavior.profile` on any HPA resource:

```yaml
onCreate:
  hpa:
    - name: "{{ .metadata.name }}-hpa"
      scaleTargetRef:
        apiVersion: apps/v1
        kind: Deployment
        name: "{{ .metadata.name }}"
      minReplicas: "2"
      maxReplicas: "20"
      behavior:
        profile: web
```

The profile sets both the `behavior` block and `targetCPUUtilizationPercentage`. To override the CPU target while keeping the scaling behavior, declare `targetCPUUtilizationPercentage` explicitly:

```yaml
hpa:
  - name: "{{ .metadata.name }}-hpa"
    scaleTargetRef:
      apiVersion: apps/v1
      kind: Deployment
      name: "{{ .metadata.name }}"
    minReplicas: "2"
    maxReplicas: "20"
    targetCPUUtilizationPercentage: "55"  # overrides the profile's default
    behavior:
      profile: api
```

---

## Rules

**Profile or explicit — not both.**

`behavior.profile` cannot coexist with `behavior.scaleUp` or `behavior.scaleDown`. Orkestra rejects the Katalog at load time if both are present.

```yaml
# Valid
behavior:
  profile: api

# Valid
behavior:
  scaleUp:
    stabilizationWindowSeconds: 0
    selectPolicy: Max
    policies:
      - type: Percent
        value: 100
        periodSeconds: 15
  scaleDown:
    stabilizationWindowSeconds: 300
    selectPolicy: Min
    policies:
      - type: Percent
        value: 10
        periodSeconds: 60

# Invalid — rejected at load time
behavior:
  profile: api
  scaleDown:
    stabilizationWindowSeconds: 900
```

**Unknown profiles fail fast.** A typo (`webb`, `Api`) is a Katalog load error.

**CPU target is independent.** The profile's CPU target applies only when `targetCPUUtilizationPercentage` is not set. Declaring both is valid and the explicit value takes precedence.

---

## Choosing a profile

| Situation | Profile |
|-----------|---------|
| User-facing web app, tolerates brief spikes | `web` |
| Backend API serving other services, needs stable pool | `api` |
| Real-time pipeline, every cold-start adds latency | `latency-sensitive` |
| Nightly or periodic batch job, releases resources fast | `batch` |
| Internal tool or dev environment where cost matters | `cost-optimized` |
