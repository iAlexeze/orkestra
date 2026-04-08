---
title: "Index"
weight: 30
---

# Guides

This section contains task‑oriented documentation for using and extending Orkestra.

---

## User Guide

The User Guide is for platform engineers, SREs, and application teams who want to run operators with Orkestra.

| Topic | Description |
|-------|-------------|
| [How Reconciliation Works](./user-guide/how-reconciliation-works.md) | What happens when you create, update, or delete a CR. |
| [Multi‑Version CRD Explained](./user-guide/multi-version-crd-explained.md) | How Orkestra manages multiple API versions declaratively. |
| [Extending Orkestra](./user-guide/extending-orkestra.md) | Adding hooks, custom reconcilers, and new resource types. |
| [Production Deployment](./user-guide/deployment.md/#helm-deployment) | High availability, TLS, RBAC, and resource tuning. |
| [Generate a Certificate with Cert‑Manager](./user-guide/certificate-with-cert-manager.md) | Production‑grade TLS for the conversion webhook. |
| [Generate an OpenSSL Self‑Signed Certificate](./user-guide/certificate-with-openssl.md) | Development‑only certificates. |
| [Choosing Katalog vs Komposer](./user-guide/choosing-katalog-vs-komposer.md) | When to use each, and how they work together. |
| [Use Cases](../use-cases/index.md) | Real‑world examples of Orkestra in production. |
| [Best Practices](./user-guide/best-practices.md) | Recommended patterns for CRDs, Katalogs, and dependencies. |

---

## Developer Guide

The Developer Guide is for contributors and advanced users who want to build, test, and extend Orkestra itself.

| Topic | Description |
|-------|-------------|
| [Development Environment](./developer-guide/development-environment.md) | Setting up a local development cluster, running tests, and debugging. |
| [CLI Reference](../reference/cli/index.md) | Complete reference for the `ork` command and its subcommands. |

---

**Start with the [Getting Started](../getting-started/index.md) guide if you are new to Orkestra. Use the User Guide for task‑focused instructions, and the Developer Guide for contributing.** 🎼