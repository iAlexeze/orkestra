---
title: "The Missing Package Manager for Kubernetes Operators"
weight: 50
description: "*OrkestraRegistry — Declarative Distribution for Operator Behavior*"
---

*OrkestraRegistry — Declarative Distribution for Operator Behavior*

  *Orkestra Project — March 2026*

---

## Abstract

The Kubernetes operator ecosystem solved automation but never solved distribution.

Operators are shared as binaries — not because binaries are the right abstraction,
but because no alternative existed. The result is an ecosystem where reuse is
expensive, composition is rare, and every operator is an isolated system with its
own runtime, lifecycle, and operational surface.

OrkestraRegistry introduces a different model: operator behavior distributed as
declarative artifacts. Instead of shipping controllers, it ships reconciliation
logic. Instead of deploying processes, it composes patterns into a shared runtime.

This changes the economics of the ecosystem. Operators become lightweight,
versioned dependencies rather than standalone systems. Patterns evolve from
imperative code to declarative specifications. Distribution moves from binaries
to OCI-backed packages.

The Kubernetes ecosystem has package managers for images and charts.
OrkestraRegistry extends that model to operators — making reuse the default,
and composition the norm.

---

## 2. OrkestraRegistry

OrkestraRegistry is the operator ecosystem for Orkestra. It distributes
operator behavior — YAML Katalogs — as OCI artifacts. The Orkestra runtime
interprets these Katalogs. Multiple patterns compose into a single process.

### 2.1 What a pattern is

A pattern is a directory with five files:

```
postgres/v14/
  crd.yaml          the CRD definition to install
  katalog.yaml      operator behavior — reconcile templates, conversion rules
  komposer.yaml     example showing how to import and override
  cr.yaml           example custom resource
  README.md         field documentation and recommended overrides
```

### 2.2 Distribution as OCI artifacts

Patterns are published as OCI artifacts — the same distribution format as
container images. This is a deliberate choice with consequences for the
entire ecosystem.

OCI registries are already operated by every cloud provider, every enterprise
IT organization, and most development teams. The tooling — authentication,
access control, replication, scanning, retention policies — is mature.

### 2.3 Consuming patterns

A Komposer references OCI patterns in its `sources.oci` block:

```yaml
sources:
  registry:
    - url: ghcr.io/orkestra-sh/registry/postgres:v14
      oci: true
spec:
  crds:
    # Production overrides — win on name conflict with any source
    - name: postgres
      workers: 4
      resync: 30s
```

---

## 8. Conclusion

The operator ecosystem will not remain an ecosystem of binaries indefinitely.
The operational cost is too high. The barrier to contribution is too steep.
The reuse rate is too low.

OrkestraRegistry demonstrates the alternative: operator behavior distributed
as declarations, composed at runtime, overridden without forking, upgraded
through declarative conversion rules.

The infrastructure is ready. The ecosystem follows the patterns.

---

*Orkestra — Declarative Operators for Kubernetes*
*March 2026 — https://github.com/iAlexeze/orkestra*
