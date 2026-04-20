# Changelog

## [Unreleased]

### Added
- **Windows support**: Native `ork.exe` and `orkcc.exe` binaries with `.zip` packages (in addition to `.tar.gz`)
- **Winget publishing**: Automated manifest generation for Windows Package Manager
- **Reusable workflows**: Split monolithic release pipeline into composable components:
  - `build-matrix.yml` – cross‑platform binary builds
  - `package-examples.yml` – example packs
  - `build-push-images.yml` – container images (GHCR)
  - `release-helm.yml` – Helm chart release
  - `sign-and-release.yml` – GPG signing + GitHub Release
  - `publish-homebrew.yml` – Homebrew formulas
  - `publish-winget.yml` – Winget manifests
  - `release-summary.yml` – final status summary
- **Caching optimisation**: Per‑matrix binary caching to speed up rebuilds
- **GPG signing for ZIP archives**: Windows `.zip` files are now signed alongside tarballs

### Fixed
- Corrected `go build` syntax (missing `-o` flag) in matrix build
- Fixed typo in cache path (`dis/` → `dist/`)
- Removed duplicate Windows assets in release `files:` section
- Reusable summary workflow now accepts job results as inputs

### Changed
- Helm chart publishing now only runs for stable releases (not pre‑releases)
- Windows binaries are packaged as both `.tar.gz` and `.zip` for better user experience
- Homebrew formulas update uses `ORK_PUBLISH_PAT` instead of `HOMEBREW_TAP_TOKEN`