# **The Orkestra Katalog**
### *A Declarative Bundle for CRDs, Behavior, and Runtime Orchestration*

The **Katalog** is the heart of Orkestra's architecture.  
It is a **declarative, composable, dependency‑aware bundle** that describes:

- what CRDs exist
- how they behave
- how they depend on each other
- how they should be wired at runtime
- how Orkestra should generate clients, informers, reconcilers, and workers
- where to fetch API types and hooks from

It is the single source of truth for the entire operator runtime.

If the Kubernetes API is the "data plane,"  
the **Katalog is the control plane** for Orkestra.

---

# What Is a Katalog?

A **Katalog** is a structured definition that contains one or more **CRD entries**.  
Each entry describes:

| Field | Purpose |
|-------|---------|
| `name` | Unique identifier |
| `enabled` | Whether the CRD is active at runtime |
| `group`/`version`/`kind`/`plural` | API metadata |
| `workers` | Concurrency per CRD |
| `resync` | Reconcile frequency |
| `dependsOn` | Startup/shutdown dependencies |
| `apiTypes` | Location of Go API types (for YAML mode) |
| `reconciler` | How to handle reconciliation (default, hooks, or custom) |
| `queue` | Queue configuration (per‑CRD or shared) |
| `finalizers` | List of finalizers to manage |

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

### 🔹 Remote API type fetching
Specify `apiTypes.location` and Orkestra fetches the Go types at generation time — they can live in completely separate repositories.

### 🔹 Dependency‑aware orchestration
CRDs start in topological order and shut down in reverse order.

### 🔹 Per‑CRD workers and resync
Each CRD defines its own concurrency and refresh interval.

### 🔹 Pluggable reconcilers
Three levels of involvement:
- **Zero‑code** (`reconciler.default: true`)
- **Hook‑based** (add only the hooks you need)
- **Full custom** (implement the entire reconciler)

### 🔹 Remote hooks
Specify `hooks.location` and Orkestra fetches your business logic dynamically.

### 🔹 Multi‑CRD operator composition
Build operators with 2, 5, or 50 CRDs — all from one Katalog.

---

# Katalog Structure

A Katalog is composed of **CRD entries**:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: platform-katalog
spec:
  finalizers:
    - my-default-finalizer/platform-katalog

  crds:
    # ── Zero‑code CRD ──────────────────────────────────
    - name: project
      enabled: true
      group: platform.orkestra.io
      version: v1alpha1
      kind: Project
      plural: projects
      workers: 3
      resync: 10m
      dependsOn: []
      
      apiTypes:
        location: github.com/ialexeze/orkestra/example-crds/api/types/project/v1alpha1
      
      reconciler:
        default: true  # No code required!

    # ── Hook‑based CRD ─────────────────────────────────
    - name: managednamespace
      enabled: true
      group: platform.orkestra.io
      version: v1alpha1
      kind: ManagedNamespace
      plural: managednamespaces
      workers: 2
      resync: 30s
      dependsOn: ["project"]
      
      apiTypes:
        location: github.com/ialexeze/orkestra/example-crds/api/types/managedNamespace/v1alpha1
      
      reconciler:
        default: true
        hooks:
          location: github.com/yourorg/your-hooks
          package: hooks.ManagedNamespaceHooks

    # ── Full custom CRD ────────────────────────────────
    - name: application
      enabled: true
      group: platform.orkestra.io
      version: v1alpha1
      kind: Application
      plural: applications
      workers: 2
      resync: 3s
      dependsOn: ["project", "managednamespace"]
      
      apiTypes:
        location: github.com/ialexeze/orkestra/example-crds/api/types/application/v1alpha1
      
      reconciler:
        default: false
        constructor: "reconciler.NewApplicationReconciler"
      
      queue:
        maxQueueDepth: 500
```

This is the declarative definition that Orkestra uses to build the runtime.

---

# How Orkestra Uses the Katalog

When Orkestra starts:

1. **Load the Katalog** (Go or YAML)
2. **Filter enabled CRDs**
3. **Validate dependency graph** (detect cycles, missing deps)
4. **Run `ork generate registry`** (if YAML mode)
   - Fetch API types from `apiTypes.location`
   - Fetch hooks from `hooks.location`
   - Generate `initialize/registry.go` with all bindings
5. **Build scheme registry** from generated code
6. **Generate clients** via SharedClientFactory
7. **Generate informers** with per‑CRD resync
8. **Generate reconcilers** (generic, hooks, or custom)
9. **Build dependency‑aware controller**
10. **Start workers per CRD** in topological order
11. **Begin event dispatching**

Everything is driven by the Katalog.

---

# The `ork generate registry` Command

This is where the **magic happens**. Given a Katalog, Orkestra:

```bash
ork generate registry --katalog initialize/crd-katalog.yaml
```

**What it does:**

1. Reads all enabled CRDs
2. For each CRD with `apiTypes.location`, runs `go get` to fetch the package
3. For each CRD with `hooks.location`, runs `go get` to fetch the hooks
4. Generates `initialize/registry.go` containing:
   - `RegisterRuntimeObjects()` – object factories for all CRDs
   - `RegisterScheme()` – scheme registration for all CRDs
5. Wires everything together

**If `go mod tidy` is needed, run it once. Done.**

This means API types and hooks can live in **completely separate repositories**, versioned independently, and Orkestra consumes them as first‑class Go modules.

---

# Go Mode vs YAML Mode

The Katalog supports two modes:

## **Go Mode**
- CRDs defined in Go (`initialize/crd-katalog.go`)
- Strong typing
- Automatic scheme registration (no `ork generate` needed)
- Best for framework authors and core development

## **YAML Mode**
- CRDs defined in YAML
- GitOps‑friendly
- Remote katalogs supported (HTTP/HTTPS URLs)
- Requires `ork generate registry` once
- Best for platform teams and multi‑cluster orchestration

Both modes produce the **same runtime behavior**.

---

# Dependency Graph

Each CRD can declare:

```yaml
dependsOn:
  - project
  - managednamespace
```

Orkestra builds a DAG and:

- detects cycles
- validates all dependencies exist
- computes topological order
- starts CRDs in dependency order
- shuts them down in reverse order
- ensures cross‑CRD correctness

This is essential for multi‑CRD operators where resources depend on each other.

---

# CLI Integration

The Katalog is fully introspectable via the CLI:

```bash
# List all CRDs in the Katalog
ork katalog list

# Show detailed CRD configuration
ork katalog describe project

# Visualize dependencies
ork graph deps
ork graph order
ork graph tree

# Show runtime status
ork kontroller status
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
Fetch reconcilers dynamically from Git or OCI as compiled plugins or WASM modules.

## **3. Katalog Marketplace**
A curated library of:

- CRDs
- reconcilers
- hooks
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
Validate katalogs before runtime with schema checking and policy enforcement.

## **6. Katalog‑Driven Code Generation**
Optional codegen for teams that want static clients (escape hatch).

## **7. Katalog‑Aware UI**
A dashboard that visualizes:

- dependency graphs
- controller health
- worker pools
- reconcile metrics
- live CR counts

---

# Summary

The **Katalog** is the foundation of Orkestra's runtime.  
It is:

- **declarative** – define, don't code
- **composable** – mix and match CRDs
- **dependency‑aware** – topological ordering
- **mode‑agnostic** – Go or YAML
- **introspectable** – CLI and API
- **extensible** – hooks, custom reconcilers, remote types

It transforms operator engineering from boilerplate into orchestration.

Use it to define your CRDs.  
Use it to wire your controllers.  
Use it to build multi‑CRD systems with elegance and clarity.