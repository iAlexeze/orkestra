# Orkestra Design Philosophy
Why Orkestra is built the way it is — and what principles guide its architecture.

This document complements the [CRD Lifecycle](./crd-lifecycle.md) and explains the deeper reasoning behind Orkestra’s design.

---

## 1. **CRDs Are Data**

Operators should be **declared**, not coded.

A CRD in Orkestra is:

- a data structure  
- not a Go type  
- not a controller implementation  

This allows:

- zero‑code operators  
- dynamic CRDs  
- runtime‑defined behavior  
- multi‑source composition (Komposer)  

---

## 2. **Dependency‑Aware Lifecycle**

Operators often depend on each other.

Orkestra enforces:

- topological startup  
- reverse shutdown  
- readyCh synchronization  
- automatic activation/deactivation  

This prevents:

- race conditions  
- partial startup  
- broken dependency chains  

---

## 3. **Zero‑Code by Default**

Most operators don’t need Go or custom code.

Orkestra provides:

- declarative templates  
- conditional provisioning  
- drift correction  
- resource builders  
- owner references  

This covers 90% of operator use cases.

---

## 4. **Typed When Needed**

When you need:

- type‑safe hooks  
- external API calls  
- complex logic  

You can add Go hooks or custom reconcilers.

But the runtime remains the same.

---

## 5. **Self‑Healing**

CRDs can appear, disappear, or change at any time.

Orkestra:

- detects missing CRDs  
- activates them when they appear  
- deactivates them when deleted  
- reactivates them when reinstalled  

This makes operators resilient to:

- cluster drift  
- CRD upgrades  
- partial deployments  

---

## 6. **Isolation and Safety**

Orkestra isolates failures at:

- CR level  
- CRD level  
- worker level  
- runtime level  

`safereconcile()` ensures:

- no panics escape  
- no worker crashes  
- no runtime crashes  

---

## 7. **Deterministic Startup and Shutdown**

Startup:

1. Health server  
2. Kubeclient  
3. Event recorder  
4. Queues  
5. Informers  
6. DependencyKontroller  

Shutdown is the reverse.

This guarantees:

- no dropped events  
- no partial reconciles  
- no double processing  

---

## **What’s Next?**

Explore the **Architecture Diagrams** for a complete visual reference.

👉 [Architecture Diagrams →](./architecture-diagrams.md)
