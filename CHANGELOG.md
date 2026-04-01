# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Orkestra Dashboard** — a comprehensive web-based UI providing real-time visibility into operator operations
  - **CRD Overview Dashboard**: displays all managed CRDs with key metrics including workers, queue depth, resource count, error rates, and health status
  - **CRD Detail View**: in-depth metrics per CRD including runtime health, reconciliation statistics, version conversion webhook status, admission webhook metrics, and reconciler configuration details
  - **System Metrics Dashboard**: real-time Go runtime metrics including GC pause times, heap memory usage, and goroutine counts with auto-refresh capabilities
  - **Metrics API**: `/dashboard/api/metrics` endpoint exposing structured JSON metrics for programmatic access
  - **Static Asset Embedding**: all dashboard assets embedded directly into the binary using Go's embed directive
  - **Automatic Data Refresh**: dashboard auto-refreshes every 5-10 seconds for real-time monitoring

### Changed

- **Documentation** — added comprehensive dashboard usage guide covering installation, navigation, and metrics interpretation
- **End-to-End Testing** — validated dashboard functionality using all example workloads from beginner to advanced scenarios
- **Examples** — updated example CRDs to demonstrate dashboard visibility features

### Fixed

- **Template Function Support** — added proper template function handling (`add`, `mul`, `div`, `formatTime`, `relativeTime`) for dynamic dashboard calculations
- **Timestamp Formatting** — improved time display with local timezone conversion and relative time formatting for better UX

### Remaining Work

- **Hook Testing** — comprehensive testing of Go hooks and webhook integrations pending
- **Constructor Testing** — validation of custom reconciler constructors in progress