---
title: "Orkestra Runtime Flow"
weight: 2
description: "A deep dive into how events move through Orkestra — from Kubernetes → Informers → Workers → Reconciler → Template Engine..."
---

A deep dive into how events move through Orkestra — from Kubernetes → Informers → Workers → Reconciler → Template Engine → Kubernetes.

This document complements the [Architecture Overview](./architecture-overview.md) by showing the **runtime execution path** in detail.

---

## 1. **End‑to‑End Runtime Flow**

```mermaid
sequenceDiagram
    autonumber

    participant API as Kubernetes API
    participant INF as Informer
    participant WQ as Workqueue
    participant WK as Worker
    participant DIS as Dispatcher
    participant REC as GenericReconciler
    participant TMP as Template Engine
    participant K8s as Kubernetes API (Apply)

    API->>INF: CR event (Add/Update/Delete)
    INF->>WQ: Enqueue key
    WQ->>WK: Pop key
    WK->>DIS: Dispatch by GVK
    DIS->>REC: Reconcile(key)
    REC->>TMP: Resolve templates + conditions
    TMP->>K8s: Create/Update/Delete resources
    REC->>API: Emit events
```

---

## 2. **Informer → Workqueue**

Every CRD has:

- its own informer  
- its own queue  
- its own worker pool  

Informers watch the API server and enqueue keys on:

- Add  
- Update  
- Delete  

Orkestra uses **SharedIndexInformers**, so all workers share a warm cache.

---

## 3. **Worker Execution**

Workers pop keys from the queue and call:

```
safeReconcile(key)
```

This wrapper:

- catches panics  
- handles retries  
- prevents worker crashes  
- ensures at‑least‑once semantics  

---

## **4. Dispatch by GVK**

The KontrollerRegistry maps:

```
GVK → Reconciler
```

This allows Orkestra to support:

- dynamic CRDs  
- typed CRDs  
- custom reconcilers  
- hooks  
- zero‑code templates  

All through the same dispatch layer.

---

## **5. GenericReconciler**

The GenericReconciler handles:

- finalizers  
- managed labels  
- managed annotations  
- deletion  
- reconcile implementation  

Priority:

1. Go hooks  
2. Declarative templates  
3. No‑op  

---

## **6. Template Engine**

The template engine performs:

- field resolution (`{{ .spec.image }}`)
- conditional provisioning (`when:` blocks)
- resource building (Deployments, Services, Secrets…)
- drift correction (reconcile: true)
- owner references
- idempotency

This is where declarative operator logic becomes real Kubernetes objects.

---

## **7. Status + Events**

Every reconcile emits:

- Kubernetes events  
- Prometheus metrics  
- CRD health updates  

---

## **What’s Next?**

Continue to the **CRD Lifecycle** to understand how Orkestra handles:

- missing CRDs  
- deleted CRDs  
- reappearing CRDs  
- dependency‑aware activation  

👉 [CRD Lifecycle →](./crd-lifecycle.md)