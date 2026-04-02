# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `ork kompose` command for resolving Komposer files into merged Katalogs.
- `ork diff` command for colorized unified diffs between arbitrary files.

### Changed
- Registry pulls now use native ORAS-Go instead of shelling out to `oras`.
- GHCR authentication flow fixed (GET token exchange).
- RBAC generation is now deterministic across Komposer and Katalog workflows.

### Improved
- Generated Katalogs are pruned of empty/null fields for cleaner output.
- CLI UX consistency and error messages.
