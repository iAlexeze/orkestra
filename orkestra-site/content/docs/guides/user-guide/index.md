---
title: "Index"
weight: 40
---

# Overview

The Orkestra User Guide helps you learn how to use Orkestra effectively — from installation and configuration to extending the platform with your own CRDs and Katalogs. Whether you are just getting started or building advanced operator workflows, this guide provides practical, task-focused documentation.

---

## What You Will Find in the User Guide

The User Guide is organized into clear, focused sections.

---

### Getting Started

A beginner-friendly introduction to installing Orkestra, deploying your first CRD, understanding reconciliation, and exploring the Katalog and health endpoints. Start here if you are new to Orkestra or Kubernetes operators.

---

### Extending Orkestra

Learn how to extend Orkestra with your own CRDs, Katalogs, templates, conversion rules, and dependencies. This section shows how to build declarative operators without writing Go code.

---

### Production Deployment

Guidance for running Orkestra in production environments, including high availability, TLS and certificate management, RBAC and security, resource tuning, and observability.

---

### Certificate Management

Orkestra requires TLS for its conversion webhooks. This section covers generating certificates with cert-manager for production and OpenSSL self-signed certificates for development.

---

### Choosing Between Katalog and Komposer

Orkestra provides two complementary concepts: Katalog defines how a CRD behaves, while Komposer loads Katalogs from registries. This section explains when to use each, how they work together, and how to structure environment overrides.

---

### Use Cases

Real-world examples of how teams use Orkestra to manage application deployments, build internal platforms, create reusable operator behaviors, manage multi-version APIs, and automate infrastructure components.

---

### Best Practices

Recommended patterns for designing CRDs, structuring Katalogs, writing templates, managing dependencies, versioning APIs, and operating Orkestra in production.

---

### Troubleshooting

Common issues and their solutions, including webhook TLS errors, conversion failures, reconciliation loops, worker inactivity, missing Katalogs, and registry authentication.

---

### Reference Documentation

Deep-dive reference material including API specifications, Katalog schema, Komposer schema, metrics reference, health endpoints, and architecture overview.

---

## Who This Guide Is For

The User Guide is designed for platform engineers, SREs, DevOps teams, Kubernetes operators, application developers, and anyone building declarative automation on Kubernetes.
