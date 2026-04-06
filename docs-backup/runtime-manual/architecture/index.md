# Orkestra Architecture Overview

Orkestra is a **declarative operator runtime**.  
You describe *what* an operator should do in YAML (a **Katalog**), and Orkestra handles *how* it runs.

This overview explains the major subsystems of the Orkestra runtime and how they work together to deliver a self‑healing, dependency‑aware operator engine.

---

## 1. High‑Level Overview

Orkestra consists of four major subsystems:

1. **Pre‑Runtime (Komposer + Katalog Loader)**  
   Merges multiple Katalog sources and produces a validated runtime Katalog.

2. **Runtime Core**  
   Informers, workqueues, reconciler registry, template engine.

3. **Dependency‑Aware Kontroller**  
   Starts CRDs in topological order, handles activation/deactivation, manages worker pools.

4. **Reconciler Layer**  
   GenericReconciler + optional Go hooks + template engine.

For a complete visual representation of the entire system, see the  
👉 **[Full Architecture View](./full-architecture-view.md)**

---

## 2. Pre‑Runtime: Komposer + Katalog Loader

Before Orkestra starts reconciling anything, it builds the **runtime Katalog**:

- Merge multiple files, Helm charts, URLs  
- Apply overrides  
- Validate schema  
- Apply defaults  
- Build dependency graph  
- Register CRDs and reconciler configs  

This produces a complete, normalized operator definition.

---

## 3. Informer Factory

For each CRD:

- A SharedIndexInformer is created  
- If the CRD exists → informer starts  
- If missing → informer is created but not started  
- When the CRD appears → retry loop activates it  

Informers feed events into per-CRD workqueue.

---

## 4. Dependency‑Aware Kontroller

The Kontroller:

- Computes startup order (topological sort)
- Waits for dependencies using ready channels
- Starts worker pools per CRD
- Monitors CRD existence
- Deactivates CRDs when deleted
- Reactivates CRDs when they reappear
- Shuts down in reverse dependency order

This is the heart of Orkestra’s self‑healing model.

---

## 5. Reconciler Registry

Each CRD maps to a **Reconciler**:

- GenericReconciler (zero‑code)
- Optional Go hooks (OnReconcile, OnDelete, OnNotFound)
- Template Engine (Deployments, Services, Secrets, ConfigMaps, Jobs…)

The registry dispatches events to the correct reconciler based on GVK.

---

## 6. Template Engine

The template engine handles:

- Field resolution (`{{ .spec.image }}`)
- Conditional provisioning (`when:` blocks)
- Resource builders (Deployment, Service, Secret…)
- Drift correction (reconcile: true)
- Owner references
- Idempotency

This is where declarative operator logic becomes real Kubernetes objects.

---

## What’s Next?

Continue to the **Runtime Flow** to understand how Orkestra handles:

- workers  
- informers  
- reconcile pipeline  

👉 **[Runtime Flow →](./runtime-flow.md)**
