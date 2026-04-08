# Changelog

## [Unreleased] 

### Fixed
- **Dependency Kordinator blocking on `healthy` condition**  
  The main startup loop no longer blocks when a CRD requires a dependency to be `healthy`. CRDs that are not immediately ready are deferred and activated later by the background retry loop, eliminating starvation of alphabetically later CRDs in the same topological tier.

- **UI timestamp display**  
  `StartedAgo` and `LastReconcileAgo` now correctly display relative times. Backend timestamps were changed from a custom format to RFC3339, ensuring reliable parsing in the frontend.

### Added
- **External API call metrics**  
  - `orkestra_external_calls_total` – counts calls by CRD, name, URL, and result (success/error).  
  - `orkestra_external_call_duration_seconds` – histogram of external call latency.  
  - `orkestra_external_call_errors_total` – counts errors by error type.  
  These metrics provide deep visibility into operator dependencies on external services.

### Changed
- **Timestamp formatting for health endpoints**  
  `StartedAt()` and `LastReconcileAt()` now return UTC timestamps in RFC3339 format (`2026-04-08T21:46:11Z`) for consistent machine parsing.

### Improved
- **Health state consistency**  
  CRD health now correctly reflects `degraded` state when dependencies are unhealthy or missing. Dependents continue running in a degraded mode rather than blocking.

- **Retry loop enhancements**  
  Phase 3 added to periodically evaluate deferred CRDs (skipped at startup due to unsatisfied dependencies). Activation occurs immediately once conditions are met.

### Tested
- **End‑to‑end validation**  
  Verified startup order with mixed `started` and `healthy` dependencies, runtime CRD deletion/recreation, and external API call metric emission. All scenarios behave as expected.

---