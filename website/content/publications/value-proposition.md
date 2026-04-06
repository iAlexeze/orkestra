---
title: "Orkestra Value Proposition"
weight: 50
description: "*Orkestra Project — March 2026*"
---

*Orkestra Project — March 2026*

---

## Executive Summary

Kubernetes operator management is a significant and underappreciated cost
centre in platform engineering. The operator-per-CRD pattern — where each
custom resource definition requires a separate binary, a separate deployment,
and a separate operational surface — has produced clusters running twenty
to fifty operator processes consuming gigabytes of memory.

Orkestra eliminates this overhead. A single Orkestra runtime replaces all
operator processes. A twelve-line Katalog replaces weeks of operator
development.

The savings are not incremental. They are structural. Orkestra does not make
operator development faster. It makes operator development unnecessary for
the common case.

---

## 1. The Problem Orkestra Solves

### 1.1 Operator sprawl

**Orkestra replaces N operator processes with one.** A single Orkestra
instance managing fifteen CRDs consumes approximately 50 MB of resident
memory and 0.05 CPU cores [3]. Fifteen separate operators would conservatively
consume 750 MB to 3 GB of memory and proportional CPU.

### 1.2 Operator development cost

**Orkestra reduces per-CRD operator development to the time required to write
a Katalog.** For the common case — create a Deployment and Service for each
CR, drift-correct on change, cascade-delete on removal — the Katalog is
twelve to twenty lines of YAML.

### 1.3 Operational fragmentation

**Orkestra provides a unified operational interface for every managed CRD.**
The health API (`/katalog/{crd}/health`) has one format. The Prometheus
metrics have consistent label conventions across every CRD.

### 1.4 Multi-version CRD complexity

**Orkestra makes CRD versioning a declaration, not an infrastructure project.**
Conversion rules are declared in the Katalog alongside reconcile templates.
The conversion webhook is built into the runtime.

Production results: 62 conversions, zero failures, sub-millisecond average
latency. The time to add a second CRD version dropped from days to minutes.

---

## 2. What Orkestra Provides

### 2.1 The complete operator stack, automatically

Every CRD declared in a Katalog receives a complete, production-grade
operator stack:

| Capability | Traditional operator | Orkestra |
|---|---|---|
| Informer watching GVK | Written or generated | Automatic |
| Per-CRD worker pool | Written | Automatic, configurable |
| Health endpoint | Written or omitted | Automatic, per-CRD |
| Prometheus metrics | Written or omitted | Automatic, consistent |
| Kubernetes events | Written | Automatic |
| Finalizers | Written | Automatic |
| Owner references | Written | Automatic |
| Drift correction | Written | Declared with `reconcile: true` |
| Dependency ordering | Not provided | Declared with `dependsOn` |
| Version conversion | Written + deployed | Declared in Katalog |
| Admission policy | Written + separate webhook server | Declared in Katalog |

---

## 3. The Numbers

### 3.1 Memory consumption

```
process_resident_memory_bytes    49,176,576   (~47 MB)
```

Fifteen separate Go operator processes would conservatively consume
750 MB at 50 MB each. The reduction is 93%.

### 3.2 Development velocity

| Task | Traditional operator | Orkestra |
|---|---|---|
| First working operator | 3-6 weeks | < 1 hour |
| Adding a new resource type | 1-2 days | Minutes |
| Adding CRD version | 3-5 days | Minutes |
| Adding governance policy | 1 week (webhook server) | One validation rule |

---

## 7. Summary

Orkestra solves three problems that no other tool solves together:

**Operator development cost.** The time to build a production-grade operator
drops from weeks to minutes for the common case.

**Operator operational cost.** N operator processes collapse to one runtime.

**API evolution cost.** CRD versioning becomes a declaration rather than an
infrastructure project.

The savings are structural, not incremental. Orkestra does not make existing
approaches faster. It removes the need for those approaches in the common case.

**Your CRD is enough. Orkestra handles the rest.**

---

*Orkestra — Declarative Operators for Kubernetes*
*March 2026 — https://github.com/iAlexeze/orkestra*
