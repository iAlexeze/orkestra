# 01 — Profile reference

## Resource profiles

Set via `resources.profile` on any deployment, statefulset, job, or cronjob.

| Profile | CPU request | CPU limit | Memory request | Memory limit | Use for |
|---------|-------------|-----------|----------------|--------------|---------|
| `tiny` | 25m | 100m | 64Mi | 128Mi | Sidecars, lightweight agents |
| `small` | 100m | 500m | 128Mi | 512Mi | Low-traffic APIs, CLI tools |
| `medium` | 250m | 1 | 256Mi | 1Gi | Standard web services |
| `large` | 500m | 2 | 512Mi | 2Gi | Data-intensive services |
| `burst` | 200m | 2 | 256Mi | 2Gi | Spiky workloads — wide request/limit spread |
| `steady` | 300m | 600m | 256Mi | 512Mi | Stable workloads — tight spread |
| `compute-heavy` | 1 | 2 | 512Mi | 1Gi | CPU-bound workers |
| `memory-heavy` | 250m | 500m | 1Gi | 2Gi | Caches, JVM apps |

---

## Security profiles

### Container security (`securityContext.profile`)

| Profile | AllowPrivilegeEscalation | RunAsNonRoot | ReadOnlyRootFilesystem | Capabilities |
|---------|--------------------------|--------------|------------------------|--------------|
| `baseline` | false | — | — | Drop: NET_RAW |
| `restricted` | false | true | — | Drop: ALL |
| `hardened` | false | true | true | Drop: ALL |

`restricted` matches the Kubernetes [restricted Pod Security Standard](https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted).

### Pod security (`podSecurity.profile`)

| Profile | RunAsNonRoot | RunAsUser | RunAsGroup | FSGroup |
|---------|-------------|-----------|------------|---------|
| `baseline` | false | — | — | — |
| `restricted` | true | 1000 | — | — |
| `hardened` | true | 65534 | 65534 | 65534 |

---

## Probe profiles

Set via `probes.liveness.profile`, `probes.readiness.profile`, or `probes.startup.profile`.

| Profile | InitialDelay | Period | FailureThreshold | Timeout | Use for |
|---------|-------------|--------|-----------------|---------|---------|
| `fast` | 5s | 10s | 2 | 5s | HTTP APIs that start in seconds |
| `standard` | 15s | 20s | 3 | 10s | Most web services (default) |
| `patient` | 30s | 30s | 5 | 10s | Batch workers, slow-starting services |
| `slow-start` | 0s | 10s | 30 | 10s | JVM apps, databases (5-min startup window) |

`DefaultProbeTimings` matches `standard`.

---

## Autoscale profiles

Set via `autoscale.profile`. Expands into a complete `autoscale` block using the CRD's declared `workers` and `queue.maxDepth` as the baseline.

| Profile | Trigger | Scale up | Interval | Cooldown | Use for |
|---------|---------|----------|----------|----------|---------|
| `burst` | queue > 60% max | 4× workers, 10× queue | 5s | 30s | Spiky event ingestion |
| `steady` | queue > 40% max AND busy > 70% | 2× workers, 3× queue | 30s | 2m | Predictable background processing |
| `batch` | cron: 11 PM nightly | 3× workers, 8× queue | 60s | 5m | Nightly batch jobs |
| `latency-sensitive` | P95 reconcile > 200ms | 2.5× workers | 15s | 1m | Real-time pipelines |
| `cost-optimized` | idle > 60% AND queue > 80% max | 0.5× workers, 0.5× queue | 30s | 10m | Reduces capacity when underloaded |

---

## HPA behavior profiles

Set via `behavior.profile` on any HPA resource. Expands into a complete `HorizontalPodAutoscalerBehavior` block (scaleUp + scaleDown policies and stabilization windows) and sets a suggested `targetCPUUtilizationPercentage` when none is declared explicitly.

| Profile | CPU target | Scale-up window | Scale-up policy | Scale-down window | Scale-down policy | Use for |
|---------|-----------|-----------------|-----------------|-------------------|-------------------|---------|
| `web` | 70% | 0s | Max(100%/15s, 4 pods/15s) | 300s | 10%/60s | Frontend web services, moderate traffic |
| `api` | 60% | 0s | Max(100%/15s, 4 pods/15s) | 600s | 5%/60s | Backend APIs requiring stable instance pools |
| `latency-sensitive` | 50% | 0s | Max(200%/15s, 10 pods/15s) | 900s | 5%/120s | Real-time services where cold-start latency matters |
| `batch` | 80% | 30s | 100%/60s | 120s | 50%/60s | Batch jobs that scale fast and release quickly |
| `cost-optimized` | 80% | 180s | 25%/60s (Min) | 60s | 50%/60s | Workloads where over-provisioning cost is the priority |

**Scale-up window (stabilization)**: 0s means the HPA acts on the very first signal — fastest possible reaction. Higher values debounce scale-up decisions.

**Scale-down window**: How long the HPA waits after a sustained drop before removing pods. Longer windows protect against oscillation at the cost of slower scale-in.

**SelectPolicy Max/Min**: `Max` picks the policy that adds the most pods; `Min` picks the one that adds the fewest. Scale-down uses `Min` to be conservative; `cost-optimized` scale-up uses `Min` to grow slowly.

→ Next: [02-internals.md](02-internals.md)
