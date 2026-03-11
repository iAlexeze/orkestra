# 🎼 **OrKestra — The Universal CRD Runtime for Kubernetes**  
### *Kompose. Konduct. OrKestrate.*

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

# Why Orkestra?

Traditional operator frameworks require:

- hand‑rolled clients  
- hand‑rolled informers  
- repetitive boilerplate  
- static wiring  
- controller‑runtime magic  
- one controller per CRD  

Orkestra replaces all of that with a **declarative katalog** and a **runtime engine** that:

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

# Example CRDs

| Resource           | API Group               | Version     | Status |
|-------------------|--------------------------|-------------|--------|
| `Project`          | `platform.orkestra.io`  | `v1alpha1`  | ✅ Production Ready |
| `ManagedNamespace` | `platform.orkestra.io`  | `v1alpha1`  | ✅ Production Ready |

Adding new CRDs takes **minutes**, not days.

---

# Architecture Overview

```mermaid
flowchart TB
 subgraph CRDs["Custom Resources (Configurable)"]
        P["Project CRD<br>(enabled: true, workers: 3, resync: 10m)"]
        M["ManagedNamespace CRD<br>(enabled: true, workers: 2, resync: 30s, dependsOn: project)"]
        D["Disabled CRD<br>(enabled: false)"]
  end
 subgraph API["Kubernetes API Server"]
  end
 subgraph Core["Core Komponents"]
        KC["KubeClient"]
        FC["SharedClientFactory"]
        EV["Event Recorder"]
        HS["Health Server"]
        WQ["Workqueue (Shared)"]
  end
 subgraph Informers["Informer Factory"]
    direction LR
        SIF["SharedInformerFactory"]
        PI["Project Informer<br>resync: 10m"]
        MI["ManagedNamespace Informer<br>resync: 30s"]
  end
 subgraph Katalog["Runtime Katalog"]
        REG["Kontroller Katalog"]
        R1["Project Entry<br>GVK → Reconciler"]
        R2["ManagedNamespace Entry<br>GVK → Reconciler"]
  end
 subgraph Workers["Per-CRD Worker Pools"]
        W1["Project Workers<br>(3)"]
        W2["ManagedNamespace Workers<br>(2)"]
  end
 subgraph Kontroller["Kontroller (Dependency-Aware)"]
    direction LR
        C["Dependency Kontroller"]
        Workers
        DISPATCH["GVK Dispatch Logic"]
  end
 subgraph HA["High Availability"]
        LE["Leader Election"]
  end
 subgraph Reconcilers["Reconciler Katalog"]
        PR["Project Reconciler"]
        MR["ManagedNamespace Reconciler"]
  end
    CRDs L_CRDs_API_0@--> API
    API L_API_KC_0@--> KC
    KC --> FC
    FC L_FC_SIF_0@--> SIF
    SIF L_SIF_PI_0@-. enabled: true .-> PI & MI
    SIF L_SIF_DI_0@-. enabled: false .-> DI["(Skipped)"]
    PI L_PI_STORE_0@--> STORE[("Shared Store")] & WQ
    MI L_MI_STORE_0@--> STORE & WQ
    WQ L_WQ_C_0@--> C
    C L_C_Workers_0@--> Workers & HS
    Workers L_Workers_DISPATCH_0@--> DISPATCH
    DISPATCH L_DISPATCH_REG_0@--> REG
    REG L_REG_R1_0@--> R1 & R2
    R1 L_R1_PR_0@-.-> PR
    R2 L_R2_MR_0@-.-> MR
    PR L_PR_EV_0@--> EV
    MR L_MR_EV_0@--> EV
    LE L_LE_C_0@--> C
    EV L_EV_API_0@--> API

     D:::disabled
     DI:::disabled
    classDef disabled fill:#FFB3B3,stroke:#333,stroke-width:2px,stroke-dasharray: 5 5
    style D fill:#FFB3B3,stroke:#333,stroke-width:2px
    style KC fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
    style FC fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
    style SIF fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
    style REG fill:#FF6D00,stroke:#333,stroke-width:4px,color:#FFFFFF
    style C fill:#FF6D00,stroke:#333,stroke-width:4px,color:#FFFFFF
    style Workers fill:#FFD966,stroke:#333,stroke-width:2px
    style LE fill:#FF6D00,stroke:#333,stroke-width:2px
    style PR fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
    style MR fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
    style DI fill:#FFB3B3,stroke:#333,stroke-width:2px,stroke-dasharray: 5 5

    L_CRDs_API_0@{ animation: fast } 
    L_API_KC_0@{ animation: fast } 
    L_FC_SIF_0@{ animation: fast } 
    L_SIF_PI_0@{ animation: fast } 
    L_SIF_MI_0@{ animation: fast } 
    L_SIF_DI_0@{ animation: fast } 
    L_PI_STORE_0@{ animation: fast } 
    L_PI_WQ_0@{ animation: fast } 
    L_MI_STORE_0@{ animation: fast } 
    L_MI_WQ_0@{ animation: fast } 
    L_WQ_C_0@{ animation: fast } 
    L_C_Workers_0@{ animation: fast } 
    L_C_HS_0@{ animation: fast } 
    L_Workers_DISPATCH_0@{ animation: fast } 
    L_DISPATCH_REG_0@{ animation: fast } 
    L_REG_R1_0@{ animation: fast } 
    L_REG_R2_0@{ animation: fast } 
    L_R1_PR_0@{ animation: slow } 
    L_R2_MR_0@{ animation: slow } 
    L_PR_EV_0@{ animation: fast } 
    L_MR_EV_0@{ animation: fast } 
    L_LE_C_0@{ animation: fast } 
    L_EV_API_0@{ animation: fast }
```

👉 **See:** [Archtecture Deep Dive](./docs/architectural-deep-dive.md) for a full breakdown.

---

# Key Features

## Multi‑CRD Support with Zero Boilerplate
Each CRD contributes only:

- API types  
- Reconciler  

Everything else is generated dynamically.

## Dependency‑Aware Kontroller
CRDs declare dependencies:

```yaml
dependsOn: ["project"]
```

Orkestra:

- starts CRDs in topological order  
- shuts them down in reverse order  
- ensures correctness across multi‑CRD systems  

## Per‑CRD Resync Intervals
Each CRD can define its own resync:

```yaml
resync: 10m
```

Orkestra applies it automatically when creating informers.

## Per‑CRD Worker Pools
Each CRD defines its own concurrency:

```yaml
workers: 5
```

High‑throughput CRDs scale independently.

## Dual Katalog Architecture (Go + YAML)
Two modes:

### **Go Mode (Typed)**
- full type safety  
- automatic scheme registration  

### **YAML Mode (Dynamic)**
- load CRDs from local or remote YAML  
- GitOps‑friendly  
- perfect for multi‑cluster orchestration  

**👉 See:** [What is a Katalog](./docs/katalog.md) for a full breakdown.

## Built‑in Metrics
Prometheus metrics include:

- queue depth per CRD  
- reconcile duration  
- reconcile totals  
- worker utilization  

## High Availability
- leader election  
- warm caches in all replicas  
- instant failover  

## Graceful Shutdown
- stops accepting new items  
- drains workers  
- shuts down CRDs in dependency order  

**👉 See:** [Startup & Shutdown](./docs/startup-shutdown.md) for a full breakdown)

---

# Quick Start

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
export KATALOG_MODE=YAML
export KATALOG_PATH=initialize/crd-katalog.yaml
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

# 🖥️ **Orkestra CLI**

Orkestra ships with a powerful command‑line interface (`ork`) that lets you explore, visualize, and interact with the Katalog and the running controller runtime.  
It’s designed to feel as natural as `kubectl`, but focused entirely on CRDs, reconcilers, and dependency orchestration.

The CLI supports:

- inspecting the Katalog  
- visualizing dependency graphs  
- exploring CRD metadata  
- listing active controllers  
- understanding how Orkestra wires your CRDs internally  

Full documentation is available in **[Orkestra CLI](./docs/cli.md)**.


# 🔧 Extending Orkestra: Add a New CRD in Minutes

You only write:

1. API types  
2. Reconciler  
3. Katalog entry (Go or YAML)  

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

**👉 See:** [YAML Mode Use Cases](./docs/yaml-mode-use-cases.md)

---

# 🙏 Acknowledgments

Built on top of the Kubernetes `client-go` libraries and inspired by the patterns used in the Kubernetes controller manager.

---

# 📄 License

MIT License — see [LICENSE](./LICENSE).
