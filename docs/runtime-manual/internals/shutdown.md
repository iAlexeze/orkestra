# Graceful Shutdown (Internals)

Orkestra guarantees:

- no dropped events  
- no double processing  
- no partial reconciles  

---

## Shutdown Order

Reverse of startup:

1. DependencyKontroller  
2. SharedInformerFactory  
3. Default Workqueue  
4. QueueRegistry  
5. EventRecorder  
6. Kubeclient  
7. HealthServer  

---

## Worker Draining

Workers:

- stop accepting new keys  
- finish in‑flight reconciles  
- wait groups ensure clean exit  
