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

Set via `autoscale.profile`. Expands into a complete `autoscale` block using the CRD's declared `workers` and `queue.maxQueueDepth` as the baseline.

| Profile | Trigger | Scale up | Interval | Cooldown | Use for |
|---------|---------|----------|----------|----------|---------|
| `burst` | queue > 60% max | 4× workers, 10× queue | 5s | 30s | Spiky event ingestion |
| `steady` | queue > 40% max AND busy > 70% | 2× workers, 3× queue | 30s | 2m | Predictable background processing |
| `batch` | cron: 11 PM nightly | 3× workers, 8× queue | 60s | 5m | Nightly batch jobs |
| `latency-sensitive` | P95 reconcile > 200ms | 2.5× workers | 15s | 1m | Real-time pipelines |
| `cost-optimized` | idle > 60% AND queue > 80% max | 0.5× workers, 0.5× queue | 30s | 10m | Reduces capacity when underloaded |

→ Next: [02-internals.md](02-internals.md)
