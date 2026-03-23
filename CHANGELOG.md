# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Global endpoint toggle** — new `endpoints.enabled` field allows disabling all CRD‑specific HTTP endpoints (`/health`, `/info`, `/validation`) with a single switch. When set to `false`, no routes are registered and the CRD is omitted from the Katalog endpoint list.

- **Per‑CRD endpoint toggles** — `endpoints.health`, `endpoints.info`, and `endpoints.validation` can now be enabled or disabled declaratively in the Katalog. Omitted fields use operator defaults. Disabled endpoints are not registered, resulting in a cleaner and more intentional API surface.

### Changed

- **Endpoint registration logic** — CRDs with `endpoints.enabled: false` now skip all endpoint registration before the health server starts. Individual endpoint flags (`health`, `info`, `validation`) continue to work when `enabled` is omitted or true.

- **Katalog route registration** — health, info, and validation routes are now conditionally registered based on the CRD’s endpoint configuration. CRDs with disabled endpoints no longer expose those HTTP routes.

### Security
