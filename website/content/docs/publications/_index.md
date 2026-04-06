---
title: "Publications"
weight: 7
---

# Publications

This section contains high‑level papers, philosophical foundations, and in‑depth analyses of Orkestra’s architecture and impact.

---

## Philosophy & Vision

| Document | Description |
|----------|-------------|
| [Your CRD Is Enough](../blog/your-crd-is-enough.md) | The foundational essay. Argues that operators should be declarative and that the CRD itself contains everything needed for reconciliation. |
| [Why Orkestra](../blog/why-orkestra.md) | The case for declarative operators. Explains the paradox of writing imperative code to extend a declarative platform. |
| [Trust & Failure Model](./trust-and-failure-model.md) | How Orkestra achieves resilience through per‑CRD isolation, panic recovery, and idempotent operations. |

---

## Architecture & Research

| Document | Description |
|----------|-------------|
| [One Runtime, Many CRDs](./one-runtime-many-crds.md) | A research‑style paper arguing against the one‑operator‑per‑CRD orthodoxy. Demonstrates how per‑CRD isolation enables consolidation without loss of separation of concerns. |
| [Declarative Version Conversion](./declarative-conversion.md) | A technical paper on Orkestra’s built‑in conversion webhook and declarative version rules. Compares to the standard Kubebuilder webhook approach. |
| [The Orkestra Registry](./orkestra-registry.md) | Describes the registry as a library of declarative operator patterns, distributed as OCI artifacts. Covers the three‑layer architecture, promotion path, and OCI publishing. |

---

## Analysis

| Document | Description |
|----------|-------------|
| [Metrics Analysis](./metrics-analysis.md) | Performance analysis of an Orkestra instance managing 170+ resources across 5 CRDs. Includes memory, CPU, goroutine counts, and reconcile latency. |

---

## White Papers

| Document | Description |
|----------|-------------|
| [Declarative Operators Whitepaper](./declarative-operators-whitepaper.md) | A comprehensive white paper covering the history of operators, the problems with existing frameworks, and Orkestra’s declarative alternative. |

---

**These publications are written for a technical audience interested in the philosophy, architecture, and performance of Orkestra. For practical guides, see the [Guides](../getting-started/index.md) section.** 🎼
