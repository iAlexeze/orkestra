# Why Orkestra?

Kubernetes has always promised declarative infrastructure.  
But operators — the very mechanism used to extend Kubernetes — have remained stubbornly imperative.

Every operator framework to date has required:
- writing Go
- scaffolding boilerplate
- generating code
- wiring informers
- managing schemes
- building controllers
- maintaining custom reconcilers

This creates a paradox:  
**to make Kubernetes more declarative, you must write imperative code.**

Orkestra breaks that paradox.

---

## The Core Idea

**Orkestra makes operators declarative.  
Fully. End‑to‑end. No Go required.**

A user writes a Katalog YAML describing:
- the CRDs they want to manage
- the templates that define how resources should be reconciled
- optional hooks, constructors, and typed APIs

Then they run:

```
ork run --katalog ./katalog.yaml
```

And a production‑grade operator comes to life.

No code generation.  
No compilation.  
No boilerplate.  
No controller-runtime.  
No Kubebuilder scaffolding.

Just intent → behavior.

---

## Why This Matters

### 1. **Operators become accessible**
You no longer need to be a Go engineer to build an operator.  
Platform teams, SREs, and application owners can define behavior declaratively.

### 2. **Operators become portable**
A Katalog is just YAML.  
It can be versioned, templated, rendered, diffed, and promoted like any other manifest.

### 3. **Operators become composable**
Komposers allow teams to merge Katalogs from:
- Git repos  
- Helm charts  
- remote URLs  
- local files  
- environment‑specific overrides  

This creates a clean separation between:
- **CRD definition**  
- **CRD composition**  
- **CRD operation**

### 4. **Operators become observable**
Every CRD automatically gets:
- `/katalog/{crd}` info endpoint  
- `/katalog/{crd}/health` health endpoint  
- reconcile metrics  
- error rates  
- uptime  
- dynamic queue depth  
- worker counts  

No custom code required.

### 5. **Operators become safe**
Orkestra enforces:
- dependency ordering  
- health degradation  
- readiness gates  
- startup synchronization  
- graceful shutdown  

All without the user writing a single line of Go.

---

## The Philosophy

Orkestra is built on three principles:

### **1. Declarative First**
If Kubernetes can express it declaratively, Orkestra should too.

### **2. Composition Over Code**
Operators should be assembled, not programmed.

### **3. Runtime Over Build‑Time**
Behavior should be interpreted at runtime, not baked into binaries.

---

## The Vision

Orkestra aims to make operators:
- easier to write  
- easier to maintain  
- easier to reason about  
- easier to share  
- easier to scale  

The Kubernetes community has long needed a way to build operators without writing operators.

Orkestra is that way.
