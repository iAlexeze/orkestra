# Full Architecture View

This page provides the complete visual representation of the Orkestra runtime.  
It complements the conceptual explanation in the  
👉 [Architecture Overview](./index.md)

The diagram below shows how Orkestra loads Katalogs, discovers CRDs, initializes informers, manages worker pools, dispatches reconcile events, and applies templates to the Kubernetes API.

---

## Full System Diagram

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
%% WORKERS + KORDINATOR
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
```

## What’s Next?

Explore the conceptual explanation:

👉 [Architecture Overview](./index.md)

Or dive into the runtime execution path:

👉 [Runtime Flow](./runtime-flow.md)
