# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Per‑CRD endpoint toggles** — `endpoints.health`, `endpoints.info`, and `endpoints.validation` can now be enabled or disabled declaratively in the Katalog. Omitted fields use operator defaults. Disabled endpoints are not registered, resulting in a cleaner and more intentional API surface.

### Changed

- **Katalog route registration** — health, info, and validation routes are now conditionally registered based on the CRD’s endpoint configuration. CRDs with disabled endpoints no longer expose those HTTP routes.

### Security
