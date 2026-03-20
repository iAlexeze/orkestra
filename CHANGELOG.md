# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added

- **Helm chart** (`charts/orkestra/`) — production-ready chart with
  Deployment, Service, Ingress, HPA, PDB, NetworkPolicy, ClusterRole,
  ClusterRoleBinding, ServiceAccount, and Katalog ConfigMap. Published
  automatically to `https://ialexeze.github.io/orkestra` via
  `chart-releaser-action` on every full release
- **Dockerfile** — two-stage build using `golang:1.22-alpine` builder
  and `gcr.io/distroless/static-debian12:nonroot` final image. Fully
  static binary (`CGO_ENABLED=0`), `readOnlyRootFilesystem: true`,
  runs as uid 65532. Multi-platform: `linux/amd64` and `linux/arm64`
- **Container image pipeline** — `release.yml` now builds and pushes to
  GitHub Container Registry (`ghcr.io/iAlexeze/orkestra`) on every tagged
  release. Tags pushed: exact version (`v1.0.0`), minor alias (`v1.0`),
  major alias (`v1`), and `latest` on full releases only. Build cache
  via GitHub Actions cache for faster subsequent builds
- **Authenticated remote sources** — `LoadFileWithAuth` in `pkg/utils`
  accepts optional `*FileAuth` credentials. Three auth types supported:
  `bearer` (generic token), `github` (GitHub PAT for private repos),
  `basic` (username + password for Artifactory and similar). All
  credentials resolved from environment variables at startup — never
  literal values in YAML
- **`FileSource` type** in `pkg/types` — replaces `[]string` for
  `KatalogSources.Files`. Supports both simple string form
  (`- ./path/to/katalog.yaml`) and authenticated struct form
  (`- url: ... auth: ...`) via `UnmarshalYAML`. Backward compatible —
  existing Komposers require no changes
- **`docs/deployment.md`** — complete deployment guide covering: quick
  local test, Helm install, Katalog management (inline / external
  ConfigMap / remote URL), multi-CRD Komposer patterns, remote sources,
  authenticated sources, production checklist, HA configuration, GitOps
  with ArgoCD and Flux, multi-cluster patterns, and upgrade procedures
- **`docs/komposer.md`** updated — authenticated sources section added
  with bearer, GitHub, and basic auth examples; "Mixing auth types"
  subsection showing all three in one Komposer; "Injecting credentials
  in Kubernetes" with Secret and `extraEnvFrom` pattern; new enterprise
  example with four source types and mixed auth; source table updated
  with auth column

### Changed

- `release.yml` — image build job added running in parallel with the
  binary build job; Helm chart release job added after GitHub release;
  final summary updated to show image digest, tags, Helm, and Homebrew
  status in one table. Docker Hub removed — GHCR only for v1.0
- `LoadFile` in `pkg/utils` — now calls `LoadFileWithAuth(path, nil)`
  internally, preserving full backward compatibility. HTTP client now
  shared with 30-second timeout — previously had no timeout, which could
  hang Orkestra startup indefinitely on slow or unresponsive sources.
  Distinct error messages for 401, 403, and 404 responses instead of
  generic "unexpected status"
- `KatalogSources.Files` type changed from `[]string` to `[]FileSource`
  — transparent to existing YAML via `UnmarshalYAML` on `FileSource`

### Security

- Container image runs as non-root (`runAsUser: 65532`) with
  `readOnlyRootFilesystem: true` and all Linux capabilities dropped
- Helm chart applies `seccompProfile: RuntimeDefault` and
  `allowPrivilegeEscalation: false` by default
- Auth credentials for remote sources are always read from environment
  variables — never accepted as literal values in YAML declarations
- Helm chart PodDisruptionBudget enabled by default (`minAvailable: 1`)
  to prevent accidental complete outage during node maintenance