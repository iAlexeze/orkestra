# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Core Runtime
- **Multi-version conversion** – Declarative YAML-based version conversion between CRD API versions. No webhook server required.
- **Admission webhooks** – Validation and mutation rules declared directly in Katalog YAML. No separate webhook deployment needed.
- **Built-in resource management** – Any Kubernetes built-in (Pod, Service, Deployment, etc.) can be treated as a CRD with validation and mutation rules.
- **OCI registry patterns** – Pull complete operator patterns from GHCR or any OCI registry using ORAS-Go native integration.
- **Git registry patterns** – Pull patterns from GitHub, GitLab, or generic Git repositories.
- **Pattern validation** – Five-file structure validation (crd.yaml, katalog.yaml, komposer.yaml, cr.yaml, README.md) on pull.
- **Declarative dependencies** – `dependsOn` supports both simple list (defaults to `started`) and explicit conditions (`healthy`, `started`, `complete`, `ready`).
- **`ork kompose` command** – Resolve Komposer files into merged Katalogs.
- **`ork diff` command** – Colorized unified diffs between arbitrary files.
- **`ork validate` command** – Validate Katalog and Komposer files without running.
- **`ork generate rbac` command** – Generate ClusterRole RBAC from any Katalog or Komposer.

#### Control Center (New)
- **Multi-instance aggregation** – Monitor multiple Orkestra runtimes from a single web UI.
- **Katalog landing page** – View all Katalogs across all instances with health status and summary statistics.
- **Katalog dashboard** – Detailed view per Katalog with CRD health cards, queue pressure indicators, error rates, and uptime.
- **CRD detail view** – Deep dive into individual CRDs with runtime health, queue visualization, version conversion metrics, and admission stats.
- **Real-time refresh** – Auto-refresh every 10 seconds with configurable interval.
- **Search and filter** – Filter CRDs by name and status (healthy, started, pending, degraded).
- **Standalone binary** – Single binary deployment with embedded assets.
- **Helm chart support** – Deploy as part of the Orkestra Helm chart with `controlCenter.enabled: true`.

#### Helm Chart
- **Separate runtime and control center deployments** – Independent scaling and lifecycle management.
- **Multi-URL support** – Configure multiple Orkestra runtime URLs for the control center.
- **Configurable probes** – Liveness and readiness probes for both components.
- **Network policies** – Optional network policies for runtime and control center.

### Changed

#### Runtime
- Registry pulls now use native ORAS-Go instead of shelling out to `oras`.
- GHCR authentication flow fixed (GET token exchange).
- RBAC generation is now deterministic across Komposer and Katalog workflows.
- Health server now supports both HTTP and HTTPS (for conversion/admission webhooks).
- Improved leader election with configurable lease duration.

#### Control Center
- Renamed from "Dashboard" to "Control Center" – now `/controlcenter` instead of `/dashboard`.
- Route structure: `/controlcenter` (landing), `/controlcenter/katalog/:name` (dashboard), `/controlcenter/katalog/:name/crd/:crd` (detail).
- Backend health checks no longer block UI rendering – cached data is always shown.

### Improved

- Generated Katalogs are pruned of empty/null fields for cleaner output (up to 50% reduction).
- CLI UX consistency and error messages.
- Template rendering with buffer to prevent partial responses on errors.
- Graceful shutdown handling for both runtime and control center.
- Logging with configurable log levels (debug, info, warn, error).

### Fixed

- `superfluous response.WriteHeader` warnings in control center handlers.
- Missing `truncate` template function causing parse errors.
- CRD detail links now correctly route through Katalog context.
- Logo asset serving path corrected to `/controlcenter/assets/static/logo.png`.

### Deprecated

- Standalone dashboard binary (`ork-cc` remains, but integrated `ork control` coming soon).

### Removed

- Shell dependency for ORAS pulls – now pure Go.
- Old `/dashboard` route (redirects to `/controlcenter` for backward compatibility).

### Security

- Control center runs as non-root user (UID 1001).
- Read-only root filesystem for control center container.
- Dropped all unnecessary Linux capabilities.

### Testing

- Unit tests for template functions and URL parsing.
- Integration tests for multi-instance aggregation.
- Manual testing with multiple Orkestra runtimes:
  - Two runtime instances on different ports
  - Cross-instance Katalog navigation
  - CRD detail fetching from correct instance
  - Graceful degradation when instances are down

## [Next Steps]

### Short-term (v1.1.0)
- [ ] **`ork control start` command** – Launch control center from the main Orkestra CLI:
  ```bash
  ork control start --port 8090 --url http://localhost:8080,http://localhost:8082
  ```
- [ ] **End-to-end testing suite** – Automated tests for:
  - Multi-instance discovery and aggregation
  - Katalog navigation across instances
  - CRD detail fetching
  - Health check behavior
  - Helm chart upgrades
- [ ] **Deactivation path** – Gracefully handle CRDs that are removed from the runtime after startup:
  - Mark missing CRDs as "removed" in the UI
  - Auto-cleanup after configurable TTL
  - Log warnings without crashing

### Medium-term (v1.2.0)
- [ ] **User authentication** – Basic auth, OAuth2, SSO integration for control center.
- [ ] **Time-series graphs** – Historical metrics visualization using Prometheus data.
- [ ] **Alerting** – Email/Slack/PagerDuty integration for degraded CRDs.
- [ ] **Dark mode** – Theme support for the control center.
- [ ] **Export functionality** – CSV/JSON export of metrics and health data.

### Long-term (v2.0.0)
- [ ] **Multi-tenancy** – Team-based access control and isolated views.
- [ ] **Custom dashboards** – Save and share custom views.
- [ ] **Audit logging** – Track who viewed what and when.
- [ ] **WebSocket updates** – Real-time updates without page refresh.
