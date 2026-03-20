# Operator Patterns: The Declarative Way

## Abstract

Operators are the backbone of modern Kubernetes platforms.  
They encode domain knowledge, automate reconciliation, and extend the API surface.

But operator development has remained:
- code‑heavy  
- boilerplate‑heavy  
- framework‑heavy  
- inaccessible to non‑Go engineers  

This whitepaper introduces a new model:  
**Declarative Operators**, powered by Orkestra.

---

## 1. The Problem with Traditional Operators

Traditional operator frameworks (Kubebuilder, Operator SDK, Metacontroller) require:
- writing Go controllers
- generating deep‑copy code
- wiring schemes and informers
- managing reconcile loops manually
- handling errors, retries, and backoff
- building and publishing images

This creates:
- high cognitive load  
- steep learning curves  
- slow iteration cycles  
- fragmented patterns  
- inconsistent implementations  

Operators become *software projects*, not *infrastructure definitions*.

---

## 2. The Operator Sprawl Crisis

Modern platforms often run dozens of operators — one for Prometheus, one for Cert Manager, one for PostgreSQL, one for Kafka, and so on. Each operator:

- consumes memory and CPU
- requires its own update cycle
- has its own configuration format
- exposes its own metrics (or none at all)
- implements its own health checking
- handles dependencies differently (or not at all)

This sprawl creates operational debt that grows with every new operator added. Platform teams spend more time managing operators than the resources those operators manage.

Orkestra's multi‑CRD runtime collapses this sprawl. One runtime manages many CRDs, with consistent behavior, unified observability, and shared infrastructure.

---

## 3. Declarative Operators

A declarative operator is defined entirely in YAML:

- CRDs  
- reconcile templates  
- hooks  
- constructors  
- dependencies  
- workers  
- resync periods  
- queue depth  
- health thresholds  

The operator runtime interprets these declarations at runtime.

No code.  
No build step.  
No scaffolding.

---

## 4. The Orkestra Model

Orkestra introduces two primitives:

### **Katalog**
Declares CRDs and their behavior.

### **Komposer**
Composes multiple Katalogs into a single operator.

This separation mirrors Kubernetes itself:
- Katalog = Deployment manifest  
- Komposer = Helm chart / Kustomize overlay  

**Why two primitives?**  
A Katalog defines *what* an operator does. A Komposer defines *where* its definitions come from — files, Helm charts, remote URLs, or environment variables. This separation allows teams to own their CRD definitions while platform teams own composition.

---

## 5. Dependency Patterns

Traditional operators handle dependencies implicitly, if at all. Orkestra makes dependencies explicit and enforceable.

### **Hard Dependencies**
A CRD that declares a hard dependency will not start its workers until the dependency is ready. If the dependency never becomes ready, the CRD remains degraded but the operator continues running.

### **Soft Dependencies**
Soft dependencies do not block startup. The CRD starts, but its health reflects the missing dependency. This is ideal for optional integrations.

### **Graph‑Based Ordering**
Orkestra computes a topological order from declared dependencies. Startup happens in dependency order; shutdown happens in reverse order. This ensures correctness across multi‑CRD systems.

---

## 6. Template‑Driven Reconciliation

Orkestra introduces a template‑driven reconcile engine:

```yaml
onCreate:
  deployments:
    - image: "{{ .spec.image }}"
      replicas: "{{ .spec.replicas }}"
      reconcile: true
```

The runtime:
- resolves templates against the live CR
- applies resources idempotently
- sets owner references for garbage collection
- handles updates when `reconcile: true`
- manages deletion via owner references or explicit cleanup jobs

This is reconciliation as data, not code.

---

## 7. The Registry Pattern

The OrkestraRegistry provides reusable implementations for standard Kubernetes resources:

- Deployments
- Services
- Secrets
- ConfigMaps
- ServiceAccounts
- Jobs
- CronJobs

Each implementation handles:
- create/update/delete idempotency
- owner reference management
- drift correction
- multi‑namespace distribution (for Secrets and ConfigMaps)
- source synchronization (copy‑from patterns)

This registry is extensible — adding a new resource type means adding one file and one line of registration.

---

## 8. Health as a First‑Class Pattern

Every CRD automatically exposes:

| Endpoint | Purpose |
|----------|---------|
| `/katalog/{crd}` | Configuration and live stats |
| `/katalog/{crd}/health` | Health status with degradation details |
| `/katalog` | Aggregate view of all CRDs |

**Health fields:**
- `started` — workers are running
- `healthy` — all dependencies satisfied, reconciliation succeeding
- `errorRate` — percentage of failed reconciles
- `consecutiveFails` — current failure streak
- `lastError` — last error message
- `lastReconcile` — timestamp of last reconcile
- `resourceCount` — live CR count from informer cache

**Source tracking:**  
Fields like `workersSource` and `resyncSource` tell operators whether a value came from the Katalog or a default — zero configuration mystery.

---

## 9. Composition as a Pattern

Komposers allow:
- multi‑team collaboration  
- environment‑specific overrides  
- GitOps‑friendly layering  
- Helm‑based CRD distribution  
- remote Katalog sourcing  

**Merge rules:**
- Duplicate names across sources → error (catches mistakes)
- Inline `spec.crds` → override (explicit intent)
- Disabled CRDs → preserved (auditable)

This enables enterprise‑scale operator ecosystems where platform teams provide base Katalogs and application teams override only what they need.

---

## 10. Zero‑Code Operators: The End State

With Orkestra, an operator is a YAML file:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
spec:
  crds:
    - name: website
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites
      reconciler:
        default: true
        onCreate:
          deployments:
            - image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              reconcile: true
          services:
            - port: "80"
              targetPort: "{{ .spec.port }}"
              reconcile: true
```

```bash
ork run --katalog website.yaml
```

This is a complete, production‑ready operator:
- ✅ Manages Website CRDs
- ✅ Creates Deployments and Services
- ✅ Reconciles on changes
- ✅ Cleans up on delete
- ✅ Exposes health endpoints
- ✅ Emits Kubernetes Events
- ✅ Emits Prometheus metrics
- ✅ Handles graceful shutdown
- ✅ Supports leader election for HA

**Zero Go. Zero code generation. Zero compilation.**

---

## 11. Benefits

### **Reduced complexity**
No Go, no codegen, no scaffolding.

### **Faster iteration**
Change YAML → operator behavior changes instantly.

### **Safer operations**
Built‑in health, readiness, and metrics.

### **Better collaboration**
Teams can share Katalogs like Helm charts.

### **More predictable behavior**
Declarative patterns eliminate imperative drift.

### **Resource efficiency**
One runtime manages many CRDs, reducing memory and CPU overhead.

### **Unified observability**
Consistent health endpoints and metrics across all CRDs.

---

## 12. Conclusion

Declarative operators represent the next evolution of Kubernetes extensibility.

Orkestra demonstrates that:
- operators can be declarative  
- operators can be composed  
- operators can be observable  
- operators can be safe  
- operators can be built without writing Go  

The patterns introduced here — declarative reconciliation, dependency graphs, health as data, multi‑source composition — are not just simplifications. They are a new way of thinking about operators.

Operators are no longer software projects.  
They are infrastructure definitions.

**Orkestra is the runtime that makes this possible.**

---

*Orkestra — Declarative Operators for Kubernetes*  
*March 2026*