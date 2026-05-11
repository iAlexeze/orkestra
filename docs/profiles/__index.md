# Profiles

Profiles are named presets that expand into a fully-formed configuration at Katalog load time.

You declare a profile name. Orkestra resolves it into the concrete values — CPU limits, worker counts, probe timings — before the runtime starts. By the time a reconcile loop runs, there is no profile. There is only a fully-expanded spec, exactly as if you had written it by hand.

---

## Why profiles exist

Writing raw Kubernetes configuration is correct but carries no intent.

```yaml
# What does this tell you?
resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 2
    memory: 2Gi
```

A profile captures the **reason** the numbers are what they are.

```yaml
resources:
  profile: burst
```

`burst` tells you — and the operator — that this workload handles sudden spikes. The numbers follow from that decision, not the other way around.

The same logic applies to autoscaler behavior and probe timing. Profiles encode intent. Intent survives change.

---

## How profiles work

Every profile family follows the same pattern:

1. **Declare** — write a profile name in the Katalog
2. **Validate** — Orkestra checks the name at load time; unknown profiles fail fast
3. **Expand** — Orkestra replaces the profile name with the full configuration
4. **Run** — the runtime sees the expanded spec; the profile name is gone

Profiles can be static or dynamic:

```yaml
# Static — the profile is always "steady"
resources:
  profile: steady

# Dynamic — resolved from the CR at reconcile time
resources:
  profile: "{{ .spec.resourceProfile | default \"standard\" }}"
```

Both are valid. Template expressions are validated at runtime. Static names are validated at load time.

---

## The three profile families

### Resource profiles
*What CPU and memory this workload gets.*

Named presets for Kubernetes `resources.requests` and `resources.limits`.  
Applied to any Deployment, StatefulSet, ReplicaSet, or Pod.

```yaml
resources:
  profile: small
```

→ [Resource Profile](./resource-profile.md)

---

### Autoscale profiles
*How this operator scales its own workers and queue.*

Named presets for `operatorBox.autoscale` — the operator's internal autoscaler.  
Expand relative to the operator's declared baseline.

```yaml
autoscale:
  profile: latency-sensitive
```

→ [Autoscale Profile](./autoscale-profile.md)

---

### Probe profiles
*How long Kubernetes waits before acting on probe results.*

Named presets for startup, liveness, and readiness probe timing.  
Applied to any Deployment, StatefulSet, ReplicaSet, or Pod.

```yaml
probes:
  startup:
    type: tcp
    profile: slow-start
  liveness:
    type: http
    path: /health
    profile: standard
```

→ [Probe Profile](./probe-profile.md)

---

## Rules that apply to every profile

- **Atomic** — a profile fully defines its configuration. You cannot combine a profile with manual fields of the same type.
- **Fail-fast** — an unknown profile name is a Katalog load error, not a runtime error.
- **No mixing** — a resource profile and manual `resources:` fields cannot coexist on the same resource.
- **Template-safe** — profile names can be template expressions. Static names are validated immediately. Template expressions are validated when the CR is reconciled.

---

**Next →** [Resource Profile](./resource-profile.md)
