# Changelog

All notable changes to Orkestra are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **RBAC Generator (`ork generate rbac`)** — produces a complete, ready‑to‑apply
  ServiceAccount, ClusterRole, and ClusterRoleBinding with least‑privilege rules
  derived directly from the Katalog.
- **Table‑driven resource detection** — unified engine for determining which
  Kubernetes resource types a CRD uses across OnCreate/OnReconcile/OnDelete.
- **RBAC rule table** — declarative mapping of Kubernetes API groups and
  resources, enabling automatic RBAC generation for all supported resource types.
- **Placeholders for future resource types** — structured extension points for
  HPAs, PDBs, NetworkPolicies, PodTemplates, storage classes, and more.

### Changed

- **RBAC generation architecture** — replaced per‑resource `HasX()` helpers with
  a single table‑driven detection model, eliminating duplication and ensuring
  consistency across all resource types.
- **Documentation** — updated security and RBAC sections to reflect Orkestra’s
  security‑first, least‑privilege design and the new RBAC generator workflow.

### Security

- **Least‑privilege by design** — Orkestra now generates RBAC rules strictly
  based on declared CRDs and resource templates, eliminating wildcard or
  over‑permissive roles commonly found in Kubernetes operators.
