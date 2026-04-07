---
title: "Conditional Provisioning in Orkestra"
weight: 11
description: "Declarative, per‑resource conditions for dynamic operator behavior"
---

Conditional provisioning lets you declare **when** a resource should be created, updated, or skipped — based entirely on the fields of the Custom Resource (CR) being reconciled.

This allows a single Katalog to express multiple deployment modes, optional components, environment‑specific behavior, and feature flags **without writing Go code**.

!!! note
    Conditional provisioning is evaluated on *every* reconcile.  
    If a condition fails, the resource is skipped cleanly — no errors, no partial state.

---

# **Why Conditional Provisioning Exists**

Operators often need to create resources *only if* certain fields are present or certain values are set.

Examples:

- Create a `Service` only if `spec.exposePublicly = true`
- Create a `ConfigMap` only if `spec.config` exists
- Create a `Deployment` only if `spec.replicas > 0`
- Create a `Secret` only if `spec.credentialsRef` is provided
- Create a `Job` only if `spec.runMigrations = true`

!!! tip
    In traditional operators, this logic lives in Go code.  
    In Orkestra, it lives in **YAML**, inside your Katalog.

---

# **How It Works**

Every resource under:

- `onCreate`
- `onReconcile`
- `onDelete`

may include a `when:` block:

```yaml
when:
  - field: spec.exposePublicly
    equals: "true"
```

Each entry in `when:` is a **Condition**.

- All conditions are **AND’ed**  
- If *any* condition fails → the resource is **skipped**  
- Skipping is **not an error**  

!!! caution
    Conditions never delete resources.  
    If a resource was previously created and conditions later fail, Orkestra will **stop reconciling it**, but will not remove it.  
    Use `onDelete` if you need explicit teardown logic.

---

# **Supported Condition Operators**

| Operator | Meaning |
|----------|---------|
| `equals` | Field value equals the condition value (string compare) |
| `notEquals` | Field value does *not* equal the condition value |
| `contains` | Field contains the value as a substring |
| `prefix` | Field starts with the value |
| `suffix` | Field ends with the value |
| `exists` | Field exists and is non‑empty |
| `notExists` | Field is absent or empty |
| `gt` | Field value is numerically greater than the condition value |
| `lt` | Field value is numerically less than the condition value |

!!! tip
    `equals` is the default operator.  
    If `equals` is present, `operator:` is ignored.

---

# **Field Resolution**

`field:` uses **dot notation** to reference CR fields:

- `spec.image`
- `spec.replicas`
- `spec.exposePublicly`
- `metadata.name`
- `metadata.labels.tier`

Nested fields are supported.

If a field does not exist:

- `exists` → **false**  
- `notExists` → **true**  
- all other operators → **false**

!!! note
    Missing fields never cause errors — they simply fail the condition.

---

# **Examples**

## **Conditional Service Creation**

```yaml
services:
  - name: "{{ .metadata.name }}-public"
    type: "LoadBalancer"
    port: "80"
    targetPort: "{{ .spec.port }}"
    namespace: "{{ .metadata.namespace }}"
    reconcile: true
    when:
      - field: spec.exposePublicly
        equals: "true"
```

If the CR has:

```yaml
spec:
  exposePublicly: "false"
```

The service is **skipped**.

---

## **Multiple Conditions (AND)**

```yaml
when:
  - field: spec.environment
    equals: "production"
  - field: spec.replicas
    gt: "2"
```

This resource is created only when:

- `spec.environment == "production"`  
**AND**
- `spec.replicas > 2`

---

## **Optional ConfigMap**

```yaml
configMaps:
  - name: "{{ .metadata.name }}-config"
    namespace: "{{ .metadata.namespace }}"
    data:
      config.yaml: "{{ .spec.config }}"
    reconcile: true
    when:
      - field: spec.config
        exists: true
```

If `spec.config` is missing, the ConfigMap is skipped.

---

## **Feature Flags**

```yaml
deployments:
  - name: "{{ .metadata.name }}-worker"
    image: "{{ .spec.workerImage }}"
    replicas: "{{ .spec.workerReplicas }}"
    namespace: "{{ .metadata.namespace }}"
    reconcile: true
    when:
      - field: spec.features.workers
        equals: "enabled"
```

!!! tip
    Feature flags are one of the most common uses of conditional provisioning.

---

# **How Orkestra Evaluates Conditions**

During reconcile:

1. The GenericReconciler loads the CR from the informer cache  
2. Before creating a resource, it evaluates all `when:` conditions  
3. If all pass → resource is created/updated  
4. If any fail → resource is skipped  
5. Skipped resources produce a debug log:

```
conditions not met — skipping resource
```

!!! note
    Skipping is a **clean no‑op**.  
    It does not count as an error and does not requeue the CR.

---

# **Best Practices**

- Use `equals: "true"` for booleans  
CRDs often store booleans as strings.

-  Use `exists` for optional fields  
Cleaner than comparing to empty strings.

-  Use numeric operators (`gt`, `lt`) for scaling logic  
Great for autoscaling‑like behavior.

-  Keep conditions simple  
Complex logic belongs in the CR, not the Katalog.

-  Remember: all conditions are AND’ed  
If you need OR logic, split into multiple resources.

!!! warning
    Avoid deeply nested conditions — they make the operator harder to reason about.  
    Prefer explicit CR fields that express intent.

---

# **Summary**

Conditional provisioning gives you:

- Feature flags  
- Environment‑specific behavior  
- Optional resources  
- Clean YAML‑only logic  
- Zero Go code  
- Predictable, idempotent behavior  

It is one of the most powerful parts of Orkestra’s declarative operator model.

---

---

## Related Documentation

- **Concept:** [Templating Engine](../concepts/templating.md)
- **Reference:** [Katalog Schema — Conditions](../../reference/katalog-schema.md#conditions)
- **Next Use Case:** [Zero‑Code Operators](../../use-cases/zero-code-operators.md)
