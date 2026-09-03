# Profiles

Profiles are named presets that expand into fully-formed configuration at Katalog load time. The runtime never sees a profile name — by the time a Deployment is created, the profile has already been replaced by concrete values.

```yaml
deployments:
  - name: "{{ .metadata.name }}"
    image: "{{ .spec.image }}"
    resources:
      profile: medium       # cpu:250m/1, memory:256Mi/1Gi
    securityContext:
      profile: restricted   # allowPrivilegeEscalation:false, runAsNonRoot:true, drop:ALL
    probes:
      liveness:
        type: http
        path: /healthz
        profile: standard   # initialDelay:15s, period:20s, failureThreshold:3
```

Three lines. No resource arithmetic. No security context boilerplate. No probe timing lookup.

---

## The five profile kinds

### Resource profiles

`resources.profile` on any Deployment, StatefulSet, Job, or CronJob.

Named presets for CPU and memory requests/limits. Pick the shape that matches your workload; skip the capacity planning for individual fields.

| Profile | CPU request | CPU limit | Memory request | Memory limit |
|---|---|---|---|---|
| `tiny` | 25m | 100m | 64Mi | 128Mi |
| `small` | 100m | 500m | 128Mi | 512Mi |
| `medium` | 250m | 1 | 256Mi | 1Gi |
| `large` | 500m | 2 | 512Mi | 2Gi |
| `burst` | 200m | 2 | 256Mi | 2Gi |
| `steady` | 300m | 600m | 256Mi | 512Mi |
| `compute-heavy` | 1 | 2 | 512Mi | 1Gi |
| `memory-heavy` | 250m | 500m | 1Gi | 2Gi |

**Try it:**
```bash
ork init --pack use-cases
cd profiles/01-resource
ork run
kubectl apply -f ../cr.yaml
```

---

### Security profiles

`securityContext.profile` (container) and `podSecurity.profile` (pod).

Named presets for Kubernetes security contexts. `restricted` matches the Kubernetes restricted Pod Security Standard. `hardened` adds a read-only root filesystem and runs as a non-root, non-privileged user.

| Profile | Container | Pod |
|---|---|---|
| `baseline` | Drop NET_RAW | — |
| `restricted` | Drop ALL, runAsNonRoot | runAsNonRoot, runAsUser:1000 |
| `hardened` | Drop ALL, runAsNonRoot, readOnlyRootFilesystem | runAsNonRoot, runAsUser:65534, fsGroup:65534 |

**Try it:**
```bash
cd profiles/02-security
ork run
kubectl apply -f ../cr.yaml
```

---

### Probe profiles

`probes.liveness.profile`, `probes.readiness.profile`, `probes.startup.profile`.

Named timing presets for Kubernetes probes. `slow-start` is designed specifically for startup probes on JVM apps and databases — 30 failures × 10s period = 5-minute startup window before Kubernetes gives up.

| Profile | InitialDelay | Period | FailureThreshold | Timeout |
|---|---|---|---|---|
| `fast` | 5s | 10s | 2 | 5s |
| `standard` | 15s | 20s | 3 | 10s |
| `patient` | 30s | 30s | 5 | 10s |
| `slow-start` | 0s | 10s | 30 | 10s |

**Try it:**
```bash
cd profiles/03-probes
ork run
kubectl apply -f ../cr.yaml
```

---

### Rolling update profiles

`rollingUpdate.profile` on any Deployment.

Named presets for `maxSurge` and `maxUnavailable`. `safe` ensures zero capacity drop during rollouts. `blue-green` doubles capacity temporarily for the cleanest cutover.

| Profile | maxSurge | maxUnavailable |
|---|---|---|
| `safe` | 1 | 0 |
| `fast` | 25% | 25% |
| `blue-green` | 100% | 0 |

**Try it:**
```bash
cd profiles/04-rolling-update
ork run
kubectl apply -f ../cr.yaml
# Then trigger a rollout:
kubectl patch service my-service --type=merge -p '{"spec":{"image":"nginx:1.26"}}'
```

---

### PDB profiles

`pdb.behavior.profile` on any PodDisruptionBudget.

Named presets for voluntary disruption limits. `zero-downtime` prevents any pod from being evicted. `rolling` allows exactly one at a time. `relaxed` allows 25%.

| Profile | Setting | Value |
|---|---|---|
| `zero-downtime` | minAvailable | 100% |
| `rolling` | maxUnavailable | 1 |
| `relaxed` | maxUnavailable | 25% |

**Try it:**
```bash
cd profiles/05-pdb
ork run
kubectl apply -f ../cr.yaml
```

---

### Autoscale profiles

`autoscale.profile` on a CRD entry.

Named presets for Orkestra's declarative autoscaler — expands into trigger thresholds, scale-up/down policies, intervals, and cooldowns. The autoscale profile is distinct from the others: it configures the operatorBox runtime, not child Kubernetes resources.

**Try it:**
```bash
ork init --pack advanced
cd 12-autoscale
ork run
```

---

## Profiles vs explicit fields

Profiles and explicit field declarations are mutually exclusive — you can use one or the other, not both on the same resource.

```yaml
# Profile — one word, expands to full config
resources:
  profile: medium

# Explicit — full control, no expansion
resources:
  requests:
    cpu: "250m"
    memory: "256Mi"
  limits:
    cpu: "1"
    memory: "1Gi"
```

Use profiles when the preset matches your workload. Use explicit fields when you need values the presets don't cover.

---

## Try it

```bash
ork init --pack use-cases
cd profiles
```

---

**Next →** [Reconcile Pipeline](../01-reconcile-pipeline.md)
