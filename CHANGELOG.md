# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Multi-version CRD conversion** — full support for declarative up-conversion, down-conversion, defaulting, and structural field mapping across CRD versions.
- **Conversion mapping engine** — deterministic field transformation system supporting renamed fields, added/removed fields, nested object mapping, and type changes.
- **Conversion pipeline integration** — conversions now run automatically during reconciliation with validation, fallback, and clear error reporting.
- **OCI Registry Support** — Komposer can now load Katalogs from OCI registries (e.g., GitHub Container Registry, Docker Hub) using `sources.oci` references. This enables distribution of operator patterns as standard OCI artifacts.
- **Registry Source Resolution** — OCI artifacts are fetched via ORAS library, cached locally, and merged into the runtime configuration.
- **Pattern Packaging Standard** — defined artifact format for Katalogs: a tarball containing `crd.yaml`, `katalog.yaml`, `komposer.yaml`, `cr.yaml`, and `README.md` with a custom media type (`application/vnd.orkestra.katalog.v1.tar+gzip`).
- **Documentation overhaul** — complete rewrite of the multi-version CRD, conversion rules, and versioning strategy sections with improved structure, diagrams, and examples.
- **Registry Documentation** — new sections covering OCI publishing, pattern structure, and consumption in Komposers.
- **Migration patterns** — new guidance for evolving CRDs safely across versions using Orkestra’s declarative conversion model.

### Changed

- **Documentation structure** — reorganized to better separate beginner, intermediate, and advanced workflows; improved cross-linking and conceptual clarity.
- **Versioning model** — clarified mental model for CRD evolution, conversion ordering, and reconciliation behavior across versions.
- **Komposer Resolution** — extended to handle `oci://` scheme and resolve patterns from container registries.

### Fixed

- **Conversion validation errors** — improved detection and reporting of invalid or incomplete conversion rules.
- **Mapping edge cases** — resolved issues with nested object transforms and missing field defaults during conversion.

### Testing

- **Registry source tests** — integration tests for OCI pull and pattern resolution (not yet enabled in CI; manual testing only).

### Security

- No security changes in this release. OCI artifacts are fetched without signature verification in this version; signing planned for a future release.