# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### CLI
- **`ork control start`** – Launch the Control Center directly from the `ork` CLI.
- **`ork control version`** – Show Control Center version information.
- Automatic binary discovery in `PATH` and `~/.orkestra/bin`.

#### Control Center
- Full integration with `ork control start` command supporting:
  - Custom port (`--port`, `-p`)
  - Multiple runtime URLs (`--urls`, `-u`)
  - Configurable refresh interval (`--refresh`)
  - Log level control (`--log-level`)

### Changed

#### Installation
- Default install directory changed from `/usr/local/bin` to `$HOME/.orkestra/bin` (user-local, no sudo required).
- Install script now creates directory automatically and prints PATH instructions.
- Makefile `OUTPUT_DIR` now uses `$(HOME)/.orkestra/bin` for consistency.

### Fixed
- Control center now respects all CLI flags when launched via `ork control start`.

### Documentation
- Added comprehensive Control Center documentation covering:
  - Architecture (Control Center vs Control Panels)
  - Health states (Orkestra Health vs Katalog Health)
  - Worker pool visualization
  - Queue pressure monitoring
  - RBAC permissions viewer
  - Dependency health tracking