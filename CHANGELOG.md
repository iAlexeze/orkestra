# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Deterministic dependency graph** — startup order is now cached using `sync.Once`, ensuring consistent ordering across runs. Nodes are processed in sorted order for deterministic behavior. Circular dependencies are detected and reported with clear error messages.

- **Proper shutdown ordering** — CRDs now shut down in reverse topological order (dependents before dependencies), ensuring clean teardown without broken references.

- **Single retry loop for missing CRDs** — previously, multiple goroutines were started; now a single retry loop handles activation of CRDs that appear after startup, preventing race conditions.

- **Activation system** — when a missing CRD appears, its informer starts, workers begin, and the ready channel closes, unblocking any dependents waiting for it.

- **Conditional provisioning** — resources now support `when` blocks. Services are created only when conditions like `exposePublicly: true` are met, evaluated during template resolution.

- **Logging improvements** — startup and shutdown orders are now logged with arrows (e.g., `frontend → backend`) for clear visibility. Activation progress is logged with emoji indicators.

### Fixed

- **Generic reconciler no‑op issue** — `CRDInfo` now correctly passes `ReconcilerConfig` to the generic reconciler, ensuring templates are executed instead of resulting in a no‑op.

- **Dependency blocking** — dependents now correctly block on ready channels until their dependencies are ready, even when dependencies appear after startup.

- **Multiple retry loops** — retry loop now starts once instead of once per CRD, eliminating race conditions.

### Changed

- **Dependency graph** — complete rewrite for determinism and concurrency safety.
- **Startup flow** — CRDs now wait for dependencies before starting workers, ensuring correct order.
- **Shutdown flow** — now follows reverse topological order for clean teardown.

### Security

- No security changes in this release.