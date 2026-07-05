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

## Built-ins versus user-defined profiles

Orkestra ships with built-in profiles — `deny-all`, `small/medium/large/xlarge`, `safe`, `zero-downtime`, `high-throughput/conservative/development` — so the feature works immediately without any extra YAML. They cover common Kubernetes patterns.

But the feature is designed for the profiles you write yourself.

A team writing `profile: org-medium` is expressing an organizational contract: this namespace gets the capacity we agreed on for medium-sized teams. The numbers follow from that decision. Change the `org-medium` definition once and it propagates to every Katalog and Motif that references it — no grep, no PR chain, no drift.

That is what built-ins cannot do. `medium` is Orkestra's guess at what medium means. `org-medium` is your team's actual answer.

User-defined profiles are declared in a `profiles:` block at the root of any Katalog or Motif:

```yaml
profiles:
  resourceQuotas:
    - name: org-medium
      description: Standard allocation for a medium-sized team namespace
      hard:
        pods: "25"
        cpu: "4"
        memory: "8Gi"

  networkPolicies:
    - name: org-deny-all
      description: Block all traffic — start here and add policies for what you need
      policyTypes: [Ingress, Egress]

  rollingUpdate:
    - name: org-safe
      description: Never reduces capacity during a rollout
      maxSurge: "1"
      maxUnavailable: "0"
```

They resolve before built-ins. A profile named `deny-all` in your `profiles:` block shadows the built-in. A conflict between two imported Motifs declaring the same profile name in the same class is a hard error at load time.

See [User-defined profiles](./10-user-defined-profiles.md) for the full reference.

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
| [Reconciler](./11-reconciler-profile.md) | Reconciler tuning — workers, resync interval, queue depth | `operatorBox.reconciler` |
| [User-defined](./10-user-defined-profiles.md) | Custom named profiles declared in your Katalog or Motif | All profile-supporting fields |

---

## Rules that apply to every profile

- **Atomic** — a profile fully defines its configuration. You cannot combine a profile with manual fields of the same type.
- **Fail-fast** — an unknown profile name is a Katalog load error, not a runtime error.
- **No mixing** — a profile and explicit fields of the same type cannot coexist on the same resource.
- **Template-safe** — profile names can be template expressions. Static names are validated immediately; template expressions are validated at reconcile time.
- **User-defined** — teams can declare custom named profiles in a Katalog or Motif `profiles:` block. They resolve before built-ins. See [User-defined profiles](./10-user-defined-profiles.md).
