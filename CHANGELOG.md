# Changelog

## [Unreleased] — CHANGELOG — Rename DependencyKontroller → DependencyKordinator

### **Changed**
- Renamed package **`kontroller` → `kordinator`**.
- Renamed struct **`DependencyKontroller` → `DependencyKordinator`**.
- Updated constructor to **`NewDependencyKordinator`**.
- Updated all imports, references, and type usages accordingly.

### **Rationale**
The component previously named *DependencyKontroller* no longer behaved like a traditional controller.  
It evolved into a **coordination layer** responsible for orchestrating multiple subsystems:

- CRD startup sequencing based on dependency graph  
- Worker lifecycle coordination  
- Health propagation (CRD → Orkestra → Katalog)  
- Queue registry and informer registry wiring  
- Safe reconcile orchestration  
- Dependency gating via `startedCh` and `healthyCh`  
- Ordered shutdown and draining  
- Centralized access to kubeclient, events, CRD health map, and reconcilers  

Because it **coordinates** controllers rather than *being* one, the name “Kontroller” became misleading.

The new name **Kordinator** reflects its true role:

> A system‑level orchestrator that brings together controllers, queues, informers, health, and dependency logic into a single coordination unit.

### **Impact**
- No functional behavior changed.
- No API semantics changed.
- Only naming and package paths updated for clarity.
- Existing controllers, reconcilers, and workers continue to operate unchanged.
- Improves architectural readability and contributor understanding.

### **Migration Notes**
- Update imports from `kontroller` → `kordinator`.
- Replace references to `DependencyKontroller` with `DependencyKordinator`.
- No other code changes required.
