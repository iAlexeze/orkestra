# Observability (Internals)

Orkestra exposes:

- health endpoints  
- Katalog endpoints  
- metrics  
- events  

---

## Health Endpoints

- `/health` — liveness  
- `/ready` — readiness  
- `/katalog` — CRD overview  
- `/katalog/{crd}` — CRD details  
- `/katalog/{crd}/health` — CRD health  

---

## Metrics

Per‑CRD:

- reconcile count  
- reconcile duration  
- worker count  
- queue depth  
- CRD activation latency  
- CRD activation count  

---

## Events

GenericReconciler emits:

- Reconciled  
- Deleting  
- Deleted  
- Errors  
