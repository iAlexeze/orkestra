# Reconciler Pipeline (Internals)

This document describes the full reconcile pipeline inside Orkestra.

---

## 1. Event → Workqueue

Informer receives:

- Add  
- Update  
- Delete  

It enqueues the key into the shared workqueue.

---

## 2. Worker Pop

Workers pop keys and dispatch by GVK.

---

## 3. GenericReconciler

The GenericReconciler handles:

- finalizers  
- managed labels  
- managed annotations  
- deletion  
- reconcile implementation  

---

## 4. Reconcile Paths

Priority:

1. **Go hooks**  
2. **Declarative templates**  
3. **No‑op**  

---

## 5. Template Engine

- field resolution  
- conditional provisioning  
- resource builders  
- drift correction  
- owner references  

---

## 6. Events + Metrics

Every reconcile emits:

- Kubernetes events  
- Prometheus metrics  
- Health status