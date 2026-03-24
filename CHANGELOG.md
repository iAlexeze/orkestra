# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Testing framework** — complete testing infrastructure for unit, integration, and end‑to‑end tests.
  - `tests/` directory with structure for all test types
  - Test helpers (`fake_kubeclient.go`, `testutils.go`) for integration tests
  - Fixtures (`katalogs/`, `crds/`) for reusable test data
  - Makefile targets: `test-unit`, `test-integration`, `test-e2e`, `test-coverage`, `test-all`
  - GitHub Actions workflow running unit and integration tests on every push and pull request
  - Comprehensive testing strategy documentation in `tests/README.md`

- **Testing strategy** — documented approach for achieving confidence before v1 release:
  - Unit tests for core logic (CRDHealth, dependency graph, merge rules)
  - Integration tests with fake Kubernetes clients (reconciler, activation, komposer)
  - E2E tests with kind (website example, activation, dependencies)
  - Priority‑based test plan (P0 → P1 → P2 → P3)

### Changed

- **Testing infrastructure** — no functional changes; this is additive only.

### Security

- No security changes in this release.