# Changelog

## [Unreleased] — (April 5, 2026)

### Added

#### Declarative State Machines
- `operator: notExists` in `when:` conditions — detects first reconcile before any status is written
- `operator: in` with comma-separated list — `"Pending,"` matches Pending or empty (first-reconcile + phase in one condition)
- `operator: hasPrefix`, `operator: hasSuffix` condition operators
- `when:` support on individual `status.fields` entries — conditional status writes
- Last-writer-wins semantics across multiple entries for the same path — declare terminal states last
- Declarative pipeline example (`examples/phases/`) proving three-step sequential execution in YAML

#### Provider Library
- `Provider` interface (`pkg/types/provider.go`) — `Name()`, `Reconcile()`, `Delete()`
- `ProviderRegistry` — thread-safe, panic-on-duplicate, `NoOpProviderRegistry` for tests
- `KubeReader` — narrow read-only cluster access for providers (Secret, ConfigMap)
- `ProviderDeclaration.Field(key, default)` and `Require(key)` convenience methods
- `run_providers.go` — dispatch engine: template resolution, condition filtering, ordered dispatch
- `runProviderDelete` — collects all errors before returning, attempts all providers
- `KatalogProviderRequirement` — catalog-level provider manifest (`spec.providers[]`)
- `ProviderBlock`, `RawProviderDeclaration` — per-CRD declaration types
- `ParseProviderBlocks` — converts raw YAML map to structured blocks, flattens nested fields
- `pkg/providers/aws/provider.go` — real AWS provider using SDK v2 (S3, RDS, Route53)
- `pkg/providers/mongodb/provider.go` — real MongoDB provider using Go driver (database, user, collection)
- `internal/providers.go` — `loadProviders` non-fatal registry builder
- Provider blocks captured in factory closures — no signature changes to DependencyKontroller

#### CR / Events API Endpoints
- `GET /katalog/{crd}/cr` — list all CR instances from informer cache
- `GET /katalog/{crd}/cr/{namespace}/{name}` — CR detail with children and status
- `GET /katalog/{crd}/cr/{name}` — cluster-scoped variant
- `GET /katalog/{crd}/cr/{...}/events` — recent Kubernetes events, newest-first, capped at 100
- `BuildCRListHandler`, `BuildCRDetailAndEventsHandler` registered in `konstructOrkestra` step 5b
- Parallel children fetch with 3-second hard deadline — partial results on timeout
- `hasTemplateBlocks` fast path — zero API calls for hooks/constructor-only CRDs

#### CRD and CR Generation
- `pkg/generator/crd_generator.go` — `CRDGenerator` and `CRGenerator`
- Schema inference from validation rules, mutation defaults, and template expressions
- Printer columns derived from `status.fields` (phase always first)
- Conversion webhook config generated when conversion paths are declared
- `ork generate crd --katalog katalog.yaml -o crd.yaml`
- `ork generate cr  --katalog katalog.yaml -o cr.yaml`
- `--all` flag generates one file per CRD into a directory
- `--crd` flag targets a specific CRD in a multi-CRD Katalog

#### Resolver
- `resolver.Data()` — returns the internal object map (spec, status, metadata, children)
- Used by status condition evaluation, provider declaration filtering, evaluateConditions

#### Control Center
- `cr_list.html` — CR instance table with phase badges (colour-coded by state), ready indicator, age
- `cr_detail.html` — status fields, conditions table, child resource cards, events table
- `controlcenter.go` rewritten — clean two-level router, `renderTemplate`, `renderError`, `handleNotFound`
- `client.go` — `getJSON[T]` generic helper, `FetchCRList`, `FetchCRDetail`, `FetchCREvents`
- `templatefuncs.go` — `hasPrefix`, `hasSuffix`, `phaseColor`, `phaseIcon` merged in
- "View Instances" button in `crd.html` linking to CR list with resource count badge
- Auto-refresh every 10 seconds on CR pages

#### Documentation
- `docs/papers/declarative-state-machines.md` — full before/after with Go constructor code
- `docs/papers/provider-library.md` — provider model paper with ecosystem vision
- `docs/concepts/providers.md` — two-layer model, YAML structure, error semantics
- `docs/concepts/extending-providers.md` — step-by-step guide for writing new providers
- `docs/concepts/katalog-as-source-of-truth.md` — CRD generation design and rationale
- `docs/concepts/conditional-status-fields.md` — updated with notExists and in operator
- `examples/phases/README.md` — Go constructor vs declarative comparison with production output

### Fixed

#### State Machine
- Jobs treated as terminal resources — `DeleteIfOwned` removed from `run_jobs.go`
  Previously caused build Jobs to be deleted and recreated on every reconcile cycle
- Empty string treated as `"0"` in numeric condition comparisons
  Kubernetes omits zero-value integers from unstructured status — absent field is zero, not error
- `ReadChildren` collects names unconditionally — conditions gate creation, not observation
  Previously only templates whose `when:` passed had child names collected

#### Children Map
- All children map keys lowercase throughout: `cronJob` → `cronjob`, `configMap` → `configmap`, `serviceAccount` → `serviceaccount`
- Nil guard on child resource status — newly created resources have no status block
- `default` note enforced to require 2 arguments: `{{ default .children.cronjob.status.lastScheduleTime "" }}`

#### CronJob Registry
- Added missing fields: `suspend`, `successfulJobsHistoryLimit`, `failedJobsHistoryLimit`, `concurrencyPolicy`, `startingDeadlineSeconds`
- All fields accept template expressions; `Resolve()` performs correct type conversion
- Drift detection updated to cover all five new fields
- `ork validate` no longer rejects valid CronJob Katalogs with these fields

#### CR Endpoints — Performance
- `ResourceVersion: "0"` on all List calls — serves from API server watch cache not etcd
  Detail endpoint latency: 4 seconds → <50ms
  Deployment-watcher timeout (2 minutes): eliminated
- Parallel children fetch — 7 GVR queries concurrent not sequential
- 3-second deadline on children and events — never blocks on slow API server

#### deactivateCRD Drain
- Sentinel items unblock workers waiting on `GetWithContext`
  Cancelling context alone did not unblock workers; queue remains live for reactivation
  Sentinels dropped in `processItemForGVK` before reconcile dispatch

#### Control Center
- `controlcenter.go` `client` field renamed to `httpClient` — consistent across all files
- `cr_handlers.go` in kontroller — removed `mergeTemplates` and GVR var imports (import cycle)
  Children fetched by `orkestra-owner` label selector across all known GVRs instead
- `toGVR` function removed — `schema.GroupVersionResource` passed directly to dynamic client
- `crTemplateFuncs` separate file removed — merged into `templateFuncs` directly

### Changed
- `critical` field removed from CRD entries — a runtime should not crash because one CRD is struggling
- `loadProviders` called before factory loop in `konstructOrkestra` — registry captured in closures
- Control Center router rewritten as clean two-level dispatch (`routeKatalog` → `routeCR`)
- `client.go` `FetchCRDDetail` uses `humanDuration` helper — duplicate time-formatting blocks removed

### Performance
- CR list endpoint: <1ms (pure informer cache, zero API calls)
- CR detail endpoint: <50ms (was 4s–2min depending on CRD type)
- Children fetch: parallel with 3s cap (was sequential with full HTTP timeout per GVR)
- Events fetch: 3s cap (was unbounded)
- `ResourceVersion: "0"`: single largest performance improvement — eliminates etcd round-trips

---

## [Previous] — Session 6

See transcript at `/mnt/transcripts/2026-04-05-05-44-25-orkestra-build-session-6.txt`