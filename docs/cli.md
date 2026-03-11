# 🎛️ **Orkestra CLI Guide**
### *Interact with the Katalog. Visualize dependencies. Explore your runtime.*

The `ork` CLI is the command‑line interface for Orkestra.  
It provides deep insight into:

- the **Katalog** (CRD definitions + behavior)
- the **dependency graph**
- the **runtime controllers**
- the **CRD metadata**
- the **internal wiring** of Orkestra

This document covers all available commands with examples.

---

# 📦 **Katalog Commands**

The **Katalog** is the declarative bundle that defines:

- CRDs  
- dependencies  
- workers  
- resync intervals  
- reconcilers  
- informer options  
- Go/YAML mode behavior  

The CLI lets you inspect it directly.

---

## 🔹 `ork katalog list`

List all CRDs in the katalog (enabled + disabled).

```bash
$ ork katalog list
CRDs in katalog:
- project              (enabled)
- managednamespace     (enabled)
- application          (disabled)
```

---

## 🔹 `ork katalog describe <crd>`

Human‑readable description of a CRD.

```bash
$ ork katalog describe project
Name:        project
Group:       platform.orkestra.io
Version:     v1alpha1
Kind:        Project
Plural:      projects
Workers:     3
Resync:      10m
Enabled:     true
DependsOn:   []
```

---

## 🔹 `ork explain <crd>`

Technical explanation of how Orkestra wires the CRD internally.

```bash
$ ork explain managednamespace
ManagedNamespace (platform.orkestra.io/v1alpha1)
----------------------------------------
API Path:     /apis/platform.orkestra.io/v1alpha1
GVK:          platform.orkestra.io/v1alpha1, Kind=ManagedNamespace
Object Type:  *v1alpha1.ManagedNamespace
List Type:    *v1alpha1.ManagedNamespaceList
Reconciler:   *reconciler.ManagedNamespaceReconciler
Dependencies: [project]
```

This is the Orkestra equivalent of `kubectl explain`.

---

# 🔍 **Graph Commands**

Orkestra builds a dependency graph from the Katalog.  
The CLI lets you visualize it in multiple formats.

---

## 🔹 `ork graph deps`

Show the dependency graph.

```bash
$ ork graph deps
project
managednamespace -> [project]
application -> [project managednamespace]
```

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--dot` | Output in Graphviz DOT format |
| `--all` | Include disabled CRDs |

Example:

```bash
$ ork graph deps --dot
digraph CRDs {
  "managednamespace" -> "project";
  "application" -> "project";
  "application" -> "managednamespace";
}
```

---

## 🔹 `ork graph order`

Show CRDs in dependency‑safe startup order.

```bash
$ ork graph order
1. project
2. managednamespace
3. application
```

This is the topological sort of the dependency DAG.

---

## 🔹 `ork graph tree`

Pretty tree view of dependencies.

```bash
$ ork graph tree
project
  managednamespace
    application
```

---

# 📘 **Get Commands**

These commands reflect the **runtime state**, not just the katalog.

---

## 🔹 `ork get crds`

List enabled CRDs (the ones actually running).

```bash
$ ork get crds
Enabled CRDs:
- project (platform.orkestra.io/v1alpha1)
- managednamespace (platform.orkestra.io/v1alpha1)
```

---

## 🔹 `ork get controllers`

List active controllers (CRDs with reconcilers).

```bash
$ ork get controllers
Controllers:
- project
- managednamespace
```

---

# 🧪 **Version**

```bash
$ ork version
Orkestra v0.4.0
Mode: Go
Katalog: initialize/katalog.go
```

---

# 📄 **Summary**

The Orkestra CLI gives you:

- full visibility into the Katalog  
- dependency graph visualization  
- controller introspection  
- CRD metadata exploration  
- runtime debugging tools  

It’s designed to feel natural for Kubernetes engineers while exposing the unique power of Orkestra’s runtime architecture.