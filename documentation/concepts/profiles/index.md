# Profiles

Profiles are named presets that expand into a fully-formed configuration at Katalog load time.

You declare a profile name. Orkestra resolves it into the concrete values — CPU limits, worker counts, probe timings, security settings — before the runtime starts. By the time a reconcile loop runs, there is no profile. There is only a fully-expanded spec, exactly as if you had written it by hand.

---

## Why profiles exist

Most teams get Kubernetes resource configuration wrong — not because they don't care, but because the right numbers require production data they don't have yet. So they guess. They copy from Stack Overflow. They set limits too low and get OOMKilled at 2am, or too high and waste half the node. Probe timeouts are set once and never revisited. Security contexts are left empty because figuring out the right capabilities takes time no one has.

Profiles are a decision made once by someone who thought it through, shared with everyone who shouldn't have to think about it again.

Writing raw Kubernetes configuration is correct but carries no intent:

```yaml
resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 2
    memory: 2Gi
```

A profile captures the **reason** the numbers are what they are:

```yaml
resources:
  profile: burst
```

`burst` tells you — and the next engineer who reads this Katalog — that this workload handles sudden spikes. The numbers follow from that decision, not the other way around.

The same logic applies to probe timing and security posture. Profiles encode intent. Intent survives change.

---

## How profiles work

Every profile family follows the same pattern:

1. **Declare** — write a profile name in the Katalog
2. **Validate** — Orkestra checks the name at load time; unknown profiles fail fast
3. **Expand** — Orkestra replaces the profile name with the full configuration
4. **Run** — the runtime sees the expanded spec; the profile name is gone

Profiles can be static or dynamic:

```yaml
# Static — always "steady"
resources:
  profile: steady

# Dynamic — resolved from the CR at reconcile time
resources:
  profile: "{{ .spec.resourceProfile | default \"small\" }}"
```

Static names are validated at load time. Template expressions are validated when the CR is reconciled.

---

## Profile families

| Family | What it controls | Applied to |
|--------|-----------------|------------|
| [Autoscale](./01-autoscale-profile.md) | Operator worker scaling | `operatorBox.autoscale` |
| [Resource](./02-resource-profile.md) | CPU and memory requests/limits | Deployment, StatefulSet, ReplicaSet, Pod, Job, CronJob |
| [Probe](./03-probe-profile.md) | Health check timing | Deployment, StatefulSet, ReplicaSet, Pod |
| [Security](./04-security-profile.md) | Container and pod security contexts | Deployment, StatefulSet, ReplicaSet, Pod, Job, CronJob |
| [HPA Behavior](./05-hpa-behavior-profile.md) | Kubernetes HPA scale-up/down policies | `hpa[*].behavior` |
| [PDB Behavior](./06-pdb-profile.md) | PodDisruptionBudget disruption limits | `pdb[*].behavior` |
| [Rolling Update](./07-rolling-update-profile.md) | Deployment/StatefulSet/ReplicaSet rollout strategy | `deployments[*].rollingUpdate`, `statefulSets[*].rollingUpdate`, `replicaSets[*].rollingUpdate` |
| [ResourceQuota](./08-resourcequota-profile.md) | Namespace resource limits (pods, CPU, memory) | `resourceQuotas[*]` |
| [NetworkPolicy](./09-networkpolicy-profile.md) | Traffic allow/deny rules for pods | `networkPolicies[*]` |

---

## Rules that apply to every profile

- **Atomic** — a profile fully defines its configuration. You cannot combine a profile with manual fields of the same type.
- **Fail-fast** — an unknown profile name is a Katalog load error, not a runtime error.
- **No mixing** — a profile and explicit fields of the same type cannot coexist on the same resource.
- **Template-safe** — profile names can be template expressions. Static names are validated immediately; template expressions are validated at reconcile time.
