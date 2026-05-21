# 03 — Autoscale: Sibling‑in‑Binary

This example builds on **[02-based-on-own-metrics](../02-based-on-own-metrics/README.md)** and introduces **two new autoscaling concepts** inside a *single operator binary*:

---

## New Concepts Introduced

### **1. Autoscale Profiles (`autoscale.profile`)**
A profile is a **preset autoscaling strategy**.  
You declare intent — Orkestra computes the parameters.

Available profiles include:

- **burst** — aggressive scaling for sudden spikes  
- **steady** — smooth, conservative scaling  
- **batch** — optimized for large, predictable workloads  
- **latency-sensitive** — prioritizes fast reconciliation  
- **cost-optimized** — minimal scaling, slower but cheaper  

In this example, the **Processor** uses:

```
autoscale:
  profile: burst
```

Orkestra expands this into a full autoscale configuration:

- workers: baseline × 4  
- queueDepth: baseline × 10  
- interval: 5s  
- cooldown: 30s  
- trigger: computed from metrics.queueDepth  

You can see the computed values in the **Control Center → Autoscaler Baseline** panel.

---

### **2. Cross‑CRD Autoscaling (Sibling Metrics)**
The **Auditor** autoscaler watches **Loader’s live queue depth**:

```
field: cross.loader.metrics.queueDepth
```

This is *zero‑latency*, *in‑process*, *no network call*, *no external metrics*.

When Loader’s queue exceeds 75 for 30 seconds:

- Auditor workers scale from **1 → 4**  
- Auditor resync tightens from **60s → 15s**

This demonstrates **runtime dependency scaling** between CRDs in the same binary — without any `dependsOn` declaration.

---

## What You Learn

- How **autoscale profiles** work and how Orkestra computes full autoscale parameters from a single line.
- How **multiple CRDs inside the same operator binary** can autoscale independently.
- How to use **cross.<crd>.metrics.\*** to read sibling metrics in real time.
- How **profile‑based autoscaling** differs from **explicit autoscaling**.
- How downstream CRDs (Processor, Auditor) react differently to load on the upstream Loader.
- How Orkestra performs **cross‑operator dependency scaling** without restarts, redeployments, or external metrics.
- How the Control Center visualizes **baseline vs override** for multiple CRDs simultaneously.

---

## Prerequisites

- Ork CLI  
- Kubernetes cluster (Kind works great)

Install Ork CLI:

```bash
curl get.orkestra.sh | bash
```

Create a Kind cluster and run this example:

```bash
ork run -f katalog --dev
```

Start the Control Center:

```bash
ork control
# username:password → orkestra
```

Visit: **http://localhost:8081**

---

## Run the Example

### **1. Apply the CRDs**

```bash
kubectl apply -f crd-loader.yaml
kubectl apply -f crd-processor.yaml
kubectl apply -f crd-auditor.yaml
```

### **2. Apply the CRs**

```bash
kubectl apply -f cr-loader.yaml
kubectl apply -f cr-processor.yaml
kubectl apply -f cr-auditor.yaml
```

Watch in the Control Center as each CRD creates its Deployment/Service and begins reconciling.

> **Note:**  
> Only **Loader** has *no autoscaler*.  
> Processor and Auditor both autoscale — but in *different ways*.

---

# Load the Processor (Profile‑Based Autoscale)

```bash
./load.sh processor up 100
```

Observe:

- Processor queue grows  
- Processor autoscaler (profile: burst) activates  
- Workers scale from **2 → 8**  
- QueueDepth expands from **100 → 1000**  
- Resync tightens from **30s → 5s**  
- Loader and Auditor remain unaffected  

This demonstrates **profile‑based autoscaling on own metrics**.

---

# Load the Loader (Cross‑CRD Autoscale)

```bash
./load.sh loader up 100
```

Observe:

- Loader queue grows  
- Once queue hits **75+ for 30s**, Auditor autoscaler activates  
- Auditor workers scale **1 → 4**  
- Auditor resync tightens **60s → 15s**  
- Processor remains unaffected  

This demonstrates **cross‑CRD autoscaling** using sibling metrics.

---

# Reduce the Load

```bash
./load.sh loader down 10
```

Notes:

- Loader queue drains slowly — this is expected.  
- Auditor remains scaled up until Loader queue stays **below 75** for the full **5‑minute cooldown**.  
- After cooldown, Auditor returns to baseline (1 worker, 60s resync).

This is **runtime dependency scaling** without any explicit `dependsOn` configuration.

No restarts.  
No redeployments.  
No external metrics.  
All live, all in‑process.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
