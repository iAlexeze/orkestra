# Orkestra Architecture Diagrams 
### A visual guide to the internal components of the Orkestra runtime

This document provides a complete, visual overview of how Orkestra works internally.  
It complements the [Architecture](./architecture.md), [Trust Model, and Failure Modes](./trust-and-failure-model.md) documents by showing  
**how the pieces fit together**.

---

## 1. High‑Level Architecture (Two Modes)

Orkestra supports two ways to supply operator definitions:

---

### 1A. Direct Katalog → Runtime (simple mode)

```mermaid
flowchart LR
    A["Katalog.yaml<br>(1 file, many CRDs)"] --> B[Orkestra Runtime]
    B --> C[OrkestraRegistry]
    C --> D[Kubernetes API]

    style B fill:#FF6D00,stroke:#333,stroke-width:2px,color:#fff
    style C fill:#00C853,stroke:#333,stroke-width:2px,color:#fff
```

This is the simplest and most common mode.

---

### 1B. Komposer → Unified Katalog → Runtime (multi‑source mode)

```mermaid
flowchart LR
    A[Files] --> E[Komposer]
    B[Helm Charts] --> E
    C[Inline Overrides] --> E
    D["Registry Entries (future)"] --> E

    E --> F[Unified Katalog]
    F --> G[Orkestra Runtime]
    G --> H[OrkestraRegistry]
    H --> I[Kubernetes API]

    style E fill:#FF6D00,stroke:#333,stroke-width:2px,color:#fff
    style F fill:#00C853,stroke:#333,stroke-width:2px,color:#fff
    style G fill:#FF6D00,stroke:#333,stroke-width:2px,color:#fff
    style H fill:#00C853,stroke:#333,stroke-width:2px,color:#fff
```

**Komposer is a build step, not part of the runtime.**  
It produces a single Katalog that the runtime consumes.

---

## 2. Reconcile Loop (per CRD)**

```mermaid
flowchart LR
  A["CR event<br>(add/update/delete)"] --> B[Informer]
  B --> C[Workqueue]
  C --> D[Worker Pool]
  D --> E[safeReconcile]
  E --> F[Template Resolver]
  F --> G[OrkestraRegistry]
  G --> H[Kubernetes API]
  H --> I[Status + Metrics]

  style E fill:#FF6D00,stroke:#333,stroke-width:2px,color:#fff
  style G fill:#00C853,stroke:#333,stroke-width:2px,color:#fff
```

**Key properties:**

- Each CRD has its own queue  
- Each CRD has its own workers  
- `safeReconcile` isolates panics  
- Registry operations are idempotent  

---

## **3. safeReconcile — Fault Isolation**

```mermaid
flowchart TD
    A[Worker] --> B[safeReconcile]
    B -->|panic| C["recover()"]
    B -->|error| D[Requeue with backoff]
    B -->|success| E[Forget item]

    style B fill:#FF6D00,stroke:#333,stroke-width:2px,color:#fff
```

**Guarantee:**  
A panic in one CR **cannot** crash the worker or the runtime.

---

## 4. Dependency Engine

```mermaid
graph TD
    P[project] --> A[application]
    P --> N[namespace]
    A --> M[monitoring]

    class P,A,N,M crd;
```

**Rules:**

- Dependencies must be healthy before dependents start  
- Missing dependencies retry in background  
- Healthy CRDs never block  

---

## 5. Komposer Merge Pipeline (multi‑source mode)

```mermaid
flowchart LR
    A[Files] --> E[Komposer]
    B[Helm Charts] --> E
    C[Inline Overrides] --> E
    D["Registry Entries (future)"] --> E

    E --> F[Merger]
    F --> G[Unified Katalog]
    G --> H[Runtime]

    style E fill:#FF6D00,stroke:#333,stroke-width:2px,color:#fff
    style G fill:#00C853,stroke:#333,stroke-width:2px,color:#fff
```

**Features:**

- Deduplication  
- Override rules  
- Environment‑specific layering  
- Version pinning  

---

## 6. Template Resolution

```mermaid
flowchart LR
    A[CR Object] --> B[Resolver]
    B --> C[Resolved Values]
    C --> D[Registry Spec]
    D --> E[Resource Apply]
```

**Resolver responsibilities:**

- Evaluate `{{ .spec.* }}`  
- Apply defaults  
- Enforce namespace rules  
- Resolve labels, slices, maps  

---

## 7. OrkestraRegistry Flow

```mermaid
flowchart LR
    A[Resolved Spec] --> B[Create]
    A --> C[Update]
    A --> D[Delete]

    B --> E[Kubernetes API]
    C --> E
    D --> E

    style B fill:#00C853,stroke:#333,stroke-width:2px,color:#fff
    style C fill:#00C853,stroke:#333,stroke-width:2px,color:#fff
    style D fill:#00C853,stroke:#333,stroke-width:2px,color:#fff
```

**Registry guarantees:**

- Idempotent  
- Safe to retry  
- Safe to reapply  
- Owner references always set  

---

## 8. Health & Metrics Pipeline

```mermaid
flowchart LR
    A[Reconcile Result] --> B[Health Subsystem]
    B --> C[CRD Health]
    B --> D[Metrics]
    D --> E[Prometheus]
```

Metrics include:

- reconcile duration  
- reconcile count  
- queue depth  
- worker activity  
- CRD activation latency  

---

## 9. Failure Containment Model

```mermaid
flowchart TD
    A[CR Failure] --> B[CRD Queue]
    B --> C[safeReconcile]
    C -->|retry| B
    C -->|panic| D["recover()"]
    C -->|error| E[backoff]

    style C fill:#FF6D00,stroke:#333,stroke-width:2px,color:#fff
```

**Isolation boundaries:**

- CR-level  
- CRD-level  
- Worker-level  
- Runtime-level  

---

## 10. End-to-End Flow

```mermaid
sequenceDiagram
    participant User
    participant K8s
    participant Runtime
    participant Resolver
    participant Registry

    User->>K8s: Apply CR
    K8s->>Runtime: Event
    Runtime->>Runtime: Queue item
    Runtime->>Resolver: Resolve templates
    Resolver->>Registry: Build spec
    Registry->>K8s: Create/Update/Delete
    K8s->>Runtime: Status/Events
```

---

## Summary

This document provides a complete visual map of:

- the two supported input modes (Katalog vs Komposer)  
- the reconcile loop  
- the dependency engine  
- the Komposer pipeline  
- the registry  
- the resolver  
- the health subsystem  
- the failure isolation model  

It is the **single most important reference** for understanding Orkestra’s internals.

**Whats Next?**
  - [Trust and Failure Model](../trust-and-failure-model.md)