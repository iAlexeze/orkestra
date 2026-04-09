# Changelog

## [Unreleased] — The Day Orkestra Grew Up*

### Added
- Introduced a **/startup probe** to cleanly separate:
  - process initialization  
  - readiness for traffic  
  - long‑term health  
- Added `startup` atomic flag to `HealthServer` with `StartupComplete()` and `SetStartupComplete()` helpers.
- Added `/startup` HTTP route with JSON responses matching `/health` and `/ready`.
- Added consistent logging and middleware support for `/startup`.

### Changed
- **Readiness is no longer tied to the Kordinator or leadership state.**  
  Orkestra now becomes Ready as soon as:
  - health server is listening  
  - webhook server (if enabled) is listening  
  - TLS certs are loaded  
- Leadership election now runs **after** Orkestra is Ready, via a post‑start hook.
- Followers are now **always Ready**, ensuring:
  - stable webhook endpoints  
  - multi‑Pod availability  
  - smooth leadership transitions  
- Kordinator health is now reflected only in `/health` and internal Orkestra health, not readiness.

### Fixed
- Resolved a long‑standing circular dependency:
  - Pod not Ready → no endpoints  
  - no endpoints → webhook unreachable  
  - webhook unreachable → informers fail  
  - informers fail → workers never start  
  - workers never start → Pod never becomes Ready  
- Fixed scenario where only the leader Pod ever reached Ready state.
- Fixed conversion webhook failures caused by missing endpoints during startup.
- Fixed premature finalizer removal during leadership transitions (the “Missing Finalizer” bug).

### Improved
- Orkestra’s lifecycle is now fully aligned with Kubernetes best practices:
  - `/startup` gates initialization  
  - `/ready` controls traffic routing  
  - `/health` reflects internal orchestration state  
- Leadership is now treated as a **role**, not a readiness condition.
- Kordinator startup is now deterministic and no longer blocked by readiness.
- Dependency graph orchestration is more stable during Pod churn and restarts.

### Notes
This release represents a major conceptual refinement:

> **Orkestra is the runtime.  
> The Konductor is the elected leader.  
> The Kordinator is the orchestrator.  
> Kubernetes only sees Orkestra.**

This separation of concerns dramatically improves reliability, startup behavior, and multi‑Pod orchestration.