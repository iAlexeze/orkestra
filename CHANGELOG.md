# Changelog

## [Unreleased]

### Added

#### Provider Stats (in-memory + Prometheus)
- `pkg/health/provider_stats.go` — per-CRD in-memory provider stats (`ProviderStats`). Tracks total calls and error counts per provider name. Thread-safe via `sync.RWMutex`. Exposes `GetSnapshot()` returning `[]ProviderStatEntry` with `provider`, `total`, `errors`, and `errorRate` fields.
- `pkg/metrics/provider.go` — Prometheus counters and histogram for provider calls: `orkestra_provider_reconcile_total{crd,provider,kind,result}`, `orkestra_provider_delete_total{crd,provider,kind,result}`, `orkestra_provider_reconcile_duration_seconds{crd,provider,kind}`.

#### Provider Wiring
- `pkg/reconciler/run_providers.go` — added `providerStatsRecorder` interface and `stats` parameter to `runProviders` and `runProviderDelete`. Records `RecordSuccess`/`RecordFailure`/`RecordDeleteSuccess`/`RecordDeleteFailure` after every provider call. Nil-safe — CRDs without provider blocks pass nil stats.
- `pkg/reconciler/generic.go` — added `providerStats providerStatsRecorder` field to `GenericReconciler` and `NewGenericReconciler`.
- `cmd/internal/konstructor.go` — creates `providerStatsMap` (map from GVK to `*health.ProviderStats`) before the factory loop; passes `provStats` to both `NewGenericReconciler` (write path) and `BuildCRDInfoHandler` (read path).

#### HTTP and Control Center
- `pkg/kordinator/crd_health_handers.go` — `BuildCRDInfoHandler` gains `provStats *health.ProviderStats` parameter. When CRD declares provider blocks, the response includes a `providers` array: one entry per block with `name`, `kinds`, `total`, `errors`, `errorRate`.
- `BuildKatalogHandler` response includes `providerCount` per CRD summary row (omitted when zero).
- `cmd/controlcenter/cc/types.go` — added `ProviderInfo` type; `CRDInfo`, `CRDDetail`, and `CRDSummary` extended with provider fields.
- `cmd/controlcenter/cc/client.go` — `FetchCRDDetail` maps `info.Providers` to `detail.Providers`.
- `cmd/controlcenter/cc/template_func.go` — added `mulFloat` helper for float64 multiplication in templates.
- `cmd/controlcenter/cc/assets/templates/crd.html` — Providers section renders name, kinds, total calls, errors, and error rate.
- `cmd/controlcenter/cc/assets/templates/katalog.html` — CRD cards show provider count when non-zero.

#### New Providers (5)
- `pkg/provider/postgres/provider.go` — PostgreSQL provider (block name: `postgres`). Kinds: `database`, `role`, `extension`. Uses `github.com/jackc/pgx/v5`. Credentials via Secret key `PG_PASSWORD`. SQL injection prevention via `pgQuoteIdent`.
- `pkg/provider/redis/provider.go` — Redis cache provider (block name: `cache`). Kinds: `acluser`, `config`. Uses `github.com/redis/go-redis/v9`. ACL user management via `ACL SETUSER`; config idempotency via `CONFIG GET` before `CONFIG SET`. Credentials via Secret key `REDIS_PASSWORD`.
- `pkg/provider/mysql/provider.go` — MySQL provider (block name: `mysql`). Kinds: `database`, `user`. Uses `database/sql` + `github.com/go-sql-driver/mysql`. Idempotent via `CREATE DATABASE IF NOT EXISTS` and `CREATE USER` existence check. Credentials via Secret key `MYSQL_PASSWORD`.
- `pkg/provider/google/provider.go` — Google Cloud provider (block name: `google`). Kinds: `gcs`, `pubsub`, `cloudsql`. Uses `cloud.google.com/go/storage`, `cloud.google.com/go/pubsub`, `google.golang.org/api/sqladmin/v1`. Supports ADC (Workload Identity on GKE), service account JSON inline or file.
- `pkg/provider/azure/provider.go` — Azure provider (block name: `azure`). Kinds: `blob`, `servicebus`, `sqldatabase`. Uses `github.com/Azure/azure-sdk-for-go/sdk/azidentity` and ARM resource clients. Supports service principal credentials and DefaultAzureCredential (managed identity, CLI).

#### Provider Registration
- `cmd/internal/provider.go` — wired all 5 new providers (`postgres`, `cache`, `mysql`, `google`, `azure`) into `loadProviders`. Each follows the same required/optional error pattern as existing providers.

#### Documentation
- `pkg/provider/README.md` — updated "Current providers" table with all 7 providers.
- `docs/runtime-manual/concepts/provider.md` — updated current providers table; corrected package paths.
- `docs/technical-docs/kordinator.md` — complete rewrite based on actual source. Removed stale type names (`KordinatorRegistry`, `QueueRegistry`). Reflects real types (`ResourceKatalog`, `CRDHealth` fields, `runWorkerForGVK`, `processItemForGVK`). Added self-healing scenarios, dependency condition semantics, and key design rules.
- `docs/technical-docs/health-server.md`, `docs/technical-docs/generic-reconciler.md`, `docs/technical-docs/konstructor.md` — updated for provider stats wiring.
- `docs/runtime-manual/concepts/health-subsystem.md`, `docs/runtime-manual/concepts/observability.md` — updated provider observability sections.
