# Changelog

Here’s a clean, publication‑grade **CHANGELOG** entry that matches your existing Orkestra editorial tone — concise, factual, and structured for CNCF‑style release notes.

---

## [Unreleased]

### [Added]
- Introduced new resource builders in the Orkestra Registry to expand parity with Kubernetes primitives:
  - **StatefulSet**
  - **PersistentVolume (PV)**
  - **PersistentVolumeClaim (PVC)**
  - **PodDisruptionBudget (PDB)**
  - **HorizontalPodAutoscaler (HPA)**
  - **Ingress**
- Added `common` package for shared builder utilities:
  - `BuildResourceRequirements` — canonical Kubernetes‑native resource requirements converter
  - `ResolveNamespace(owner, spec.Namespace)` — unified namespace resolution logic
  - `parsePort`, `parseBool` — shared parsing helpers for resource builders
- Standardized resource‑builder patterns across Deployment, StatefulSet, and new resource types using the new common utilities.

### [Changed]
- Reduced duplication across all registry builders by consolidating repeated logic into `common/`.
- Updated Deployment and StatefulSet builders to use the shared `BuildResourceRequirements` and `ResolveNamespace` helpers.

### [Technical Notes]
- This work represents the first step toward a fully unified, extensible registry architecture.
- No tests have been added yet; test coverage and validation logic will be introduced in the next sprint.
