---
title: "One Runtime, Many CRDs: Fulfilling the Operator Pattern"
weight: 50
description: "*Orkestra Project — March 2026*"
---

*Orkestra Project — March 2026*

---

## Abstract

Kubernetes operator frameworks have consistently advocated for one operator
per CRD. This principle, grounded in software engineering fundamentals —
separation of concerns, the Single Responsibility Principle, encapsulation —
is correct. This paper argues, however, that previous frameworks conflated
the principle with its implementation, equating "one operator per CRD" with
"one binary per CRD."

We introduce the super-operator model: each CRD receives a complete,
isolated operator stack — its own informer, workqueue, worker pool,
reconciler, health endpoint, and failure domain — hosted by a shared
orchestration runtime. This model fulfills the one-operator-per-CRD
principle more completely than previous frameworks, while eliminating the
operational overhead that the one-binary-per-CRD assumption produces at
scale.

We describe Orkestra's implementation of this model, examine how it
addresses each of the original concerns raised against multi-CRD runtimes,
and demonstrate new capabilities that become possible when per-CRD isolation
is structural rather than conventional — including declarative composition,
cross-CRD dependency ordering, and declarative version conversion.

---

## 1. Introduction

### 1.1 The One-Operator-Per-CRD Principle

The Operator SDK best practices document is explicit: "Avoid a design
solution where more than one Kind is reconciled by the same controller."
The rationale is grounded in software engineering fundamentals: a controller
that manages multiple CRDs risks becoming a tangled monolith where changes
to one resource inadvertently affect another. Encapsulation, Single
Responsibility, and cohesion all point toward the same conclusion — keep
controllers focused.

This guidance is correct. This paper does not argue against it.

### 1.2 The Conflation

The problem is that "one operator per CRD" has been implemented as "one
binary per CRD." These are not the same statement. The first is an
architectural principle about reconciler isolation. The second is an
implementation choice about deployment topology.

The distinction matters because the implementation choice has accumulated
significant operational cost at scale — operator sprawl — while the
architectural principle it was meant to enforce can be achieved more
completely through a different implementation.

### 1.3 Operator Sprawl

Production Kubernetes clusters running modern workloads commonly run between
twenty and fifty operator processes. Each is a separate binary with separate
memory consumption, separate CPU consumption, separate API server watch
connections, separate RBAC configuration, separate metrics endpoints, separate
upgrade schedules, and separate failure domains.

The operational overhead is substantial. More significantly, the fragmentation
makes unified understanding of the cluster's extension layer impossible.
Observing the state of all CRDs requires consulting dozens of different
endpoints with no consistent format or vocabulary.

This is operator sprawl. It is a consequence of the one-binary-per-CRD
assumption, not of the one-operator-per-CRD principle.

---

## 2. The Super-Operator Model

### 2.1 Reframing Isolation

Orkestra's central claim is that per-CRD isolation does not require per-CRD
processes. Isolation is a property of runtime architecture, not of process
boundaries.

In Orkestra, each CRD receives its own dedicated runtime stack:

**Informer.** A `cache.SharedIndexInformer` watching exactly one GVK with
its own configured resync interval. No other CRD shares this informer or
its cache.

**Workqueue.** An independent workqueue with its own configured maximum
depth, backoff settings, and rate limiting. Items from one CRD's queue do
not compete with items from another's. A queue backlog for `Database` does
not starve `Website` workers.

**Worker pool.** A fixed pool of goroutines dedicated to this CRD.
No other CRD can consume these workers. A CRD with `workers: 3` always has
three workers available for its reconciliation, independent of what other
CRDs are doing.

**Reconciler.** A closure that captures exactly the Katalog configuration
for this CRD. It has no reference to any other CRD's configuration,
templates, or hooks.

**Health endpoint.** `/katalog/{crd}/health` returns the health state of
this specific CRD, driven by its own `CRDHealth` tracker.

**Metrics.** Five Prometheus metrics labeled by full GVK string. They measure
this CRD's behavior and cannot be confused with any other's.

**Failure domain.** The `safeReconcile` wrapper catches panics in the
reconciler goroutine, records them as errors, and continues. A nil pointer
dereference in the `Website` reconciler does not crash the `Database`
reconciler.

This is the super-operator model: each CRD gets everything a traditional
operator provides. What is shared is only the orchestration infrastructure —
the API server connections, the informer factory, the queue registry, the
health server, the leader election lease. The runtime is infrastructure.
The per-CRD stacks are the tenants.

### 2.2 The Analogy to kube-controller-manager

Kubernetes itself runs dozens of controllers — Deployment, ReplicaSet,
Endpoint, StatefulSet, Job, and more — in a single `kube-controller-manager`
process. Each controller is isolated, each maintains Single Responsibility,
and none affects the others.

Nobody argues that `kube-controller-manager` violates the Single
Responsibility Principle because it runs many controllers. The principle
applies to each controller's code, not to the process that hosts them.

Orkestra applies the same reasoning to user-defined operators. The runtime
is the `kube-controller-manager`. The per-CRD stacks are the controllers.

### 2.3 Your CRD Is a Super-Operator

The practical consequence of this model is striking. When a user declares
a CRD in a Katalog and runs `ork run`, that CRD receives:

- A full operator lifecycle (start, run, drain, shutdown)
- An informer with configurable resync
- A worker pool with configurable concurrency
- A workqueue with configurable depth and backoff
- A health endpoint returning live reconcile statistics
- Five Prometheus metrics
- Kubernetes event emission on every operation
- Finalizer management
- Owner reference and cascade deletion
- Leader election participation
- Graceful shutdown with in-flight reconcile completion

In a traditional framework, these capabilities require hundreds of lines of
Go and a full operator project. In Orkestra, they are provided automatically
as a consequence of declaring the CRD. The user writes a Katalog entry.
Orkestra provides the operator.

---

## 3. Addressing the Original Concerns

### 3.1 Encapsulation

The concern: when multiple CRDs share a controller, the controller knows
about all of them, creating coupling.

In Orkestra, each CRD's reconciler is a closure over exactly that CRD's
configuration. The `Website` reconciler captures the `Website` Katalog
entry and nothing else. It cannot access the `Database` reconciler's
configuration or state. The encapsulation is structural, enforced by the
runtime rather than by convention.

### 3.2 Single Responsibility Principle

The concern: a controller managing multiple CRDs has multiple reasons to
change.

In Orkestra, the runtime has one responsibility: providing operator
infrastructure to CRD entries. Each CRD entry has one responsibility:
reconciling its CRD type. Changes to one CRD's reconcile templates do not
affect any other. The per-CRD configuration is isolated in the Katalog
file, the registry, and the closure.

### 3.3 Cohesion

The concern: operations for different CRDs may not be logically related,
creating artificial dependencies when grouped together.

Orkestra enables positive cohesion that separate operators cannot achieve.
CRDs that are logically related — a `Database` that must start before an
`Application` that uses it — can declare this relationship explicitly through
`dependsOn`. The runtime enforces the ordering. This is coordination without
coupling: each CRD remains independent, but their lifecycle relationship is
declared.

CRDs that are not related have no interaction whatsoever in the runtime.
Their workers, queues, informers, and reconcilers are fully independent.

### 3.4 Unexpected Side Effects

The concern: a change to one CRD's reconciler might affect others.

Per-CRD closures, per-CRD queues, and per-CRD workers eliminate the
mechanism for side effects. The only coordination between CRDs in Orkestra
is through the dependency graph (explicit, declared) and the API server
(mediated by Kubernetes itself). There is no shared mutable state between
reconcilers.

### 3.5 Testing Complexity

The concern: testing a multi-CRD controller requires testing interactions
between CRDs.

Because each reconciler is a closure, it can be tested in isolation with
a fake informer and a fake Kubernetes client. The runtime itself can be
tested with mock components. The interactions that matter — dependencies —
are explicit and testable through the dependency graph's topological ordering.

---

## 4. New Capabilities

### 4.1 Unified Observability

When one runtime hosts all CRD operators, unified observability is a
structural consequence rather than an integration project.

```
GET /katalog                   All CRDs in one response
GET /katalog/{crd}             One CRD — config, stats, conversion data
GET /katalog/{crd}/health      200 or 503, per-CRD
GET /metrics                   All CRD metrics, labeled by GVK
```

This is impossible with separate operators. Orkestra provides it automatically.

### 4.2 Declared Dependency Ordering

```yaml
crds:
  - name: project
    dependsOn: []
  - name: namespace
    dependsOn: [project]
  - name: application
    dependsOn: [project, namespace]
```

Startup is in topological order. Shutdown is in reverse. Dependents wait for
dependencies. Missing CRDs are retried without blocking healthy ones.

This is architecturally impossible with separate processes. Separate operators
have no coordination mechanism. They start in whatever order the scheduler
chooses and fail unpredictably when dependencies are unavailable.

### 4.3 Declarative Version Conversion

The super-operator model makes multi-version CRDs tractable. Each version of
a CRD is a separate Katalog entry with its own operator stack. Conversion
rules are declared alongside reconcile templates in the same Katalog entry,
evaluated by the same template resolver.

```yaml
- name: website-v1
  apiTypes:
    version: v1
    kind: Website
  conversion:
    storageVersion: v1
    paths:
      - from: v1alpha1
        to: v1
        spec:
          image: "{{ .spec.image }}"
          seo:
            enabled: false
```

This runs in production today: 62 conversions, zero failures, sub-millisecond
latency, zero lines of Go written. The companion paper, *Declarative Version
Conversion for Kubernetes CRDs*, provides a full treatment.

### 4.4 Declarative Composition

Katalogs are data. A Komposer composes them from files, Helm charts, remote
URLs, and private registries. Platform teams publish CRD operator definitions.
Application teams consume and selectively override. Environment-specific
configuration is a Komposer override, not a fork.

```yaml
kind: Komposer
imports:
  files:
    - https://platform.myorg.io/crds/base.yaml
  registry:
    - katalog:
        application:
          version: v2.0.0
spec:
  crds:
    - name: application
      workers: 8   # production override
```

### 4.5 Resource Efficiency

A live Orkestra instance managing two CRD versions with active reconciliation
and conversion:

```
process_resident_memory_bytes   49,176,576  (~47 MB)
go_goroutines                   41
go_memstats_alloc_bytes         4,261,264   (~4 MB heap)
```

Two complete operator stacks, a running HTTPS server, active informers,
active worker pools, and 62 processed conversions — in 47 MB. Five separate
operators would consume several hundred MB while providing no coordination
between them.

---

## 5. The Shared Failure Domain

The legitimate concern with single-process multi-CRD runtimes is the shared
failure domain. A process crash stops all CRD operators simultaneously.

Orkestra addresses this through the same mechanism Kubernetes uses for
`kube-controller-manager`:

**Multiple replicas.** All replicas run informers — their caches are warm
on every pod. Only the leader runs workers. When the leader fails, a follower
takes over within the lease renewal period with an already-synced cache.

**Pod anti-affinity.** Replicas are scheduled on different nodes. A node
failure removes one replica, not all.

**Within-process recovery.** The `safeReconcile` wrapper catches panics in
individual reconcilers. Other CRDs continue unaffected.

This is the same operational profile as `kube-controller-manager` in a
production cluster. The Kubernetes project has operated this way for years.

---

## 6. Conclusion

The one-operator-per-CRD principle is correct. Orkestra does not challenge
it. Orkestra fulfills it more completely than previous frameworks by making
per-CRD isolation structural rather than conventional.

When isolation is provided by the runtime — through dedicated informers,
queues, worker pools, closures, and failure recovery — each CRD becomes a
complete, independent operator. The shared infrastructure that hosts these
operators is not a violation of isolation but the mechanism that makes
full isolation economically viable.

The result resolves the central tension in operator design: per-CRD
isolation is preserved and strengthened, while the one-binary-per-CRD
assumption that produced operator sprawl is abandoned.

Each CRD becomes a super-operator. They share only what they should share:
infrastructure, not logic.

---

*Orkestra — Declarative Operators for Kubernetes*
*March 2026 — https://github.com/orkspace/orkestra*
