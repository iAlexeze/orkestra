# Probe Profile

A probe profile is a named preset for Kubernetes health probe timing: `initialDelaySeconds`, `periodSeconds`, `failureThreshold`, `successThreshold`, and `timeoutSeconds`.

Probes have two concerns: **what to check** (HTTP path or TCP port) and **when to give up** (timing). A probe profile handles the timing so you only declare the what.

---

## Profiles

| Profile | initialDelay | period | failureThreshold | timeout | Max window |
|---------|-------------|--------|-----------------|---------|------------|
| `fast` | 5s | 10s | 2 | 5s | ~25s |
| `standard` | 15s | 20s | 3 | 10s | ~75s |
| `patient` | 30s | 30s | 5 | 10s | ~180s |
| `slow-start` | 0s | 10s | 30 | 10s | 300s |

**Max window** is the maximum time before Kubernetes declares the probe failed.

---

## Usage

Set `probes:` on any Deployment, StatefulSet, ReplicaSet, or Pod:

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}-api"
      image: "{{ .spec.image }}"
      port: "8080"
      probes:
        startup:
          type: http
          path: /health
          profile: slow-start
        liveness:
          type: http
          path: /health
          profile: standard
        readiness:
          type: http
          path: /ready
          profile: standard
```

For TCP services:

```yaml
probes:
  startup:
    type: tcp
    profile: slow-start
  liveness:
    type: tcp
    profile: standard
  readiness:
    type: tcp
    profile: standard
```

When `type` is omitted and `path` is set, HTTP is assumed.

---

## Rules

**Profile or explicit — not both.**

A probe profile cannot coexist with explicit timing fields on the same probe.

```yaml
# Valid
liveness:
  type: http
  path: /health
  profile: standard

# Valid
liveness:
  type: http
  path: /health
  initialDelaySeconds: 15
  periodSeconds: 20
  failureThreshold: 3

# Invalid — rejected at load time
liveness:
  type: http
  path: /health
  profile: standard
  initialDelaySeconds: 5
```

**Unknown profiles fail fast.** An unrecognized name is a Katalog load error.

**Port defaults to the container's declared port.** Override with `port:` on the probe when the probe target differs.

---

## Profile details

### `fast`

Quick detection for services that start instantly. Fails after ~25 seconds.

```text
initialDelaySeconds: 5
periodSeconds:       10
failureThreshold:    2
timeout:             5s
```

Use for stateless HTTP services where startup is fast and problems should be caught immediately. Good for readiness probes on APIs.

### `standard`

Balanced defaults for most services. Fails after ~75 seconds.

```text
initialDelaySeconds: 15
periodSeconds:       20
failureThreshold:    3
timeout:             10s
```

The right default when you have no specific information about the workload.

### `patient`

Tolerant of slower operations. Fails after ~180 seconds.

```text
initialDelaySeconds: 30
periodSeconds:       30
failureThreshold:    5
timeout:             10s
```

Use for batch workers, services with non-trivial initialization, or services that do expensive startup work.

### `slow-start`

5-minute tolerance window for heavy startup. Designed for startup probes only.

```text
initialDelaySeconds: 0
periodSeconds:       10
failureThreshold:    30
timeout:             10s
```

Use for startup probes on databases loading large datasets, JVM services with slow class loading, or Kafka brokers completing leader election after a restart. Once the startup probe passes, Kubernetes hands control to liveness and readiness probes.

---

## Startup vs liveness vs readiness

| Probe | Purpose | Failure action |
|-------|---------|----------------|
| `startup` | Is the container done initializing? | Restart |
| `liveness` | Is the container still alive? | Restart |
| `readiness` | Should this container receive traffic? | Remove from Service endpoints |

**HTTP API pattern:**

```yaml
probes:
  startup:
    type: http
    path: /health
    profile: patient       # time for app startup
  liveness:
    type: http
    path: /health
    profile: standard      # restart if health degrades
  readiness:
    type: http
    path: /ready
    profile: fast          # pull traffic quickly when ready
```

**Database pattern:**

```yaml
probes:
  startup:
    type: tcp
    profile: slow-start    # generous window for first boot
  liveness:
    type: tcp
    profile: standard      # restart if frozen
  readiness:
    type: tcp
    profile: standard      # stop routing if not ready
```

---

## Choosing a profile

| Situation | Profile |
|-----------|---------|
| Startup finishes in under 10 seconds | `fast` |
| Standard web service | `standard` |
| Startup takes up to 2–3 minutes | `patient` |
| Database, JVM, Kafka — startup may need 5 minutes | `slow-start` (startup probe only) |

Use `slow-start` for startup probes only. Liveness and readiness probes should use `standard` or `patient` — a slow liveness probe means problems are detected slowly.
