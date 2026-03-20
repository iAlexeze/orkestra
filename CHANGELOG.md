# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Fixed
- `install.sh` — removed duplicate `tmp_dir` assignment that caused
  `unbound variable` error on exit when `set -euo pipefail` was active
- `REPO` variable corrected to `iAlexeze/orkestra` (was `ialexeze/orkestra`)
  to match actual GitHub URL casing

### Added
- GPG release signing — every tarball now ships with a detached `.asc`
  signature and the public key (`orkestra-public-key.asc`) is published
  with every release
- Homebrew tap support — `brew tap iAlexeze/tap && brew install ork`
  now works on macOS and Linux; formula is updated automatically by the
  release workflow on every full release
- `release.yml` — GPG import, sign, self-verify, and public key export
  steps added; Homebrew formula update job added (skipped on pre-releases)
- `HOMEBREW_TAP_TOKEN` secret support in release workflow

### Changed
- `README.md` — Homebrew install added as primary macOS install method;
  GPG verification section added; Komposer section rewritten ("meta-katalog"
  removed); metrics table formatted; documentation table updated
- `ROADMAP.md` — complete rewrite based on actual Orkestra architecture;
  `ork dashboard`, authentication for remote sources, `ork diff`, `ork lint`
  added to Phase 2; "what we are not building" section added
- `docs/architecture.md` — complete rewrite; Go/YAML mode references removed;
  accurate to `konstructOrkestra`, `KonductorElection`, and actual startup
  sequence
- `docs/components.md` — complete rewrite replacing "Komponent Deep Dive";
  accurate to current komponent names, actual code, and real startup order
- `docs/extending.md` — complete rewrite; Go mode removed; three paths
  (declarative templates, Go hooks, custom constructor) documented accurately
- `docs/cli.md` — complete rewrite; removed non-existent commands
  (`ork katalog list`, `ork graph deps`, `ork get crds`); added `ork init`,
  `ork generate runtime` with when-needed table, three complete workflows
- `docs/templating.md` — complete rewrite; documents runtime interpretation,
  not the now-deleted code generation pipeline
- `docs/komposer.md` — replaces `docs/katalog-sources.md`; Katalog vs
  Komposer distinction enforced; Komposer-sourcing-Komposer error documented
- `docs/use-cases.md` — replaces `docs/yaml-use-cases.md`; all hypothetical
  and forward-looking scenarios removed; every use case is achievable today
- `docs/orkestra-registry.md` — complete reference for all eight resource
  types; ready for the OrkestraRegistry to move to its own repository
- `docs/hooks.md` — replaced with deprecation notice pointing to
  `docs/templating.md`
- `publish/why-orkestra.md` — rewritten to earn the conclusion rather than
  state it; "operators become X" bullet list replaced with four real arguments
- `publish/declarative-operators-whitepaper.md` — rewritten as a proper
  technical paper with abstract, problem statement, model description,
  counterarguments, and implications sections

### Documentation added
- `docs/gpg-setup.md` — ten-step guide for GPG key generation, GitHub
  Actions secrets setup, Homebrew tap creation, and end-to-end verification
- `docs/reconciler/reconciler.md` — architecture, flow, and contributing
  patterns for the reconciler package