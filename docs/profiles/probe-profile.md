# Probe Profile
*Health check timing presets — how long Kubernetes waits.*

A probe profile is a named preset for the timing parameters of a Kubernetes health probe: `initialDelaySeconds`, `periodSeconds`, `failureThreshold`, `successThreshold`, and `timeoutSeconds`.

Probes have two concerns: **what to check** (HTTP path or TCP port) and **when to give up** (timing). A probe profile handles the second one so you only have to declare the first.

---

## Profiles

| Profile | initialDelay | period | failureThreshold | successThreshold | timeout | Window |
|---|---|---|---|---|---|---|
| `fast` | 5s | 10s | 2 | 1 | 5s | ~25s max |
| `standard` | 15s | 20s | 3 | 1 | 10s | ~75s max |
| `patient` | 30s | 30s | 5 | 1 | 10s | ~180s max |
| `slow-start` | 0s | 10s | 30 | 1 | 10s | 300s max |

**Window** is the maximum time before Kubernetes declares the probe failed and restarts or stops routing traffic to the container.

---

## Probe types

Probe profiles work with both probe types:

### HTTP GET
Sends an HTTP GET to the declared path. Succeeds on 2xx–3xx responses.

```yaml
probes:
  liveness:
    type: http
    path: /health
    profile: standard
```

### TCP socket
Opens a TCP connection to the container's port. Succeeds if the port accepts the connection.

```yaml
probes:
  liveness:
    type: tcp
    profile: standard
```

When `type` is omitted and `path` is set, HTTP is assumed.

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

For databases and other TCP services:

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

---

## Rules

**Profile or explicit — not both.**  
A probe profile cannot coexist with explicit timing fields (`initialDelaySeconds`, `periodSeconds`, `failureThreshold`, `successThreshold`, `timeoutSeconds`) on the same probe. Use one or the other.

```yaml
# Valid — profile only
liveness:
  type: http
  path: /health
  profile: standard

# Valid — explicit only
liveness:
  type: http
  path: /health
  initialDelaySeconds: 15
  periodSeconds: 20
  failureThreshold: 3

# Invalid — profile and explicit together
liveness:
  type: http
  path: /health
  profile: standard
  initialDelaySeconds: 5  # error: cannot mix profile and explicit timing
```

**Unknown profiles fail fast.**  
An unrecognized profile name is a Katalog load error.

**Template expressions are allowed.**  
Profile names can be template expressions:

```yaml
profile: '{{ .spec.probeProfile | default "standard" }}'
```

Static names are validated at load time. Template expressions are validated at reconcile time.

**Port defaults to the container's declared port.**  
If the probe does not specify `port:`, it uses the container's `port:` field. Override with `port:` on the probe when needed.

---

## Profile details

### `fast`
*Quick detection for services that start instantly.*

Starts checking after 5 seconds, checks every 10 seconds, fails after 2 missed checks. Use for stateless HTTP services where startup is fast and problems should be caught immediately.

```
initialDelaySeconds: 5
periodSeconds:       10
failureThreshold:    2
timeout:             5s
```

---

### `standard`
*Balanced defaults for most services.*

Starts checking after 15 seconds, checks every 20 seconds, fails after 3 missed checks. This is the right default if you have no specific information about the workload.

```
initialDelaySeconds: 15
periodSeconds:       20
failureThreshold:    3
timeout:             10s
```

---

### `patient`
*Tolerant of slower operations.*

Starts checking after 30 seconds, checks every 30 seconds, fails after 5 missed checks — a total window of 180 seconds after initial delay. Use for batch workers, services with non-trivial initialization, or services that do expensive startup work.

```
initialDelaySeconds: 30
periodSeconds:       30
failureThreshold:    5
timeout:             10s
```

---

### `slow-start`
*5-minute window for heavy startup.*

Starts checking immediately (no initial delay), checks every 10 seconds, allows 30 consecutive failures before giving up — a 300-second (5-minute) tolerance window.

```
initialDelaySeconds: 0
periodSeconds:       10
failureThreshold:    30
timeout:             10s
```

Designed for **startup probes** on services with slow initialization: databases loading large datasets, JVM services with slow class loading, Kafka brokers completing leader election after a restart.

Once the startup probe passes, Kubernetes hands control to liveness and readiness probes.

---

## Startup vs liveness vs readiness

| Probe | Purpose | Failure action |
|---|---|---|
| `startup` | Is the container done initializing? | Restart |
| `liveness` | Is the container still alive? | Restart |
| `readiness` | Should this container receive traffic? | Remove from Service endpoints |

**A typical pattern for a database:**

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

**A typical pattern for an HTTP API:**

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
    path: /ready           # separate ready check
    profile: fast          # pull traffic fast when ready
```

---

## Choosing a profile

| Question | Profile |
|---|---|
| Does startup finish in under 10 seconds? | `fast` |
| Standard web service with normal startup? | `standard` |
| Does startup take up to 2-3 minutes? | `patient` |
| Database, JVM, or Kafka? Startup might need 5 minutes? | `slow-start` |

Use `slow-start` only for startup probes. Use `standard` or `patient` for liveness and readiness — a slow liveness probe means problems are detected slowly.
