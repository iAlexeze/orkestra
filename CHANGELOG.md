# Changelog

## [Unreleased]

### Added
- ReplicaSet becomes a first‑class Orkestra workload primitive.
- New `replicaSets:` template block in `operatorBox` (onCreate/onReconcile).
- Full registry implementation: create, update, delete, deleteIfOwned.
- Template resolver: `ResolveReplicaSetTemplate`.
- Reconciler integration: `runReplicaSets`, forEach expansion, conditional cleanup.
- Katalog schema support for ReplicaSetTemplateSource.
- Example pack: multi‑region fan‑out using ReplicaSets.

### Changed
- Workload orchestration no longer requires Deployments for simple pod replication.
- forEach fan‑out now supports ReplicaSet workloads directly.
- Simplified workload lifecycle: CR → ReplicaSet → Pod (Deployment layer removed).

### Removed
- Implicit reliance on Deployment rollout machinery for basic workloads.

### Fixed
- Correct ownerReferences for ReplicaSets enabling full garbage collection.
- Idempotent reconcile behavior for ReplicaSet drift correction.
