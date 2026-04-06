---
title: "CRD Runtime Health"
weight: 50
description: "Orkestra tracks the health of each CRD reconciler using a lightweight,"
---

Orkestra tracks the health of each CRD reconciler using a lightweight,
concurrency‑safe structure called `CRDHealth`.  
Each CRD gets its own instance.

---

## What Health Tracks

For every CRD, Orkestra records:

- **started** — whether the reconciler has begun processing events  
- **healthy** — whether the reconciler is considered healthy  
- **totalReconciles** — total reconcile attempts  
- **failedReconciles** — number of failures  
- **consecutiveFails** — used for degradation  
- **lastError** — last error message  
- **lastReconcile** — timestamp of last reconcile  
- **startTime** — when the reconciler first started  

All fields are atomic and safe for concurrent updates.

---

## When Health Changes

### On success

```go
RecordSuccess()
```

- increments total reconciles  
- resets consecutive failures  
- marks healthy  
- updates lastReconcile  

### On failure

```go
RecordFailure(err, degradeThreshold)
```

- increments total + failed reconciles  
- increments consecutive failures  
- stores lastError  
- marks unhealthy if failures exceed threshold  

### On startup failure

Used when informers or constructors fail before the reconciler starts.

---

## Degradation Logic

A CRD becomes **unhealthy** when:

```
consecutiveFails >= degradeThreshold
```

This threshold is configurable per CRD.

---

## Health Endpoints

Each CRD exposes:

- `/katalog/<crd>/health` — live health status  
- `/katalog/<crd>` — configuration + health summary  
- `/katalog` — all CRDs with health  

These endpoints power dashboards, readiness checks, and operator UIs.

---

## Why This Matters

The health system ensures:

- operators degrade gracefully  
- dashboards show real‑time status  
- automation can detect failing CRDs  
- reconcilers never silently fail  
- multi‑CRD operators remain observable  

It’s lightweight, concurrency‑safe, and designed for high‑volume controllers.