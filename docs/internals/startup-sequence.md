# Orkestra Startup Sequence (Internals)

This document describes the internal startup order of Orkestra. It is not
required for using Orkestra, but is essential for contributors and maintainers.

Orkestra starts in a strictly ordered sequence. Each component must be ready
before the next can start. This ensures:

- informers never start before clients exist  
- workers never start before informers are warm  
- health endpoints are live before leader election  
- dependency ordering is respected  

---

## 1. konstructOrkestra (Wiring Phase)

`konstructOrkestra` builds the entire runtime graph:

1. Load + validate Katalog  
2. Build scheme registry (typed CRDs only)  
3. Create Kubeclient  
4. Create EventRecorder  
5. Create QueueRegistry  
6. Create SharedInformerFactory  
7. Register CRDs in the KontrollerRegistry  
8. Build dependency graph  
9. Create DependencyKontroller  
10. Register health + Katalog endpoints  

Nothing is started yet — this is pure wiring.

---

## 2. Orkestra.Start()

The orchestrator starts components in this order:

1. **HealthServer**  
   - Routes registered  
   - Must be live before anything else  
   - Last to stop during shutdown  

2. **Kubeclient**  
   - REST config  
   - Dynamic client  
   - Clientset  

3. **EventRecorder**  
   - Depends on Kubeclient  

4. **QueueRegistry**  
   - Per‑CRD queues  

5. **Default Workqueue**  
   - Shared fallback queue  

6. **SharedInformerFactory**  
   - Starts all informers  
   - Warm caches  
   - Signals readiness  

7. **DependencyKontroller**  
   - Starts CRDs in dependency order  
   - Starts worker pools  
   - Begins retryMissingCRDs loop  

---

## 3. Leader Election

Only the elected leader runs workers.  
Followers run informers only.

On leadership loss:

- workers stop  
- lease released  
- follower takes over instantly (warm cache)  

---

## 4. Shutdown Sequence

Reverse of startup:

1. DependencyKontroller  
2. SharedInformerFactory  
3. Default Workqueue  
4. QueueRegistry  
5. EventRecorder  
6. Kubeclient  
7. HealthServer  

Workers drain before shutdown completes.
