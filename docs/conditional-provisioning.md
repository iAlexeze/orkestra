# Conditional Provisioning in Orkestra  
**Declarative, per‑resource conditions for dynamic operator behavior**

Conditional provisioning lets you declare **when** a resource should be created, updated, or skipped — based entirely on the fields of the Custom Resource (CR) being reconciled.

This allows a single Katalog to express multiple deployment modes, optional components, environment‑specific behavior, and feature flags **without writing Go code**.

---

# Why Conditional Provisioning Exists

Operators often need to create resources *only if* certain fields are present or certain values are set.  
Examples:

- Create a `Service` only if `spec.exposePublicly = true`
- Create a `ConfigMap` only if `spec.config` exists
- Create a `Deployment` only if `spec.replicas > 0`
- Create a `Secret` only if `spec.credentialsRef` is provided
- Create a `Job` only if `spec.runMigrations = true`

In traditional operators, this logic lives in Go code.  
In Orkestra, it lives in **YAML**, inside your Katalog.

---

# How It Works

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

All conditions in the list are **AND’ed**.  
If *any* condition fails, the resource is **skipped** for that reconcile cycle.

Skipping is **not an error** — it is a clean no‑op.

---

# Supported Condition Operators

Orkestra supports a rich set of operators:

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

### Shorthand: `equals`

You may use:

```yaml
equals: "true"
```

instead of:

```yaml
operator: equals
value: "true"
```

If `equals` is present, `operator` is ignored.

---

# Field Resolution

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

---

# Example: Conditional Service Creation

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

Then the service is **skipped**.

---

# Example: Multiple Conditions (AND)

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

# Example: Optional ConfigMap

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

# Example: Feature Flags

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

---

# How Orkestra Evaluates Conditions

During reconcile:

1. The GenericReconciler loads the CR from the informer cache.
2. Before creating a resource, it evaluates all `when:` conditions.
3. If all pass → resource is created/updated.
4. If any fail → resource is skipped.
5. Skipped resources produce a debug log:

```
conditions not met — skipping resource
```

This is intentional and expected.

---

# Best Practices

### ✔ Use `equals: "true"` for booleans  
CRDs often store booleans as strings.

### ✔ Use `exists` for optional fields  
Cleaner than comparing to empty strings.

### ✔ Use numeric operators (`gt`, `lt`) for scaling logic  
Great for autoscaling‑like behavior.

### ✔ Keep conditions simple  
Complex logic belongs in the CR, not the Katalog.

### ✔ Remember: all conditions are AND’ed  
If you need OR logic, split into multiple resources.

---

# Summary

Conditional provisioning gives you:

- Feature flags  
- Environment‑specific behavior  
- Optional resources  
- Clean YAML‑only logic  
- Zero Go code  
- Predictable, idempotent behavior  

It is one of the most powerful parts of Orkestra’s declarative operator model.


**Whats Next?**
  - [Orkestra Use Cases](./use-cases.md)
  - [What is a Katalog](./katalog.md)
  - [What is a Komposer](./komposer.md)