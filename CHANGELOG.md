# CHANGELOG

### Added
- **Cross‑Operator Autoscaling**  
  Operators can now scale based on the live runtime metrics of other operators.  
  AutoMetrics is injected into the cross‑operator IPC path (`readCross`), exposing fields such as:
  - `cross.<alias>.metrics.queueDepth`
  - `cross.<alias>.metrics.workersBusyPercent`
  - `cross.<alias>.metrics.workersIdlePercent`
  - `cross.<alias>.metrics.reconcileDurationP95Ms`
  - `cross.<alias>.metrics.errorRatePercent`

  This enables upstream/downstream coordination, distributed backpressure, and pipeline‑wide adaptive behavior without introducing new APIs or subsystems.

### Enhanced
- **readCross()** now attaches runtime metrics to informer and HTTP fallback results.
- Added structured debug logs showing attached metrics and resolution path (informer vs HTTP).
- Autoscaler condition engine now supports `cross.*.metrics.*` fields with full validation.

### Documentation
- Added dedicated pages for:
  - *Cross‑Operator Autoscaling*
  - *Cross‑Operator Autoscaling Scenarios*
  - *Cross‑Operator Metrics Internals*
- Updated:
  - Autoscaler YAML Reference  
  - Autoscaler Runtime Behavior  
  - Autoscaler Scenarios  