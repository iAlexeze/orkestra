# 🎼 **Orkestra — The Universal CRD Runtime for Kubernetes**  
### *Compose. Conduct. Orchestrate.*

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.28+-326CE5.svg)](https://kubernetes.io/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Orkestra** is a **runtime‑composable, dependency‑aware, multi‑CRD operator engine** for Kubernetes.  
It eliminates boilerplate by generating clients, informers, reconcilers, workers, and lifecycle orchestration **dynamically at runtime** — from Go or YAML definitions.

You write **API types** and **reconcilers**.  
Orkestra builds everything else.

It is not just a controller.  
It is a **universal CRD runtime**.

---

# 🚀 Why Orkestra?

Traditional operator frameworks require:

- hand‑rolled clients  
- hand‑rolled informers  
- repetitive boilerplate  
- static wiring  
- controller‑runtime magic  
- one controller per CRD  

Orkestra replaces all of that with a **declarative registry** and a **runtime engine** that:

- loads CRDs dynamically (Go or YAML)  
- builds clients and informers automatically  
- constructs a dependency graph  
- assigns per‑CRD workers  
- applies per‑CRD resync intervals  
- dispatches events by GVK  
- orchestrates startup/shutdown in dependency order  
- exposes built‑in metrics  
- runs with high availability  

This is operator engineering without the pain.

---

# 🎯 Supported CRDs (Example)

| Resource           | API Group               | Version     | Status |
|-------------------|--------------------------|-------------|--------|
| `Project`          | `platform.ialexeze.io`  | `v1alpha1`  | ✅ Production Ready |
| `ManagedNamespace` | `platform.ialexeze.io`  | `v1alpha1`  | ✅ Production Ready |

Adding new CRDs takes **minutes**, not days.

---

# 🏗️ Architecture Overview

```mermaid
flowchart TB
 subgraph CRDs["Custom Resources"]
        P["Project CRD"]
        M["ManagedNamespace CRD"]
  end
 subgraph API["Kubernetes API Server"]
  end
 subgraph Core["Core Components"]
        KC["KubeClient"]
        FC["SharedClientFactory"]
        EV["Event Recorder"]
        HS["Health Server"]
        WQ["Workqueue (Shared)"]
  end
 subgraph Informers["Informer Layer"]
        SIF["SharedInformerFactory"]
        PI["Project Informer"]
        MI["ManagedNamespace Informer"]
        STORE[("Shared Store")]
  end
 subgraph Registry["CRD Registry"]
        REG["Kontroller Registry"]
        R1["Project Entry"]
        R2["ManagedNamespace Entry"]
  end
 subgraph Kontroller["Kontroller Layer"]
        C["Dependency Kontroller"]
        W1["Worker 1"]
        W2["Worker 2"]
        W3["Worker N"]
        DISPATCH["Dispatch Logic"]
  end
 subgraph HA["High Availability"]
        LE["Leader Election"]
  end
 subgraph Reconcilers["Reconciler Layer"]
        PR["Project Reconciler"]
        MR["ManagedNamespace Reconciler"]
  end
    CRDs --> API
    API --> KC
    KC --> FC
    FC --> SIF
    SIF --> PI & MI
    PI --> STORE & WQ
    MI --> STORE & WQ
    WQ --> C
    C --> W1 & W2 & W3 & HS
    W1 --> DISPATCH
    W2 --> DISPATCH
    W3 --> DISPATCH
    DISPATCH --> REG
    REG --> R1 & R2
    R1 --> PR
    R2 --> MR
    PR --> EV
    MR --> EV
    LE --> C
    EV --> API

    style KC fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
    style FC fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
    style SIF fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
    style REG fill:#FF6D00,stroke:#333,stroke-width:4px,color:#FFFFFF
    style C fill:#FF6D00,stroke:#333,stroke-width:4px,color:#FFFFFF
    style LE fill:#FF6D00,stroke:#333,stroke-width:2px
    style PR fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
    style MR fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
    style CRDs fill:transparent,stroke:#333,stroke-width:2px
    style API fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF

    L_CRDs_API_0@{ animation: fast } 
    L_API_KC_0@{ animation: fast } 
    L_KC_FC_0@{ animation: fast } 
    L_FC_SIF_0@{ animation: fast } 
    L_SIF_PI_0@{ animation: fast } 
    L_SIF_MI_0@{ animation: fast } 
    L_PI_STORE_0@{ animation: fast } 
    L_PI_WQ_0@{ animation: fast } 
    L_MI_STORE_0@{ animation: fast } 
    L_MI_WQ_0@{ animation: fast } 
    L_WQ_C_0@{ animation: fast } 
    L_C_W1_0@{ animation: fast } 
    L_C_W2_0@{ animation: fast } 
    L_C_W3_0@{ animation: fast } 
    L_C_HS_0@{ animation: fast } 
    L_W1_DISPATCH_0@{ animation: fast } 
    L_W2_DISPATCH_0@{ animation: fast } 
    L_W3_DISPATCH_0@{ animation: fast } 
    L_DISPATCH_REG_0@{ animation: fast } 
    L_REG_R1_0@{ animation: fast } 
    L_REG_R2_0@{ animation: fast } 
    L_R1_PR_0@{ animation: fast } 
    L_R2_MR_0@{ animation: fast } 
    L_PR_EV_0@{ animation: fast } 
    L_MR_EV_0@{ animation: fast } 
    L_LE_C_0@{ animation: fast } 
    L_EV_API_0@{ animation: fast }
```

👉 **See:** [Archtecture Deep Dive](./docs/architectural-deep-dive.md) for a full breakdown.

---

# ✨ Key Features

## 🔥 Multi‑CRD Support with Zero Boilerplate
Each CRD contributes only:

- API types  
- Reconciler  

Everything else is generated dynamically.

## 🧠 Dependency‑Aware Kontroller
CRDs declare dependencies:

```yaml
dependsOn: ["project"]
```

Orkestra:

- starts CRDs in topological order  
- shuts them down in reverse order  
- ensures correctness across multi‑CRD systems  

## 🔁 Per‑CRD Resync Intervals
Each CRD can define its own resync:

```yaml
resync: 10m
```

Orkestra applies it automatically when creating informers.

## 🧵 Per‑CRD Worker Pools
Each CRD defines its own concurrency:

```yaml
workers: 5
```

High‑throughput CRDs scale independently.

## 🧩 Dual Registry Architecture (Go + YAML)
Two modes:

### **Go Mode (Typed)**
- full type safety  
- automatic scheme registration  

### **YAML Mode (Dynamic)**
- load CRDs from local or remote YAML  
- GitOps‑friendly  
- perfect for multi‑cluster orchestration  

**👉 See:** [YAML Mode Use Cases](./docs/yaml-mode-use-cases.md)


## 📊 Built‑in Metrics
Prometheus metrics include:

- queue depth per CRD  
- reconcile duration  
- reconcile totals  
- worker utilization  

## 🛡 High Availability
- leader election  
- warm caches in all replicas  
- instant failover  

## 🧹 Graceful Shutdown
- stops accepting new items  
- drains workers  
- shuts down CRDs in dependency order  

---

# 🚀 Quick Start

## 1. Clone and Configure
```bash
git clone https://github.com/ialexeze/orkestra.git
cd orkestra
cp .env.example .env
```

## 2. Choose Your Mode

### Go Mode (default)
```bash
go run ./cmd/
```

### YAML Mode
```bash
export CRD_REGISTRY_MODE=YAML
export CRD_REGISTRY=initialize/crd-registry.yaml
go run ./cmd/
```

## 3. Install CRDs
```bash
kubectl apply -f crd/config/bases/
```

## 4. Apply Sample Resources
```bash
kubectl apply -f crd/config/samples/
```

## 5. Monitor
```bash
curl localhost:8080/metrics
curl localhost:8080/health
curl localhost:8080/ready
```

---

# 🔧 Extending Orkestra: Add a New CRD in Minutes

You only write:

1. API types  
2. Reconciler  
3. Registry entry (Go or YAML)  

Everything else is generated.

**👉 Full guide:** [Extending Orkestra](./docs/extending-orchestra.md)

---

# 🌐 YAML Mode Use Cases

YAML mode unlocks:

- centralized operator marketplaces  
- organization‑wide standardization  
- multi‑cluster fleet management  
- GitOps pipelines  
- canary deployments  
- dynamic worker scaling  
- multi‑tenant isolation  
- compliance & audit trails  
- edge/IoT deployments  
- partner integrations  

👉 See: `docs/yaml-mode-use-cases.md`

---

# 🙏 Acknowledgments

Built on top of the Kubernetes `client-go` libraries and inspired by the patterns used in the Kubernetes controller manager.

---

# 📄 License

MIT License — see [LICENSE](./LICENSE).