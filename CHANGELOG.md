# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Multi-version CRD conversion** — full support for declarative up-conversion, down-conversion, defaulting, and structural field mapping across CRD versions.
- **Conversion mapping engine** — deterministic field transformation system supporting renamed fields, added/removed fields, nested object mapping, and type changes.
- **Conversion pipeline integration** — conversions now run automatically during reconciliation with validation, fallback, and clear error reporting.
- **Documentation overhaul** — complete rewrite of the multi-version CRD, conversion rules, and versioning strategy sections with improved structure, diagrams, and examples.
- **Migration patterns** — new guidance for evolving CRDs safely across versions using Orkestra’s declarative conversion model.

### Changed

- **Documentation structure** — reorganized to better separate beginner, intermediate, and advanced workflows; improved cross-linking and conceptual clarity.
- **Versioning model** — clarified mental model for CRD evolution, conversion ordering, and reconciliation behavior across versions.

### Fixed

- **Conversion validation errors** — improved detection and reporting of invalid or incomplete conversion rules.
- **Mapping edge cases** — resolved issues with nested object transforms and missing field defaults during conversion.

### Security

- No security changes in this release.
