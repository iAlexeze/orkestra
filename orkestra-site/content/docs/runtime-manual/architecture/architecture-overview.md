---
title: "Architecture Overview"
weight: 103
---

# Orkestra Full Architecture Overview

Orkestra is a **declarative operator runtime**.  
You describe *what* an operator should do in YAML (a **Katalog**), and Orkestra handles *how* to do it.

This document explains the full architecture of the Orkestra runtime: how CRDs are discovered, how workers are started, how reconcile events flow, and how Orkestra remains self‑healing and dependency‑aware.

---

## **1. High‑Level Overview**

Orkestra consists of four major subsystems:

1. **Pre‑Runtime (Komposer + Katalog Loader)**  
   Merges multiple Katalog sources and produces a validated runtime Katalog.

2. **Runtime Core**  
   Informers, workqueues, reconciler registry, template engine.

3. **Dependency‑Aware Kordinator**  
   Starts CRDs in topological order, handles activation/deactivation, manages worker pools.

4. **Reconciler Layer**  
   GenericReconciler + optional Go hooks + template engine.

The diagram below shows the full system:

---
```mermaid
flowchart TB

%% ============================
%% PRE-RUNTIME (KATALOG BUILD)
%% ============================
subgraph PreRuntime["Pre‑Runtime (Build Phase)"]
    direction LR
    KOMP["Komposer<br/>(merge files, Helm, URLs)"]
    KAT["Katalog Loader<br/>(validation + defaults)"]
    KOMP --> KAT
end

%% ============================
%% CRDs
%% ============================
subgraph CRDs["Custom Resources (User‑Defined)"]
    P["CRD A<br/>(enabled: true, workers: 3, resync: 10m)"]
    M["CRD B<br/>(enabled: true, workers: 2, resync: 30s, dependsOn: A)"]
    D["Disabled CRD<br/>(enabled: false)"]
end

%% ============================
%% API SERVER
%% ============================
subgraph API["Kubernetes API Server"]
end

%% ============================
%% CORE COMPONENTS
%% ============================
subgraph Core["Core Components"]
    KC["KubeClient"]
    CF["Client Factory"]
    EV["Event Recorder"]
    HS["Health Server"]
    WQ["Shared Workqueue"]
end

%% ============================
%% INFORMERS
%% ============================
subgraph Informers["Informer Factory"]
    direction LR
    SIF["SharedInformerFactory"]
    PI["Informer: CRD A<br/>resync: 10m"]
    MI["Informer: CRD B<br/>resync: 30s"]
    DI["Informer: Disabled CRD<br/>(not started)"]
end

%% ============================
%% RUNTIME KATALOG
%% ============================
subgraph RuntimeKatalog["Runtime Katalog"]
    REG["Reconciler Registry<br/>(GVK → Reconciler)"]
    R1["Entry: CRD A"]
    R2["Entry: CRD B"]
end

%% ============================
%% TEMPLATE ENGINE
%% ============================
subgraph TemplateEngine["Template Engine"]
    direction TB
    RES["Template Resolver<br/>(fields, defaults, conditions)"]
    TE["Resource Builders<br/>(Deployments, Services, Secrets…)"]
end

%% ============================
%% WORKERS + Kordinator
%% ============================
subgraph Kordinator["Dependency‑Aware Kordinator"]
    direction LR
    C["Dependency Kordinator"]
    subgraph Pools["Worker Pools"]
        W1["Workers: CRD A (3)"]
        W2["Workers: CRD B (2)"]
    end
    DISPATCH["GVK Dispatch"]
end

%% ============================
%% RECONCILERS
%% ============================
subgraph Reconcilers["Reconciler Layer"]
    GR["GenericReconciler<br/>(zero‑code)"]
    subgraph Hooks["Optional Hooks"]
        H1["OnReconcile"]
        H2["OnDelete"]
        H3["OnNotFound"]
    end
end

%% ============================
%% HIGH AVAILABILITY
%% ============================
subgraph HA["High Availability"]
    LE["Leader Election"]
end

%% ============================
%% CONNECTIONS
%% ============================
CRDs --> API
API --> KC
KC --> CF
CF --> SIF
SIF -.->|enabled| PI & MI
SIF -.->|disabled| DI

PI --> WQ
MI --> WQ

WQ --> C
C --> Pools
Pools --> DISPATCH
DISPATCH --> REG
REG --> R1 & R2
R1 --> GR
R2 --> GR

GR --> RES
RES --> TE
GR --> EV

LE --> C
EV --> API

KAT --> REG

%% ============================
%% STYLING
%% ============================
style D fill:darkred,stroke:#333,stroke-width:2px
style DI fill:darkred,stroke:#333,stroke-width:2px,stroke-dasharray: 5 5
style KC fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
style CF fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
style SIF fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
style REG fill:#FF6D00,stroke:#333,stroke-width:4px,color:#FFFFFF
style C fill:#FF6D00,stroke:#333,stroke-width:4px,color:#FFFFFF
style Pools fill:darkgreen,stroke:#333,stroke-width:2px
style GR fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
style LE fill:#FF6D00,stroke:#333,stroke-width:2px
style Hooks fill:nofill,stroke:#333,stroke-width:2px
style TemplateEngine stroke:#333,stroke-width:2px,color:#FFFFFF

%% ============================
%% ANIMATIONS
%% ============================
%% L_CRDs_API_0@{ animation: fast }
%% L_API_KC_0@{ animation: fast }
%% L_CF_SIF_0@{ animation: fast }
%% L_SIF_PI_0@{ animation: fast }
%% L_SIF_MI_0@{ animation: fast }
%% L_SIF_DI_0@{ animation: fast }
%% L_PI_WQ_0@{ animation: fast }
%% L_MI_WQ_0@{ animation: fast }
%% L_WQ_C_0@{ animation: fast }
%% L_C_Pools_0@{ animation: fast }
%% L_Pools_DISPATCH_0@{ animation: fast }
%% L_DISPATCH_REG_0@{ animation: fast }
%% L_REG_R1_0@{ animation: slow }
%% L_REG_R2_0@{ animation: slow }
%% L_R1_GR_0@{ animation: slow }
%% L_R2_GR_0@{ animation: slow }
%% L_GR_RES_0@{ animation: slow }
%% L_RES_TE_0@{ animation: slow }
%% L_GR_EV_0@{ animation: fast }
%% L_LE_C_0@{ animation: fast }
%% L_EV_API_0@{ animation: fast }
```

---

## **2. Pre‑Runtime: Komposer + Katalog Loader**

Before Orkestra starts reconciling anything, it builds the **runtime Katalog**:

- Merge multiple files, Helm charts, URLs  
- Apply overrides  
- Validate schema  
- Apply defaults  
- Build dependency graph  
- Register CRDs and reconciler configs  

This produces a complete, normalized operator definition.

---

## **3. Informer Factory**

For each CRD:

- A SharedIndexInformer is created  
- If the CRD exists → informer starts  
- If missing → informer is created but not started  
- When the CRD appears → retry loop activates it  

Informers feed events into per-CRD workqueue.

---

## **4. Dependency‑Aware Kordinator**

The Kordinator:

- Computes startup order (topological sort)
- Waits for dependencies using ready channels
- Starts internal kontroller with worker pools per CRD
- Monitors CRD existence
- Deactivates CRDs when deleted
- Reactivates CRDs when they reappear
- Shuts down in reverse dependency order

This is the heart of Orkestra’s self‑healing model.

---

## 5. **Reconciler Registry**

Each CRD maps to a **Reconciler**:

- GenericReconciler (zero‑code)
- Optional Go hooks (OnReconcile, OnDelete, OnNotFound)
- Template Engine (Deployments, Services, Secrets, ConfigMaps, Jobs…)

The registry dispatches events to the correct reconciler based on GVK.

---

## 6. **Template Engine**

The template engine handles:

- Field resolution (`{{ .spec.image }}`)
- Conditional provisioning (`when:` blocks)
- Resource builders (Deployment, Service, Secret…)
- Drift correction (reconcile: true)
- Owner references
- Idempotency

This is where declarative operator logic becomes real Kubernetes objects.

---

Continue to the **Runtime Flow** to understand how Orkestra handles:

- workers
- informers
- reconciler

👉 [Runtime Flow →](./runtime-flow.md)

