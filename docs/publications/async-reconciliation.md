# **Async Reconciliation in Orkestra’s OperatorBox**  
### *A Declarative Model for Event‑Driven, Non‑Blocking, Multi‑Phase Operator Workflows*

## **Abstract**  
Kubernetes operators are built on a deceptively simple contract: reconcile desired state with actual state. Yet real‑world systems rarely converge in a single step. They require waiting—waiting for pods to become Ready, waiting for external systems to respond, waiting for dependencies to stabilize. Traditional operators struggle with this because the Kubernetes reconciliation model is stateless, event‑driven, and non‑blocking.  

Orkestra introduces a new execution boundary—the **OperatorBox**—that transforms asynchronous reconciliation from a hand‑crafted state machine into a **declarative, phase‑driven workflow**. This publication explains the problem, the architectural constraints, and how Orkestra’s OperatorBox provides a clean, deterministic, and safe model for async reconciliation without writing Go code.

---

# **1. The Problem: Kubernetes Reconciliation Is Synchronous, Reality Is Not**

The Kubernetes control loop is intentionally simple:

1. Receive an event  
2. Reconcile  
3. Return  

There is no concept of:

- waiting  
- blocking  
- sleeping  
- promises  
- async/await  
- long‑running tasks  

If a reconcile handler blocks, it stalls the entire controller.  
If it loops, it burns CPU.  
If it sleeps, it delays all other CRs.  

This is why async reconciliation is one of the hardest problems in operator design.  
Real systems do not converge instantly, but the reconciliation contract demands that they must.

---

# **2. Why Async Reconciliation Is Hard in Traditional Operators**

A traditional operator must manually implement:

### **2.1 Polling loops**  
Checking readiness every few seconds.

### **2.2 Backoff logic**  
Avoiding hot loops that overwhelm the API server.

### **2.3 State machines**  
Tracking partial progress across reconcile invocations.

### **2.4 Dependency graphs**  
Ensuring resource A is ready before creating resource B.

### **2.5 Error unwinding**  
Handling partial failures without corrupting state.

### **2.6 Idempotency guarantees**  
Ensuring repeated reconciles do not cause drift.

### **2.7 Cross‑resource observation**  
Reading the live state of children to determine readiness.

All of this must be hand‑written, tested, and maintained.  
Every operator author re‑implements the same patterns, often incorrectly.

---

# **3. Orkestra’s Insight: Async Is Not a Code Pattern — It’s a Declarative Phase**

Orkestra introduces the **OperatorBox**, a fully isolated execution environment for each CRD.  
Inside the OperatorBox, reconciliation becomes **phase‑driven**:

- **Phase 1:** Apply resources  
- **Phase 2:** Wait for conditions  
- **Phase 3:** Apply dependent resources  
- **Phase 4:** Update status  

Each phase is declarative.  
Each phase is non‑blocking.  
Each phase is re‑entrant.  

The OperatorBox does not “wait” in the traditional sense.  
It simply **skips phases whose conditions are not yet satisfied**, and the runtime requeues the CR automatically.

This is async reconciliation without writing async code.

---

# **4. The Core Mechanism: children.\***  

Orkestra automatically populates a live view of all child resources:

```
.children.deployment.status.readyReplicas
.children.service.status.loadBalancer
.children.ingress.status.hosts
.children.hpa.status.currentReplicas
```

This gives the operator author:

- readiness  
- health  
- rollout status  
- observed state  
- cross‑resource dependencies  

All without writing a single line of Go.

---

# **5. Declarative Async: The `when` Clause**

Async reconciliation becomes a simple declarative gate:

```yaml
onReconcile:
  when:
    - field: children.deployment.status.readyReplicas
      operator: equals
      value: "{{ .spec.replicas }}"
```

If the condition is false:

- the block is skipped  
- the CR is requeued  
- no loops  
- no sleeps  
- no blocking  
- no custom logic  

This is the essence of Orkestra’s async model.

---

# **6. OperatorBox Isolation: Why Async Is Safe**

Async reconciliation is dangerous in traditional controllers because:

- blocking one CR blocks all CRs  
- panics crash the entire controller  
- long waits stall the worker pool  

The OperatorBox eliminates these risks:

### **6.1 Per‑CRD worker pools**  
One CRD cannot starve another.

### **6.2 Panic isolation (`safeReconcile`)**  
A panic in one CR cannot crash the runtime.

### **6.3 Automatic backoff**  
Failed conditions do not create hot loops.

### **6.4 Deterministic requeue**  
The runtime handles scheduling, not the operator author.

Async becomes safe because the OperatorBox is a sandbox.

---

# **7. Multi‑Phase Reconciliation: A Real Example**

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      replicas: "{{ .spec.replicas }}"
      reconcile: true

onReconcile:
  when:
    - field: children.deployment.status.readyReplicas
      operator: equals
      value: "{{ .spec.replicas }}"

  services:
    - name: "{{ .metadata.name }}-svc"
      port: 80
      targetPort: 8080
      reconcile: true
```

This expresses:

1. Create Deployment  
2. Wait for Deployment to be Ready  
3. Create Service  

No loops.  
No sleeps.  
No custom code.  
No state machine.  
No race conditions.

This is async reconciliation as a **purely declarative workflow**.

---

# **8. Why This Matters**

Async reconciliation is the foundation of:

- multi‑step rollouts  
- dependency ordering  
- cross‑resource orchestration  
- progressive delivery  
- multi‑region fan‑out  
- external system integration  
- long‑running provisioning  

Traditional operators solve this with thousands of lines of Go.  
Orkestra solves it with **three lines of YAML**.

---

# **9. Conclusion**

Async reconciliation is not a feature bolted onto Orkestra.  
It is a natural consequence of the OperatorBox architecture:

- isolated execution  
- declarative resource groups  
- children‑based observation  
- conditional phases  
- automatic requeue  
- panic‑safe boundaries  

Orkestra transforms reconciliation from a synchronous, imperative loop into a **declarative, event‑driven, multi‑phase workflow** that matches the realities of distributed systems.

Async reconciliation is no longer something operator authors must implement.  
It is something Orkestra provides.
