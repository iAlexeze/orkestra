---
title: "Index"
weight: 150
---

# Runtime Manual

This section explains how Orkestra works under the hood — its architecture, core concepts, and internal mechanisms.

---

## Architecture

High‑level views of the Orkestra runtime and its components.

| Document | Description |
|----------|-------------|
| [Overview](./architecture/index.md) | The big picture: how Orkestra fits into Kubernetes. |
| [Runtime Flow](./architecture/runtime-flow.md) | Startup, reconciliation, and shutdown sequences. |
| [CRD Lifecycle](./architecture/crd-lifecycle.md) | From CRD installation to reconciliation. |
| [Design Philosophy](./architecture/design-philosophy.md) | The principles that guide Orkestra’s design. |
| [Architecture Diagrams](./architecture/architecture-diagrams.md) | Visual overview of components and flows. |
| [Full Architecture View](./architecture/full-architecture-view.md) | Complete system view with all components. |

---

## Concepts

The core ideas that define how Orkestra works.

| Document | Description |
|----------|-------------|
| [Katalog](./concepts/katalog.md) | Declare operator behavior. |
| [Komposer](./concepts/komposer.md) | Compose Katalogs from multiple sources. |
| [Registry Sources](./concepts/registry-sources/index.md) | Import patterns from files, Helm, OCI, and more. |
| [Hooks](./concepts/hooks.md) | Go functions for custom logic. |
| [Constructors](./concepts/constructors.md) | Custom reconcilers in Go. |
| [Observability](./concepts/observability.md) | Metrics, health endpoints, and CLI status. |
| [Runtime](./concepts/runtime.md) | How the reconciliation loop works. |
| [Versioning](./concepts/versioning.md) | Declarative multi‑version CRD support. |
| [Conditional Provisioning](./concepts/conditional-provisioning.md) | Create resources only when conditions are met. |
| [Templating](./concepts/templating.md) | Go templates and the resolver. |
| [Dependency Model](./concepts/dependency-model.md) | CRD startup and shutdown ordering. |
| [Health Subsystem](./concepts/health-subsystem.md) | Health tracking and degradation. |
| [Reconciler Model](./concepts/reconciler-model.md) | The full reconciliation pipeline. |

---

## Internals

Deep dives into Orkestra’s implementation for contributors and advanced users can be found [here](../technical-docs/index.md).

---

**Start with the [Architecture Overview](./architecture/index.md) if you are new to the runtime. Use the [Concepts](./concepts/index.md) section to understand individual ideas, and the [Internals](../technical-docs/index.md) section for implementation details.** 🎼