# Overview

The Orkestra Developer Guide provides technical documentation for contributors and platform engineers who want to understand how Orkestra works internally or extend its capabilities. This guide explains how to set up a development environment, build and run the runtime, write typed CRDs and hooks, and contribute new features to the project.

Orkestra is designed to be approachable for both Kubernetes operators and Go developers. Whether you are adding a new CRD, implementing a custom reconciler, or contributing to the core runtime, this guide provides the structure you need.

---

## Audience

This guide is intended for:

- Contributors to the Orkestra runtime  
- Engineers extending Orkestra with new capabilities  
- Platform teams embedding Orkestra into internal tooling  
- Developers writing typed CRDs, hooks, or custom reconcilers  
- Anyone who needs to understand Orkestra’s internal architecture  

If you only want to use Orkestra (not modify it), refer to the User Guide instead.

---

## What This Guide Covers

### Development Environment
How to [set up a local environment](./development-environment.md) for building and testing Orkestra, including required tools, recommended workflows, and how to run the runtime against a local Kubernetes cluster.

### Building Orkestra
Instructions for building the Orkestra runtime and CLI from source, generating runtime registry, and understanding the build layout.

### Testing
How to run unit tests, integration tests, and end‑to‑end tests. Includes guidance on writing new tests and structuring test suites.

### Extending the Runtime
Documentation for contributors who want to add new features to Orkestra itself, such as new registry backends, template engines, metrics, or reconciliation behaviors.

### Architecture
A high‑level overview of Orkestra’s internal architecture, including informers, queues, workers, reconciliation flow, conversion webhooks, and the katalog/registry system.

### Contribution Guidelines
How to submit pull requests, coding standards, commit conventions, and expectations for contributors.

---

## Source Code Layout

The Orkestra repository is organized into several key directories:

```
cmd/                    # CLI and runtime entrypoints
pkg/
  runtime/              # Core runtime logic
  registry/             # Registry loading and katalog resolution
  komposer/             # Komposer engine
  reconciler/           # Generic reconciler and hooks
  api/                  # Typed CRD definitions (optional)
  metrics/              # Prometheus metrics
  health/               # Health endpoints
scripts/                # Developer scripts
tests/                  # Unit, integration, and E2E tests
```

This structure keeps the runtime, CLI, and extension points cleanly separated.

---

## Development Workflow Summary

A typical development workflow looks like this:

1. Set up a local Kubernetes cluster (kind, k3d, microk8s).  
2. Build the Orkestra runtime and CLI.  
3. Create or modify katalogs, CRDs, or hooks.  
4. Run `ork generate registry` if using typed CRDs or hooks.  
5. Start the runtime locally and connect it to your cluster.  
6. Apply CRDs and CRs to test behavior.  
7. Run unit, integration, and E2E tests.  
8. Submit a pull request.  

---

## Next Steps

Continue to the next section:

**Development Environment**  
How to set up a complete Orkestra development environment, including required tools, cluster setup, build instructions, and debugging workflow.
