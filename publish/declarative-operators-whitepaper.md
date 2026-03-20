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

## 2. Declarative Operators

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

## 3. The Orkestra Model

Orkestra introduces two primitives:

### **Katalog**
Declares CRDs and their behavior.

### **Komposer**
Composes multiple Katalogs into a single operator.

This separation mirrors Kubernetes itself:
- Katalog = Deployment manifest  
- Komposer = Helm chart / Kustomize overlay  

---

## 4. Declarative Reconciliation

Orkestra introduces a template‑driven reconcile engine:

```yaml
onCreate:
  deployments:
    - image: "{{ .spec.image }}"
      replicas: "{{ .spec.replicas }}"
      reconcile: true
```

The runtime:
- resolves templates  
- applies resources  
- sets owner references  
- handles updates  
- manages deletion  
- enforces dependencies  

This is reconciliation as data, not code.

---

## 5. Health as a First‑Class Pattern

Every CRD gets:
- reconcile counters  
- error rates  
- uptime  
- last error  
- last reconcile  
- degradation thresholds  
- `/health` endpoint  
- `/info` endpoint  

This standardizes operator observability.

---

## 6. Composition as a Pattern

Komposers allow:
- multi‑team collaboration  
- environment‑specific overrides  
- GitOps‑friendly layering  
- Helm‑based CRD distribution  
- remote Katalog sourcing  

This enables enterprise‑scale operator ecosystems.

---

## 7. Benefits

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

---

## 8. Conclusion

Declarative operators represent the next evolution of Kubernetes extensibility.

Orkestra demonstrates that:
- operators can be declarative  
- operators can be composed  
- operators can be observable  
- operators can be safe  
- operators can be built without writing Go  

This unlocks a new class of platform engineering workflows —  
where operators are not software projects, but **infrastructure definitions**.
