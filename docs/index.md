# Orkestra Documentation

Welcome to the Orkestra documentation — the complete reference for the world's first **zero‑code Kubernetes operator runtime**.

Orkestra turns CRDs into operators. This documentation explains how.

---

## What Is Orkestra?

Orkestra is a declarative, zero‑code Kubernetes operator runtime. You declare what you want in a YAML file — a **Katalog** — and Orkestra manages the full lifecycle of your CRDs: create, reconcile, drift‑correct, delete.

No Go. No code generation. No controller boilerplate.

---

## Why Orkestra?

CRDs were meant to make Kubernetes extensible. But to turn a CRD into an operator, you still write Go code — controllers, informers, reconcilers, finalizers, events, metrics. Every operator repeats the same patterns. Every team rebuilds the same infrastructure.

Orkestra removes that barrier. You write a Katalog. Orkestra builds the operator.

**Your CRD is enough.**

---

## How It Works

Orkestra follows the GitOps pattern of using YAML as the source of truth for operator behavior.

1. **You declare a Katalog** — a YAML file that describes your CRDs and the resources they should create (Deployments, Services, Secrets, ConfigMaps, Jobs, CronJobs).

2. **You apply your CRD** — Orkestra detects the CRD, creates informers, starts workers, and begins watching for Custom Resources.

3. **You create a Custom Resource** — Orkestra reconciles it, creates the declared resources, adds finalizers, and enables drift correction.

4. **You update or delete the CR** — Orkestra reconciles changes, corrects drift, or cleans up child resources.

For a quick overview, see the [Getting Started](./guides/getting-started.md) guide.

---

## Architecture

Orkestra is implemented as a Kubernetes runtime that watches your CRDs and reconciles according to your Katalog. It runs as a single binary — locally or in the cluster — and manages all your CRDs in one process.

- **One runtime, any number of CRDs** — each CRD gets its own informer, worker pool, and queue
- **Dependency‑aware startup** — declare `dependsOn`, Orkestra starts CRDs in topological order
- **Built‑in observability** — health endpoints, Prometheus metrics, `ork status` CLI
- **Per‑CRD tuning** — configure workers, resync intervals, and queue depth per CRD
- **Drift correction** — resources with `reconcile: true` are automatically corrected on every reconcile
- **No programming language required** — just YAML

For additional details, see the [Architecture Overview](./architecture/overview.md).

---

## Quick Start

```bash
# Install Orkestra
brew install iAlexeze/tap/ork
# or
curl -sSL https://raw.githubusercontent.com/konduktor-io/orkestra/main/install.sh | bash
```

For a step‑by‑step walkthrough, see the [Getting Started Guide](./guides/getting-started.md).

---

## Features

- **Zero‑code operators** — No Go, no Python, no controller boilerplate. Just YAML.
- **Any CRD, any resource** — Works with custom CRDs and built‑in Kubernetes resources (Pods, Deployments, Secrets)
- **Declarative templates** — Use Go templates to reference CR spec fields (`{{ .spec.image }}`)
- **Conditional provisioning** — Create resources only when conditions are met (`when: `)
- **Dependency ordering** — Declare `dependsOn`, Orkestra starts CRDs in the right order
- **Drift correction** — Resources with `reconcile: true` are automatically corrected on every reconcile
- **Per‑CRD workers** — Each CRD gets its own worker pool, queue, and resync interval
- **Built‑in observability** — Health and info endpoints (`/katalog/{crd}/health` and `/katalog/{crd}`), Prometheus metrics(`/metrics`), `ork status`
- **Komposer composition** — Compose Katalogs from files, Helm charts, and remote URLs
- **Multi‑source authentication** — Support for bearer tokens, GitHub tokens, and basic auth
- **Leader election** — High availability with warm caches on all replicas
- **Graceful shutdown** — Drains workers, stops informers, releases leader lease

---

## Development Status

Orkestra is being actively developed. The core runtime is stable and ready for testing. New features are being added regularly.

- **Current version:** v0.1.0 (alpha)
- **Roadmap:** See [Roadmap](./publications/roadmap.md)
- **Releases:** Available on [GitHub](https://github.com/orkestra-sh/orkestra/releases)

---

## Adoption

Orkestra is designed for:

- **Platform engineers** — build internal developer platforms without writing operators
- **SREs** — manage infrastructure with declarative YAML, not Go code
- **Infrastructure teams** — replace dozens of operators with one runtime
- **Application developers** — define their own CRDs without learning Kubernetes controller patterns

Early adopters are using Orkestra to manage:

- Namespace provisioners with quotas and network policies
- Database operators (PostgreSQL, MongoDB) with secrets and backups
- Application operators (Deployment + Service + ConfigMap) with conditional public exposure
- Built‑in resource governance (Pod, Deployment, Secret) with health monitoring

---

## Documentation

| Section | Description |
|---------|-------------|
| [Guides](./guides/getting-started.md) | Step‑by‑step instructions for building operators |
| [Concepts](./concepts/katalog.md) | Core ideas that define how Orkestra works |
| [Reference](./reference/katalog-schema.md) | Detailed documentation for every part of Orkestra |
| [Architecture](./architecture/overview.md) | How Orkestra works under the hood |
| [OrkestraRegistry](./orkestra-registry/orkestra-registry-vision.md) | The operator standard library |
| [Publications](./publications/why-orkestra.md) | High‑level papers and conceptual documents |

---

## Community

- [GitHub Issues](https://github.com/konduktor-io/orkestra/issues) — report bugs, request features
- [Discussions](https://github.com/konduktor-io/orkestra/discussions) — ask questions, share ideas
<!-- - Kubernetes Slack — `#orkestra` _(planned)_ -->

---

**Built with ❤️ for the Kubernetes ecosystem.** 🎼