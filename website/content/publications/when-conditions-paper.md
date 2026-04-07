---
title: "Conditional Provisioning: When Conditions and the Two Enforcement Boundaries"
weight: 50
description: "*Orkestra Project — March 2026*"
---

*Orkestra Project — March 2026*

---

## Abstract

Orkestra provides two mechanisms for controlling what resources exist in a cluster:
`when:` conditions and admission rules. They are not alternatives to each other.
They enforce different things at different boundaries, and understanding the
distinction is what enables platform engineers to build expressive, correct operator
topologies.

---

## 1. Two different questions

**Admission rules answer:** *Should this CR exist?*

When a `Website` CR with a non-compliant image is submitted, a deny rule answers:
no, this CR should not be stored. The API server rejects it before it reaches etcd.

**`when:` conditions answer:** *Given that this CR exists and is valid, which child
resources should currently exist?*

When a valid `Website` CR has `spec.environment: staging`, a `when:` condition can
answer: the LoadBalancer Service and the PodDisruptionBudget should not exist.

---

## 2. Admission: enforcement at the boundary

Admission runs at the Kubernetes API server boundary.

**Deny at admission:**

```yaml
validation:
  - field: spec.image
    prefix: "myorg/"
    message: "images must come from the internal registry"
    action: deny
```

If this rule fires, the API server returns an error to the client before the object
is stored. The CR does not exist in the cluster.

**Warn at admission:**

The CR is stored. The user sees a warning in their `kubectl` output.

---

## 3. When conditions: enforcement at the resource level

`when:` conditions run inside the reconcile loop, evaluated per-resource, per-cycle.

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

When these conditions are not met, the LoadBalancer Service is not created. No error.
No rejection. The reconcile succeeds. The CR is healthy.

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
    - name: "{{ .metadata.name }}-lb"
      type: LoadBalancer
      port: "443"
      when:
        - field: spec.tier
          notEquals: free
      # Only for paid tiers — free tier gets no LoadBalancer

  configMaps:
    - name: "{{ .metadata.name }}-rate-limits"
      when:
        - field: spec.tier
          equals: enterprise
      # Enterprise rate limits — only for enterprise tier
```

When a user changes their `Website` CR from `spec.tier: free` to `spec.tier:
pro`, the next reconcile cycle creates the LoadBalancer Service. When they
upgrade to `spec.tier: enterprise`, the rate limit ConfigMap is created.

The operator's topology changes with the data. No operator code changes. No
redeployment.

---

## 6. Admission as the gate, conditions as the topology

The two mechanisms compose cleanly because they operate at different boundaries:

**Admission** ensures only valid, compliant CRs enter the cluster. It is the
gate. Rules express invariants.

**Conditions** shape what a valid CR produces. They are the topology. They express
contingent truths.

Together, they provide the full policy model.

---

## 7. When to use each

**Use admission rules when the CR should not exist if the condition fails.**

- Image not from internal registry → CR should not be created
- Missing required field → CR should not be created

**Use `when:` conditions when the CR is valid but some resources should not exist
given the current CR state.**

- Service type depends on environment (staging vs production)
- Additional resources depend on optional spec fields
- Resources depend on tier, plan, or feature flags in the CR

---

## Conclusion

`when:` conditions and admission rules are two distinct and complementary
enforcement mechanisms. Admission rules operate at the API server boundary and
answer whether a CR should exist. Conditions operate at the reconcile boundary
and answer what a valid CR should produce.

Understanding the boundary between these mechanisms is not a detail of Orkestra's
implementation. It is the conceptual foundation of declarative operator design.
