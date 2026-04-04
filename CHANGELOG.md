# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Control Center (New)
- **Worker pool visualization** – Real-time view of per-worker state (idle, processing, stopped) with color-coded cards and animations.
- **Worker metrics** – Display total workers, active processing count, and idle count per CRD.
- **Queue pressure warnings** – Visual alerts at 50% and 80% queue depth thresholds.
- **Clickable dependencies** – Navigate directly from CRD detail view to dependent CRDs.

### Improved

- Worker state tracking with atomic counters for accurate real-time visibility.
- CRD health endpoint now exposes `workersIdle`, `workersProcessing`, and `workerDetails`.
