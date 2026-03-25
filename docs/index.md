# Orkestra Documentation

Welcome to the Orkestra documentation — the complete reference for the world's first **zero‑code Kubernetes operator runtime**.

Orkestra turns CRDs into operators. This documentation explains how.

---

<div align="center">

```
   ___       _              _
  / _ \ _  _| |___ _ _  ___| |_ _ _ __ _
 | (_) | || | / -_) ' \/ -_)  _| '_/ _` |
  \___/ \_,_|_\___|_||_\___|\__|_| \__,_|
          O R K E S T R A
```

<strong>CRDs in. Operators out.</strong>

</div>

---

## Quick Start

New to Orkestra? Start here to build your first operator in minutes.

👉 **[Get Started →](./guides/getting-started.md)**

---

## Guides

Step‑by‑step instructions for building and operating with Orkestra.

| Guide | Description |
|-------|-------------|
| [Getting Started](./guides/getting-started.md) | Install Orkestra and run your first operator |
| [Writing Your First Katalog](./guides/writing-your-first-katalog.md) | Declare CRDs and create resources from templates |
| [Writing Hooks](./guides/writing-hooks.md) | Add Go logic when templates aren't enough |
| [Writing Custom Reconcilers](./guides/writing-custom-reconcilers.md) | Take full control of reconciliation |
| [Testing Operators](./guides/testing-operators.md) | Unit, integration, and E2E testing |

---

## Concepts

Core ideas that define how Orkestra works.

| Concept | Description |
|---------|-------------|
| [Katalog](./concepts/katalog.md) | Declare operator behavior |
| [Komposer](./concepts/komposer.md) | Compose Katalogs from multiple sources |
| [Katalog & Komposer Reference](./reference/katalog-komposer-reference.md) | Full schema reference |
| [Templating](./concepts/templating.md) | Template expressions and resolution |
| [Dependency Model](./concepts/dependency-model.md) | How Orkestra manages CRD dependencies |
| [Trust & Failure Model](./core/trust-and-failure-model.md) | Why Orkestra is safe to use |
| [Your CRD is Enough](./publications/your-crd-is-enough.md) | The philosophy |

---

## OrkestraRegistry

The operator standard library — reusable, versioned operator patterns.

| Document | Description |
|----------|-------------|
| [OrkestraRegistry Vision](./orkestra-registry/orkestra-registry-vision.md) | The future of reusable operators |
| [OrkestraRegistry Technical Documentation](./orkestra-registry/orkestra-registry-technical-documentation.md) | How the registry works |

---

## Reference

Detailed documentation for every part of Orkestra.

| Document | Description |
|----------|-------------|
| [CLI Reference](./reference/cli.md) | All `ork` commands and flags |
| [Inspect Live CRD](./reference/inspect-live-crd.md) | Inspect CRDs directly from the terminal |
| [Architecture](./architecture/overview.md) | How Orkestra works under the hood |
| [Architecture Diagrams](./architecture/architecture-diagrams.md) | Visual architecture overview |
| [Komponents](./concepts/komponent.md) | What each part of Orkestra does |
| [Health Subsystem](./concepts/health-subsystem.md) | Health tracking and degradation |
| [Metrics](./reference/metrics.md) | Prometheus metrics reference |

---

## Deployment

How to run Orkestra in different environments.

| Document | Description |
|----------|-------------|
| [Deployment Guide](./guides/deployment.md) | Helm, GitOps, and production setups |
| [Use Cases](./guides/use-cases.md) | Real‑world operator patterns |

---

## Publications

High‑level papers and conceptual documents.

| Document | Description |
|----------|-------------|
| [Why Orkestra](./publications/why-orkestra.md) | The case for declarative operators |
| [Declarative Operators Whitepaper](./publications/declarative-operators-whitepaper.md) | Technical whitepaper |
| [Metrics Analysis](./publications/metrics-analysis.md) | Performance metrics for 170+ CRDs |

---

## Community

- [GitHub Issues](https://github.com/konduktor-io/orkestra/issues)
- [Discussions](https://github.com/konduktor-io/orkestra/discussions)
- Kubernetes Slack — `#orkestra` _(planned)_

---

**Built with ❤️ for the Kubernetes ecosystem.** 🎼