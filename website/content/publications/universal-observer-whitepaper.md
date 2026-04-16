---
title: "Orkestra: The Universal Observer That Belongs in Kubernetes Core"
weight: 50
description: "*Orkestra Project — March 2026*"
---

*Orkestra Project — March 2026*

---

## Abstract

Kubernetes extensibility is built on a structural absence. The API server
stores and serves custom resources. It knows their schema. It emits watch
events when they change. What it does not do is watch what happens after those
events are emitted.

This absence has compelled the invention of adjacent systems that compensate
for it. Admission webhooks exist because the API server cannot ask its own
operators whether a CR is valid. Policy engines exist because operators have
no shared policy model. Observability tooling exists because every operator
invents its own metrics and health conventions.

We argue that all of these systems are compensations for the same architectural
gap. We describe Orkestra — a runtime that acts as the permanent observer of
its managed surface — and show that when such an observer exists, the
external compensations become unnecessary. We then make the case that this
observer belongs inside Kubernetes itself.

---

## 1. The Structural Absence

### 1.1 What Kubernetes provides

Kubernetes makes a specific set of guarantees about custom resources. When a
CRD is installed, Kubernetes will store objects of that type, validate them
against the declared schema, emit watch events when they change, and serve
them over the REST API.

What it does not do is observe the downstream consequences of those changes.

### 1.2 The compensations

**Admission webhooks** compensate for the API server's inability to ask
operators whether a CR is valid.

**Policy engines** — OPA, Kyverno, Gatekeeper — compensate for the fact that
operators have no shared policy model.

**Observability tooling** compensates for the fact that every operator invents
its own metrics, health conventions, and operational interface.

### 1.3 The pattern in the compensations

The absence is: **no permanent observer watching the managed surface.**

If such an observer existed, the coordination problem that webhooks solve
would not require external delegation. The policy problem that policy engines
solve would not require a separate system. The observability problem would not
require per-operator tooling.

Orkestra is that observer.

---

## 2. The Observer Model

### 2.1 What an observer provides

**Cross-surface visibility.** An observer that watches ten CRDs sees all ten
simultaneously. It can enforce policy uniformly. It can order their lifecycle.

**In-process authority.** An observer that is trusted by the system can be
called synchronously during admission. There is no network hop. There is no
TLS.

**Standard interface by construction.** An observer that manages every CRD
through the same runtime provides the same operational interface for every CRD.

---

## 3. The Case for Core Integration

### 3.1 Precedent: kube-controller-manager

Kubernetes has always shipped with a runtime that watches many resource types
through a shared process. `kube-controller-manager` runs the Deployment
controller, the ReplicaSet controller, the StatefulSet controller, the Job
controller, and dozens of others.

This is exactly the Orkestra model applied to core Kubernetes resources.

### 3.2 The path

The path from user-space Orkestra to in-core Orkestra is multi-year and
community-governed:

1. Production adoption (Year 1)
2. CNCF Sandbox (Year 2)
3. Kubernetes Enhancement Proposal (Year 3)
4. Alpha behind a feature gate (Year 4)
5. General availability (Year 5)

---

## 7. Conclusion

Kubernetes extensibility has, since its inception, relied on an external
delegation model for the capabilities that require awareness of the managed
surface.

Orkestra demonstrates that this observer can exist, that it can be built on
the existing Kubernetes primitives, and that when it exists, the external
compensations become unnecessary.

Kubernetes made infrastructure declarative.
Orkestra makes the operators that extend Kubernetes declarative.
The observer that makes this possible belongs in the runtime.
The runtime belongs in the cluster.

---

*Orkestra — Declarative Operators for Kubernetes*
*March 2026 — https://github.com/orkspace/orkestra*
