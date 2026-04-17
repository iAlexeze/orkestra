# Changelog

## [Unreleased]

### Added
- Declarative RBAC generator now supports all operator capabilities, including:
  - Patching and updating CRDs
  - Creating and managing admission and conversion webhooks
  - Managing all resources declared in the Katalog (namespaced and cluster‑scoped)
  - Supporting mixed‑scope and multi‑CRD operators
- Bundle generator (`ork generate bundle`) now produces:
  - ServiceAccounts (runtime and control center)
  - ClusterRoles and ClusterRoleBindings derived from the Katalog
  - Katalog ConfigMap
  - Namespace‑aware manifests for GitOps workflows

### Changed
- RBAC generation is now fully declarative and derived exclusively from the Katalog.
- Helm chart has been refactored to remove all static RBAC, ServiceAccounts, and ConfigMaps.
- Chart now deploys only runtime and control center workloads; all identity and permission artifacts must be generated via the Ork CLI.
- Updated values and chart structure to reflect the new model (`runtime.serviceAccount`, `controlCenter.serviceAccount`, `runtime.katalog.existingConfigMap`, etc.).
- README rewritten to document the new workflow, including:
  - RBAC and bundle generation
  - GitOps‑first installation model
  - Removal of auto‑apply RBAC
  - Explicit referencing of generated resources in Helm values

### Security
- Eliminated all static RBAC from the chart to prevent over‑permissioning.
- Removed runtime RBAC mutation paths; Orkestra no longer creates or modifies cluster‑level RBAC.
- RBAC is now explicit, reviewable, and committed to Git before being applied.
- Establishes a clear trust boundary: cluster administrators own RBAC; Orkestra only reconciles declared resources.
- Ensures least‑privilege by construction through Katalog‑derived permissions.
