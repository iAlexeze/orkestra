---
title: "Operator Authoring vs Infrastructure Control Plane"
weight: 50
description: "Orkestra and Crossplane are often mentioned in the same conversation. Both"
---

Orkestra and Crossplane are often mentioned in the same conversation. Both
are Kubernetes-native. Both are declarative. Both deal with CRDs. Engineers
evaluating one will encounter the other — and with Crossplane v2 explicitly
expanding to cover applications, the question of how they differ deserves
a precise and honest answer.

They are solving fundamentally different problems from fundamentally different
positions. Understanding the distinction is not about competition. It is about
choosing the right tool with clarity.

---

## What Crossplane is

Crossplane is a control plane composition framework. It provides a model for
assembling resources — cloud infrastructure, Kubernetes objects, third-party
services — into higher-level abstractions that platform teams expose to
application developers.

---

## What Orkestra is

Orkestra is an operator authoring runtime. It provides a model for building
Kubernetes operators for your domain CRDs without writing Go code.

You write a Katalog that declares what your CRD should do. When a user creates
a CR, Orkestra's reconcile loop runs — creating child resources, correcting
drift, managing finalizers, writing status, emitting events.

---

## The question Orkestra asks

Orkestra asks a different question entirely.

Not: *how do we make Crossplane's model work for applications?*

But: *what does a user's CR look like if the framework is completely invisible?*

```yaml
apiVersion: demo.orkestra.io/v1alpha1
kind: Website
metadata:
  name: my-site
spec:
  image: nginx:1.25
  replicas: 2
  port: 8080
```

No `spec.crossplane` or `spec.orkestra` fields. No compositionRef. No resourceRefs. No framework
machinery of any kind. The CR is pure domain.

This is what *your CRD is enough* means — not as a slogan, but as a technical
guarantee. The user's API surface is defined entirely by your CRD schema.
Orkestra contributes nothing to it.

---

## When to use each

**Use Crossplane when:**

- You are provisioning external infrastructure — cloud databases, storage,
  networks, managed services from cloud providers
- You need the Provider ecosystem (AWS, GCP, Azure, and hundreds of others)
- You are building a platform where engineers self-service infrastructure
  through Kubernetes CRs and those resources live outside the cluster

**Use Orkestra when:**

- You have domain CRDs that should produce Kubernetes child resources
- You want operators without writing Go
- You need admission policy, multi-version conversion, and declarative status
  from a single YAML declaration
- You want governance over existing Kubernetes resources (built-in kind enrichment)
- You want your users' CRs to contain only domain fields — no framework machinery

**Use both when:**

Crossplane provisions the external infrastructure. Orkestra manages the
application-layer operators that consume it. Crossplane creates the RDS
instance. Orkestra manages the `Database` CR that creates the connection Secret,
configures the Deployment, and keeps everything in sync.

---

## Summary

| | Crossplane v2 | Orkestra |
|---|---|---|
| **Primary purpose** | Infrastructure and application control plane composition | Kubernetes operator authoring |
| **User's CR** | Carries framework machinery (`spec.crossplane`) | Pure domain — framework invisible |
| **Logic expression** | Composition Functions (KCL, CEL, Python, Go) | YAML templates, evaluated at runtime |
| **Concept count** | 7+ | 2 |
| **Learning curve** | Days to weeks | Hours |
| **Reconciliation model** | Composition rendering | Continuous drift-correcting loop |
| **Declarative status** | Limited | Three layers including child propagation |
| **Admission policy** | No | Yes — deny/warn, two enforcement points |
| **Multi-version conversion** | No | Yes — declarative paths, in-process |
| **Live operational API** | No | Yes — `/katalog` per CRD |
| **Kubernetes model** | New model on top of Kubernetes | Kubernetes model throughout |
| **Memory (15 CRDs)** | 500 MB+ (core + providers) | ~47 MB |

Orkestra places Kubernetes first, your CRD first, and your users first.
The framework disappears. The operator appears.

That is the difference.
