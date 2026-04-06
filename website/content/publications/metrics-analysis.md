---
title: "Orkestra Metrics Analysis: Control Center Deep Dive"
weight: 50
description: "---"
---

### Managing 170+ Resources Across 5 CRDs with Zero Code — Observable by Default

---

## Executive Summary

This document analyzes the Prometheus metrics from a running Orkestra instance managing **5 CRDs** (3 built-in Kubernetes resources + 2 custom resources) with a total of **170+ live resources**. The metrics demonstrate Orkestra's ability to handle production-scale workloads while providing **deep observability** through the Control Center — no custom instrumentation required.

**Key Findings:**

- **170+ resources managed** across Pods (69), Secrets (70), Deployments (30), Websites (3), and OrkApps (1)
- **4,200+ reconciliations** processed with 99.6% success rate (only 15 errors out of 4,215 total reconciliations)
- **Worker pool visualization** shows 100% utilization across all 14 workers
- **Queue depth remains near zero** — no backpressure despite high throughput
- **Consistent reconciliation latency** averaging <5ms across all CRDs
- **Efficient memory usage** at 97MB RSS for a process managing 170+ resources
- **No performance degradation** between built-in resources (Pods, Secrets, Deployments) and custom CRDs

---

## The Control Center: Observability by Default

Unlike traditional operators where you must manually add Prometheus metrics and build dashboards, **Orkestra exposes everything automatically**. The Control Center provides:

- **Worker pool visualization** — See every worker's state (idle/processing/stopped) in real time
- **Queue depth monitoring** — Track backpressure before it becomes a problem
- **Reconciliation latency histograms** — Understand performance without custom instrumentation
- **Error rate tracking** — Per-CRD error visibility
- **RBAC rule viewer** — See exactly what permissions each CRD requires
- **Dependency health** — Understand cascading failures instantly

This document analyzes the raw Prometheus metrics that power these views.

---

## 1. Environment Overview

| Metric | Value |
|--------|-------|
| **CRDs Managed** | 5 |
| **Built-in Resources** | Pods, Secrets, Deployments |
| **Custom CRDs** | Website (demo.orkestra.io), OrkApp (orkestra.konduktor.io) |
| **Total Resources** | 173 |
| **Total Reconciliations** | 4,215 |
| **Workers per CRD** | 2-3 |
| **Total Worker Pool** | 14 workers |
| **Memory Footprint** | 97.9 MB RSS |
| **CPU Time** | 18.15 seconds total |
| **Goroutines** | 86 |
| **Uptime** | ~6 minutes (from metrics) |

---

## 4. Reconciliation Performance

### 4.1 Total Reconciliations by CRD

| CRD | Success | Errors | Total | Success Rate |
|-----|---------|--------|-------|--------------|
| Secret | 3,230 | 0 | 3,230 | 100% |
| Pod | 2,043 | 15 | 2,058 | 99.3% |
| Deployment | 930 | 0 | 930 | 100% |
| Website | 84 | 0 | 84 | 100% |
| OrkApp | 29 | 0 | 29 | 100% |
| **TOTAL** | **6,316** | **15** | **6,331** | **99.8%** |

### 4.2 Reconciliation Latency (Histogram)

| CRD | P50 | P95 | P99 | Mean | Count |
|-----|-----|-----|-----|------|-------|
| Secret | <5ms | <5ms | <5ms | 0.8ms | 3,230 |
| Pod | <5ms | <5ms | <5ms | 0.17ms | 2,058 |
| Deployment | <5ms | <5ms | <5ms | 0.20ms | 930 |
| Website | <5ms | <5ms | <5ms | 1.38ms | 84 |
| OrkApp | <5ms | <5ms | <5ms | 3.99ms | 29 |

**Key Finding:** All reconciliations complete in under 5ms.

---

## 11. Conclusions

### 11.6 The Zero-Programming Language Promise is Fulfilled

All 5 CRDs — including the built-in Pod, Secret, and Deployment watchers — were defined entirely in YAML. The Control Center provides full observability into all of them without writing a single line of Go.

---

**Orkestra v1.0 — Declarative Operators for Kubernetes**
*Metrics captured: April 4, 2026*

- **Next:** [RoadMap](../roadmap.md)
