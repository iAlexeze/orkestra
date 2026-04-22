# ConfigMap as a Data-Driven Control Surface

### Rethinking Kubernetes Operators Without CRDs

---

## Abstract

Kubernetes operators traditionally rely on Custom Resource Definitions (CRDs) to introduce new APIs and drive reconciliation logic. While powerful, this model introduces operational overhead: API design, versioning, schema management, and lifecycle complexity.

This paper introduces a **data-driven operator model** implemented in Orkestra, where **existing Kubernetes resources—specifically ConfigMaps—serve as universal inputs to reconciliation**. Instead of defining new APIs, Orkestra attaches behavior to already-available primitives, transforming them into **declarative control surfaces**.

This approach enables:

* Zero-CRD operator workflows
* Reduced operational overhead
* Native Kubernetes compatibility
* A unified control plane model where **behavior is decoupled from API definition**

---

## 1. Introduction

Kubernetes’ extensibility is largely built on CRDs and controllers. The canonical operator pattern is:

1. Define a CRD (new API)
2. Write a controller
3. Reconcile desired vs actual state

While effective, this pattern introduces friction:

* CRDs must be designed, versioned, and maintained
* Controllers duplicate boilerplate logic (queues, retries, workers)
* APIs proliferate, increasing system complexity

At its core, however, a CRD is simply:

> **Structured input to a reconciliation loop**

This observation leads to a key insight:

> **If a CR is just structured input, any structured resource can serve the same role.**

---

## 2. The Core Insight

A Kubernetes `ConfigMap` is:

* Structured (key-value)
* Namespaced
* Watchable
* Always available in every cluster

Functionally, it behaves as:

> **A generic, schema-less input resource**

Therefore:

> **A ConfigMap can act as a “Custom Resource” without requiring a CRD.**

---

## 3. The Orkestra Model

Orkestra introduces a runtime that:

* Watches Kubernetes resources
* Matches them via selectors
* Applies declarative behavior (defined in a Katalog)
* Reconciles derived resources

---

### 3.1 Behavior Definition (Katalog)

```yaml
apiTypes:
  kind: ConfigMap
  selector:
    matchLabels:
      orkestra.io/katalog: pipeline-engine
```

This declares:

> “Attach this behavior to matching ConfigMaps.”

---

### 3.2 Input Resource (ConfigMap)

```yaml
kind: ConfigMap
metadata:
  name: payments-service
  labels:
    orkestra.io/katalog: pipeline-engine

data:
  repo: https://github.com/org/payments
  image: registry.local/payments
  replicas: "2"
```

---

### 3.3 Reconciliation Behavior

The runtime interprets `.data` as input and derives outputs:

```yaml
deployments:
  - name: "{{ .metadata.name }}"
    image: "{{ .data.image }}"
    replicas: "{{ .data.replicas }}"
```

---

## 4. Architectural Model

This model separates concerns cleanly:

| Layer             | Responsibility                |
| ----------------- | ----------------------------- |
| Kubernetes API    | Stores input resources        |
| ConfigMap         | Declares desired input data   |
| Orkestra          | Interprets and reconciles     |
| Derived Resources | Represent actual system state |

---

## 5. Key Design Principle

> **Orkestra does not mutate input resources. It derives outputs from them.**

This ensures:

* No controller conflicts
* No ownership ambiguity
* Predictable reconciliation behavior

---

## 6. Explicit Binding

To prevent ambiguity, resources must explicitly opt in:

```yaml
metadata:
  labels:
    orkestra.io/katalog: pipeline-engine
```

This creates a clear contract:

> “This resource is an input to this behavior.”

---

## 7. Status Without CRDs

Since ConfigMaps lack a `.status` field, Orkestra uses annotations:

```yaml
metadata:
  annotations:
    orkestra.io/phase: "Deploying"
    orkestra.io/lastCommit: "abc123"
```

This provides:

* Observability
* Debugging insight
* Zero additional API surface

---

## 8. Comparison to Traditional Operators

| Feature        | CRD-Based Operators | ConfigMap Model    |
| -------------- | ------------------- | ------------------ |
| API Definition | Required            | Not required       |
| Schema         | Strongly typed      | Flexible           |
| Setup Overhead | High                | Minimal            |
| Control        | Explicit API        | Label-based        |
| Status         | Native `.status`    | Annotations        |
| Flexibility    | High                | High (data-driven) |

---

## 9. Advantages

### 9.1 Zero-Install API Surface

No CRDs required → works in any cluster immediately.

---

### 9.2 Reduced Cognitive Load

Users interact with familiar Kubernetes primitives.

---

### 9.3 Faster Iteration

No API design cycle — behavior evolves independently.

---

### 9.4 Unified Control Plane

All logic runs inside Kubernetes reconciliation.

---

### 9.5 Safe by Design

* Input is never mutated
* Behavior is explicitly bound
* Outputs are derived deterministically

---

## 10. Limitations and Trade-offs

### 10.1 Lack of Schema Validation

ConfigMaps are schema-less → validation must be handled at runtime.

---

### 10.2 Discoverability

Without CRDs, APIs are less self-documenting.

---

### 10.3 Overloading Semantics

ConfigMaps must be clearly scoped to avoid misuse.

---

## 11. When to Use This Model

### Ideal Use Cases

* Internal platforms
* CI/CD pipelines
* Lightweight automation
* Rapid prototyping
* GitOps-style workflows

---

### When to Prefer CRDs

* Strong API contracts are required
* Public APIs are exposed
* Validation must be enforced at admission level

---

## 12. Dual-Mode Architecture

Orkestra supports both models:

### Structured Mode (CRDs)

* Strong typing
* Explicit APIs

### Data-Driven Mode (Built-ins)

* Lightweight
* Fast iteration

---

> **Same engine, different input surfaces**

---

## 13. Conceptual Shift

Traditional model:

> API → Controller → Behavior

Orkestra model:

> Data → Behavior → Reconciliation

---

## 14. Implications

This approach reframes Kubernetes:

* Not just a platform for workloads
* But a **runtime for declarative behavior**

---

## 15. Conclusion

The ConfigMap data-driven model challenges a core assumption in Kubernetes:

> That operators require new APIs.

By treating existing resources as structured input, Orkestra enables:

* Simpler operator development
* Faster iteration cycles
* Reduced system complexity

Ultimately, this leads to a new paradigm:

> **Behavior is no longer defined by APIs — it is attached to data.**

---

## 16. Final Thought

> Kubernetes resources are no longer just state containers.
> They become inputs to programmable reconciliation.

And with that:

> **Operators are no longer written — they are described.**

