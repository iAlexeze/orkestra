# Changelog

## [Unreleased]

### [Added]

- **Namespace protection webhook** — new admission endpoint `/namespace-protection` enforces `allowedNamespaces` / `restrictedNamespaces` rules on CRD CREATE and UPDATE operations. Registered only when `security.namespaceProtection.enabled: true`. Uses `failurePolicy: Fail` to ensure rules remain enforced during transient outages.
- **`NamespaceProtectionStats`** — dedicated stats type for the namespace protection webhook (`pkg/health`). Tracks total CREATE/UPDATE reviews, blocked, and allowed counts separately from deletion protection. Exposed under `"namespaceProtection"` in the `/katalog/{crd}` response and Control Center detail view.
- **Rollback subsystem** — declarative `operatorBox.rollback:` block adds failure-recovery to any operatorbox:
  - `trigger.consecutiveFailures` — activates rollback after N consecutive reconcile failures.
  - `trigger.withinDuration` — optional time window constraint on the failure count.
  - `onRollback` — resource group applied while rollback is active, using `.previous.*` resolver context populated from the last successful spec snapshot.
  - Spec snapshot written to `orkestra.konductor.io/previous-spec` (gzip+base64) before each spec change.
  - Rollback exits automatically when the operator updates the spec (generation change) and the next reconcile succeeds.
  - `CRDHealth` tracks `rollbackTotal`, `rollbackActive`, `rollbackLastAt` via injected callbacks; surfaced in `/katalog/{crd}` and Control Center.
- **Unified `Condition` type** — `Time`, `DayOfWeek`, `Cron`, `Duration`, `Source`, and `Notify` fields promoted onto the shared `Condition` struct. The same type is now used for template `when:`/`anyOf:`, autoscale conditions, rollback triggers, and notification blocks.
- **`TickCronWindow`** — general-purpose stateful cron window tracker in `pkg/types`. Preserves window state across evaluation ticks so cron fires between ticks are not missed. Any Orkestra component (autoscaler, future job runner) can bring its own `map[string]time.Time` and call it on each cycle.
- **Cross-binary autoscale metrics** — `ResolveCrossMetric` now falls back to an HTTP call to the remote operator's `/katalog/{crd}` endpoint when the observed CRD is not registered in `GlobalCrossMetricsRegistry` (different binary). Configured via `source.endpoint` on the `Condition`. Mirrors the same two-path pattern as `readCross` in the template engine.
- **Autoscale profiles** — five named presets (`burst`, `steady`, `batch`, `latency-sensitive`, `cost-optimized`) that expand into a complete `AutoscaleSpec` from the CRD's declared baseline.
- **`WorkerInfo` API** — `/katalog/{crd}` response now includes `autoscalerWorkers` with live semaphore state and autoscaler snapshot when `autoscale:` is declared.
- **`AutoMetrics` in `/katalog/{crd}`** — `"metrics"` key included in every CRD detail response when `autoscale:` is declared; serves as the HTTP source for cross-binary metric observation.

### [Changed]

- **`ProtectionStats` renamed `DeletionProtectionStats`** — clarifies the type is exclusively for deletion (`DELETE`) admission reviews. All callers updated.
- **`Protection` field renamed `DeletionProtection`** in `CRDInfoResponse` and Control Center types. JSON key changes from `"protection"` → `"deletionProtection"`. **Breaking change for API consumers.**
- **Autoscale `AnyOf` type** — `[]AutoscaleCondition` collapsed into `[]Condition`. `AutoscaleCondition` type removed. Time-based and field-based conditions now share the same evaluation path (`EvaluateOneCond` in `pkg/types/when.go`).
- **`EvaluateOneCond`** now handles `time:`, `dayOfWeek:`, and `cron:` conditions directly. Autoscaler no longer duplicates this logic; it builds a data map and delegates to `EvaluateWhen`.

### [Fixed]

- **`clearRollback` not called** — after a user corrected a spec that triggered rollback, `RollbackGenerationAnnotation` persisted as a stale annotation and `CRDHealth.rollbackActive` remained `true` indefinitely. Phase 6 of `reconcileImpl` now detects the stale annotation on the first successful reconcile and calls `clearRollback` before snapshotting.

---

### [Added]
- Introduced new resource builders in the Orkestra Registry to expand parity with Kubernetes primitives:
  - **StatefulSet**
  - **PersistentVolume (PV)**
  - **PersistentVolumeClaim (PVC)**
  - **PodDisruptionBudget (PDB)**
  - **HorizontalPodAutoscaler (HPA)**
  - **Ingress**
- Added `common` package for shared builder utilities:
  - `BuildResourceRequirements` — canonical Kubernetes‑native resource requirements converter
  - `ResolveNamespace(owner, spec.Namespace)` — unified namespace resolution logic
  - `parsePort`, `parseBool` — shared parsing helpers for resource builders
- Standardized resource‑builder patterns across Deployment, StatefulSet, and new resource types using the new common utilities.

### [Changed]
- Reduced duplication across all registry builders by consolidating repeated logic into `common/`.
- Updated Deployment and StatefulSet builders to use the shared `BuildResourceRequirements` and `ResolveNamespace` helpers.

### [Technical Notes]
- This work represents the first step toward a fully unified, extensible registry architecture.
- No tests have been added yet; test coverage and validation logic will be introduced in the next sprint.
