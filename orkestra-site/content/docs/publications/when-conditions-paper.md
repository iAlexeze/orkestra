---
title: "When Conditions Paper"
weight: 77
---

# Conditional Provisioning: When Conditions and the Two Enforcement Boundaries

*Orkestra Project — March 2026*

---

## Abstract

Orkestra provides two mechanisms for controlling what resources exist in a cluster:
`when:` conditions and admission rules. They are not alternatives to each other.
They enforce different things at different boundaries, and understanding the
distinction is what enables platform engineers to build expressive, correct operator
topologies. This paper defines both mechanisms precisely, shows where each applies,
and demonstrates the declarative topology pattern — the ability to make an operator's
behavior change continuously with the state of the CR, without redeploying or
reconfiguring anything.

---

## 1. Two different questions

Admission rules and `when:` conditions answer different questions.

**Admission rules answer:** *Should this CR exist?*

When a `Website` CR with a non-compliant image is submitted, a deny rule answers:
no, this CR should not be stored. The API server rejects it before it reaches etcd.
The user sees an error immediately. The CR does not exist anywhere in the cluster.

**`when:` conditions answer:** *Given that this CR exists and is valid, which child
resources should currently exist?*

When a valid `Website` CR has `spec.environment: staging`, a `when:` condition can
answer: the LoadBalancer Service and the PodDisruptionBudget should not exist. The
CR is valid. It is stored. It is reconciled. But these specific child resources are
not created because the conditions for creating them are not met.

The CR passes. The child resource creation is conditional. These are not the same
statement, and they cannot be substituted for each other.

---

## 2. Admission: enforcement at the boundary

Admission runs at the Kubernetes API server boundary — the moment an object is
submitted via `kubectl apply` or any API client. When `ENABLE_ADMISSION_WEBHOOK=true`,
Orkestra registers a `ValidatingWebhookConfiguration` and a
`MutatingWebhookConfiguration`. The API server calls Orkestra's `/validate` and
`/mutate` endpoints synchronously during every CREATE and UPDATE operation.

**Deny at admission:**

```yaml
validation:
  - field: spec.image
    prefix: "myorg/"
    message: "images must come from the internal registry"
    action: deny
```

If this rule fires, the API server returns an error to the client before the object
is stored. The user sees:

```
Error from server: admission webhook "validate.orkestra.orkspace.io" denied the request:
[orkestra] validation failed: field "spec.image": images must come from the internal registry
```

The CR does not exist in the cluster. There is nothing to reconcile. Nothing to
clean up. The enforcement boundary is at apply time, and it is absolute.

**Warn at admission:**

```yaml
validation:
  - field: metadata.labels.team
    operator: exists
    message: "all resources should declare a team owner"
    action: warn
```

The CR is stored. The user sees a warning in their `kubectl` output:

```
Warning: orkestra: field "metadata.labels.team": all resources should declare a team owner
website.demo.orkestra.io/my-site created
```

Warn is advisory. It surfaces policy intent without blocking deployment.

**The same rules run at reconcile time.** If `ENABLE_ADMISSION_WEBHOOK=false`, or if a CR
existed before the webhook was enabled, the same validation runs on every reconcile
cycle. `action: deny` halts reconciliation. `action: warn` surfaces the violation
on the `/katalog/{crd}` health endpoint as an active warning.

---

## 3. When conditions: enforcement at the resource level

`when:` conditions run inside the reconcile loop, evaluated per-resource, per-cycle.
They do not operate on the CR itself — they operate on whether to create a specific
child resource given the current state of the CR.

```yaml
onCreate:
  services:
    - name: "{{ .metadata.name }}-lb"
      type: LoadBalancer
      port: "443"
      when:
        - field: spec.environment
          equals: production    # only in production
        - field: spec.tlsEnabled
          operator: exists      # only when TLS is configured
```

When these conditions are not met — `spec.environment` is `staging`, or
`spec.tlsEnabled` is absent — the LoadBalancer Service is not created. No error.
No rejection. The reconcile succeeds. The CR is healthy. The Service simply does
not exist because the conditions for its existence are not satisfied.

**The critical difference:** The CR is valid at every `spec.environment` value.
`staging` is not a wrong value. `production` is not a required value. The condition
is not a constraint on the CR's validity — it is a statement about what the operator
should do given the CR's current state.

---

## 4. The declarative topology pattern

The power of `when:` conditions is not in any single condition — it is in what
conditions enable: an operator whose behavior changes continuously with the state
of the CR, without any operator-side logic to manage that change.

Consider a `Website` CR that supports multiple deployment tiers:

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      replicas: "{{ .spec.replicas }}"
      reconcile: true
      # Always created — every tier gets a Deployment

  services:
    - name: "{{ .metadata.name }}-internal"
      type: ClusterIP
      port: "80"
      reconcile: true
      # Always created — internal access for all tiers

    - name: "{{ .metadata.name }}-lb"
      type: LoadBalancer
      port: "443"
      when:
        - field: spec.tier
          notEquals: free
      # Only for paid tiers — free tier gets no LoadBalancer

  configMaps:
    - name: "{{ .metadata.name }}-rate-limits"
      data:
        rps: "1000"
      when:
        - field: spec.tier
          equals: enterprise
      # Enterprise rate limits — only for enterprise tier

  jobs:
    - name: "{{ .metadata.name }}-warmup"
      image: busybox
      command: ["cache-warmup.sh"]
      when:
        - field: spec.tier
          equals: enterprise
        - field: spec.cacheWarmup
          operator: exists
      # Cache warmup job — enterprise only, and only when requested
```

When a user changes their `Website` CR from `spec.tier: free` to `spec.tier:
pro`, the next reconcile cycle creates the LoadBalancer Service. When they
upgrade to `spec.tier: enterprise`, the rate limit ConfigMap and cache warmup
Job are created. When they downgrade, those resources are removed — not by
explicit deletion logic, but because `reconcile: true` re-evaluates the conditions
and no longer creates them.

The operator's topology changes with the data. No operator code changes. No
redeployment. One Katalog entry describes the full behaviour across all tiers.

---

## 5. Conditions do not emit errors

This is the operationally important point that is easy to miss.

When a `when:` condition evaluates to false, nothing is logged at warning level.
Nothing is emitted as a Kubernetes event. The resource is simply not created,
and the reconcile succeeds. The user who expects a LoadBalancer Service in staging
and does not find one receives no feedback from the operator about why.

This is correct behaviour — the CR is valid, the condition is met, the operator
is functioning as declared. But it creates an observability requirement: users
need to know which resources were skipped and why.

`ork describe website my-site` shows the reconcile detail including which template
resources were created and which were skipped due to conditions. The output
distinguishes between a resource that was not created because its conditions were
false and one that failed to create due to an error.

The pattern for platform engineers publishing Katalogs: document the `when:`
conditions in the README. A consumer of a Katalog pattern who does not know
about the `spec.tier` condition will spend time debugging a missing Service when
the answer is in the documentation.

---

## 6. Admission as the gate, conditions as the topology

The two mechanisms compose cleanly because they operate at different boundaries
with different semantics:

**Admission** ensures only valid, compliant CRs enter the cluster. It is the
gate. Rules express invariants — things that must always or never be true about
a CR.

**Conditions** shape what a valid CR produces. They are the topology. They express
contingent truths — things that should be true given the current state of the CR.

Together, they provide the full policy model:

```yaml
# Admission: the CR must be valid
validation:
  - field: spec.tier
    operator: contains
    # must be one of: free, pro, enterprise
    # (exact enum validation would use contains against a list, or multiple notEquals)
  - field: spec.image
    prefix: "myorg/"
    action: deny

# Reconcile: what exists depends on tier
onCreate:
  services:
    - name: "{{ .metadata.name }}-lb"
      type: LoadBalancer
      when:
        - field: spec.tier
          notEquals: free
```

The admission rule ensures no invalid tier value is stored. The condition ensures
the right resources exist for each valid tier. One mechanism cannot substitute for
the other.

---

## 7. When to use each

**Use admission rules when the CR should not exist if the condition fails.**

- Image not from internal registry → CR should not be created
- Missing required field → CR should not be created
- Replica count exceeds organisation policy → CR should not be created

**Use `when:` conditions when the CR is valid but some resources should not exist
given the current CR state.**

- Service type depends on environment (staging vs production)
- Additional resources depend on optional spec fields
- Resources depend on tier, plan, or feature flags in the CR
- Resources depend on the current phase in the CR's lifecycle

**The test:** if the CR can be valid at multiple states and each state should
produce a different set of resources, `when:` conditions are the answer. If any
CR state is invalid and should be rejected outright, admission rules are the answer.

---

## Conclusion

`when:` conditions and admission rules are two distinct and complementary
enforcement mechanisms. Admission rules operate at the API server boundary and
answer whether a CR should exist. Conditions operate at the reconcile boundary
and answer what a valid CR should produce.

The declarative topology pattern — an operator whose output changes continuously
with CR state, expressed entirely in YAML — is only possible because conditions
operate at the resource level rather than the CR level. A CR that is valid in
every state it can be in, but which produces different resources at each state,
is the fundamental building block of expressive platform operators.

Understanding the boundary between these mechanisms is not a detail of Orkestra's
implementation. It is the conceptual foundation of declarative operator design.
