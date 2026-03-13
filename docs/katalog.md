# **The Orkestra Katalog**
### *A Declarative Bundle for CRDs, Behavior, and Runtime Orchestration*

The **Katalog** is the heart of Orkestra’s architecture.  
It is a **declarative, composable, dependency‑aware bundle** that describes:

- what CRDs exist  
- how they behave  
- how they depend on each other  
- how they should be wired at runtime  
- how Orkestra should generate clients, informers, reconcilers, and workers  

It is the single source of truth for the entire operator runtime.

If the Kubernetes API is the “data plane,”  
the **Katalog is the control plane** for Orkestra.

---

# What Is a Katalog?

A **Katalog** is a structured definition that contains one or more **CRD entries**.  
Each entry describes:

- the CRD’s API (group, version, kind, plural)  
- whether it is enabled  
- its dependencies  
- its worker count  
- its resync interval  
- its reconciler (default or custom)  
- its informer options  
- its scheme information  
- its mode (Go or YAML)  

In other words:

> **A Katalog is to Orkestra what a Helm Chart is to Kubernetes —  
but for CRDs, controllers, and runtime behavior.**

---

# Why the Katalog Exists

Traditional operator frameworks require:

- static wiring  
- hand‑rolled clients  
- hand‑rolled informers  
- one controller per CRD  
- boilerplate everywhere  

Orkestra replaces all of that with a **single declarative bundle**.

The Katalog enables:

### 🔹 Dynamic CRD loading  
Go or YAML — local or remote.

### 🔹 Automatic client + informer generation  
No boilerplate, no codegen, no controller‑runtime magic.

### 🔹 Dependency‑aware orchestration  
CRDs start in topological order and shut down in reverse order.

### 🔹 Per‑CRD workers and resync  
Each CRD defines its own concurrency and refresh interval.

### 🔹 Pluggable reconcilers  
Use the generic reconciler or provide your own.

### 🔹 Multi‑CRD operator composition  
Build operators with 2, 5, or 50 CRDs — all from one Katalog.

---

# Katalog Structure

A Katalog is composed of **CRD entries**:

```yaml
crds:
  - name: project
    group: platform.orkestra.io
    version: v1alpha1
    kind: Project
    plural: projects
    enabled: true
    workers: 3
    resync: 10m
    dependsOn: []
    reconciler:
      default: true

  - name: managednamespace
    group: platform.orkestra.io
    version: v1alpha1
    kind: ManagedNamespace
    plural: managednamespaces
    enabled: true
    workers: 2
    resync: 30s
    dependsOn: ["project"]
    reconciler:
      default: false
      function: NewManagedNamespaceReconciler
      url: https://raw.githubusercontent.com/.../reconciler.go
```

This is the declarative definition that Orkestra uses to build the runtime.

---

# How Orkestra Uses the Katalog

When Orkestra starts:

1. **Load the Katalog** (Go or YAML)
2. **Filter enabled CRDs**
3. **Validate dependency graph**
4. **Generate scheme registry**
5. **Generate clients**
6. **Generate informers**
7. **Generate reconcilers**
8. **Build dependency‑aware controller**
9. **Start workers per CRD**
10. **Begin event dispatching**

Everything is driven by the Katalog.

---

# Go Mode vs YAML Mode

The Katalog supports two modes:

## **Go Mode**
- CRDs defined in Go  
- Strong typing  
- Automatic scheme registration  
- Best for framework authors  

## **YAML Mode**
- CRDs defined in YAML  
- GitOps‑friendly  
- Remote katalogs supported  
- Best for platform teams and multi‑cluster orchestration  

Both modes produce the same runtime behavior.

---

# Dependency Graph

Each CRD can declare:

```yaml
dependsOn:
  - project
```

Orkestra builds a DAG and:

- detects cycles  
- computes topological order  
- starts CRDs in dependency order  
- shuts them down safely  

This is essential for multi‑CRD operators.

---

# CLI Integration

The Katalog is fully introspectable via the CLI:

```bash
ork katalog list
ork katalog describe <crd>
ork explain <crd>
ork graph deps
ork graph order
ork graph tree
ork get crds
ork get controllers
```

See **[Orkestra CLI](./cli.md)** for full details.

---

# Future Thoughts & Roadmap

The Katalog unlocks a new era of operator design.  
Here are some future directions already on the horizon:

## **1. Katalog Repositories**
Like Helm repos — publish katalogs for:

- internal teams  
- partners  
- open‑source ecosystems  

## **2. Remote Go Reconcilers**
Fetch reconcilers dynamically from Git or OCI.

## **3. Katalog Marketplace**
A curated library of:

- CRDs  
- reconcilers  
- behaviors  
- patterns  

## **4. Katalog Composition**
Import katalogs into katalogs:

```yaml
imports:
  - github.com/org/platform-katalog
  - github.com/org/networking-katalog
```

## **5. Katalog Validation Webhooks**
Validate katalogs before runtime.

## **6. Katalog‑Driven Code Generation**
Optional codegen for teams that want static clients.

## **7. Katalog‑Aware UI**
A dashboard that visualizes:

- dependency graphs  
- controller health  
- worker pools  
- reconcile metrics  

---

# Summary

The **Katalog** is the foundation of Orkestra’s runtime.  
It is:

- declarative  
- composable  
- dependency‑aware  
- mode‑agnostic  
- introspectable  
- extensible  

It transforms operator engineering from boilerplate into orchestration.

Use it to define your CRDs.  
Use it to wire your controllers.  
Use it to build multi‑CRD systems with elegance and clarity.