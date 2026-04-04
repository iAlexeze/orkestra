# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Control Center
- **Worker pool visualization** – Real-time view of per-worker state (idle, processing, stopped) with color-coded cards and animations.
- **Worker metrics** – Display total workers, active processing count, and idle count per CRD.
- **Queue pressure warnings** – Visual alerts at 50% and 80% queue depth thresholds.
- **Clickable dependencies** – Navigate directly from CRD detail view to dependent CRDs.
- **Unified CRD state** – Consistent `state` field (healthy/started/pending/degraded) across all UI views.

### Changed

#### Control Center
- Worker state tracking now uses atomic counters for accurate real-time visibility.
- CRD health endpoint exposes `workersIdle`, `workersProcessing`, and `workerDetails`.
- Unified state badge logic between control panel and CRD detail views.

### Fixed

- Worker counting logic – no more negative values in worker metrics.
- Proper idle/processing state transitions in worker lifecycle.
- Worker state initialization on CRD startup.
- Clean worker state reset on CRD deactivation.
- State consistency between `katalog.html` and `crd.html` templates.