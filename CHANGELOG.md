# Changelog

---

### Added (feat/deletion-protection-certmanager)

**Deletion protection label propagation**
- Added `orkestra.io/deletion-protection: "true"` to `orkestraResourceLabels` — single source of truth; Deployment, Service, RBAC, ConfigMap, TLS Secret, and webhook configurations all carry it automatically
- Added `pkg/labels` package with `DeletionProtectionLabel` constant and `WithDeletionProtection` helper (immutable — never mutates the input map)
- Added `OrkestraBaseLabels()` package-level function to `pkg/konfig` for callers without a `Konfig` instance (e.g. `configmap_generator.go`)
- Added `orkestra.io/deletion-protection: "true"` to the Helm chart `orkestra.selectorLabels` helper

**Namespace deletion protection**
- `ensureSecurity` now patches the Orkestra namespace with deletion-protection labels at startup so the admission webhook's `ObjectSelector` fires on namespace-delete attempts
- Added RBAC rule (`get` + `patch` on `namespaces`) to `GenerateRBACRules` when deletion protection is enabled

**Certificate manager (`pkg/certmanager`)**
- New `Manager` interface: `EnsureCertificate` and `DeleteCertificateAndSecret`
- `k8sManager` implementation stores the TLS bundle in a Secret labeled with deletion-protection labels; upserts on conflict
- `HealthServer` now accepts a `certManagerIface` (local interface, avoids circular import) and cleans up the TLS Secret on graceful shutdown when deletion protection cleanup is enabled

**`ORKESTRA_NAMESPACE` wiring**
- Helm chart `ORKESTRA_NAMESPACE` env var switched from static `{{ .Release.Namespace }}` to Downward API `fieldRef: metadata.namespace`
- `cmd/cli/generate.go` `DefaultNamespace` constant replaced with `defaultNamespace()` function that reads `$ORKESTRA_NAMESPACE` at call time

**Merger — top-level field inheritance for Komposer**
- Fixed: `ork generate rbac` / `ork generate configmap` against a Komposer now produces the same output as running against the source Katalogs directly
- `loadKomposer` accumulates `security`, `notification`, and `providers` from every source Katalog; the Komposer's own block is merged on top with override semantics
- Added `mergeKatalogSecurity` and `mergeKatalogNotification` helpers to `pkg/merger/helper.go`
- Added `Notification *KatalogNotification` as a top-level field on `KatalogFile` (was missing — only existed on the runtime `Katalog` struct)
- `Merger` gains a `notification` field and `ToNotification()` accessor; `KomposeKatalogFromYaml` wires it into `Katalog.Notification`

**Merger documentation**
- Added `pkg/merger/README.md` — package overview, file table, merge rules, accumulation semantics, usage example
- Added `pkg/merger/docs/` directory with five progressive documents following the reconciler/katalog pattern:
  - `01-architecture.md` — full pipeline diagram and invariants
  - `02-kinds.md` — Katalog vs Komposer rules with YAML examples
  - `03-sources.md` — source types, auth, and how to add a new source type
  - `04-deduplication.md` — two deduplication scopes and error message format
  - `05-top-level-accumulation.md` — explains the accumulation bug, the fix, and per-field merge semantics

---

### Added (feat/e2e-suite-onboarding)

Onboarded the full Orkestra E2E example suite into the repository

Added beginner packs 01, 02, 03, and 03b

Added corresponding GitHub Actions workflows under .github/workflows/

Introduced a versioned, extensible example system for demonstrating Orkestra capabilities

Established a contribution path for community-authored example packs
