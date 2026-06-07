# Concepts

Concepts are the building blocks that make Orkestra expressive without being verbose. Each concept is a named abstraction — you declare intent, and Orkestra expands it into the concrete configuration before the runtime starts.

---

## Orkestra Patterns

Every file you write in Orkestra — `katalog.yaml`, `simulate.yaml`, `komposer.yaml` — is an **Orkestra Pattern**: a versioned, distributable artifact with a specific kind and a specific job. The name reflects something real: these files don't just describe resources, they encode solutions to recurring problems in the operator world.

→ [Read: Orkestra Patterns](patterns/)

---

## Profiles

[Profiles](profiles/) are named presets that expand into fully-formed configuration at Katalog load time. They cover resources, autoscaling, probes, and security. A profile is a decision made once by someone who thought it through, shared with everyone who shouldn't have to.

→ [Read: Profiles](profiles/)

---

## Operator Autoscaler

The [Operator Autoscaler](operator-autoscaler/) is a built-in subsystem that dynamically adjusts an operator's worker count, queue depth, and resync interval based on runtime metrics, time windows, and cron expressions. Fully declarative. No external controllers.

→ [Read: Operator Autoscaler](operator-autoscaler/)

---

## OperatorBox

_CRDs in. Operators out._

The [OperatorBox](operatorbox/) is the fundamental execution unit in the Orkestra runtime. Each CRD declaration in a Katalog becomes one operatorBox — an isolated operator with its own informer, queue, worker pool, and health state. Think containers, but for CRDs.

→ [Read: OperatorBox](operatorbox/)

---

## Dependency Model

The [Dependency Model](dependency-model/) is how Orkestra starts your operators in the right order — automatically. Declare `dependsOn` in your Katalog. Orkestra builds the graph, waits for missing CRDs, and shuts down in reverse. No code required.

→ [Read: Dependency Model](dependency-model/)

---

## Status Management

[Status Management](status-management/) explains how Orkestra writes CR status automatically — and how to extend it. Three layers: automatic `Ready` conditions, declarative status fields templated from the CR spec, and live child resource state propagated back into status.

→ [Read: Status Management](status-management/)

---

## Reconciler Model

The [Reconciler Model](reconciler-model/) is how a CR becomes a running resource. Orkestra follows 15 ordered steps — from cache read to status patch — every time a CR is created, updated, or deleted.

→ [Read: Reconciler Model](reconciler-model/)

---

## Ordered Deletion

[Ordered Deletion](ordered-deletion/) controls the sequence in which child resources are torn down when a CR is deleted. Two models: hard ordered (finalizer held, sequential groups) and condition-based (non-blocking, `when:`/`anyOf:` conditions).

→ [Read: Ordered Deletion](ordered-deletion/)

---

## Operator of Operators

The [Operator of Operators](operator-of-operators/) pattern lets one Orkestra operator create Custom Resources that are managed by other operators in the same Katalog — declarative sub-operator composition.

→ [Read: Operator of Operators](operator-of-operators/)

---

## Typed Operators

[Typed Operators](typed-operators/) are the escape hatch for cases that genuinely need Go code: hooks that run alongside declarative templates, constructors that replace the reconciler entirely, and operator-as-library for full control.

→ [Read: Typed Operators](typed-operators/)

---

## Health Subsystem

The [Health Subsystem](health-subsystem/) is Orkestra's liveness, readiness, and metrics surface. Four Kubernetes-native probe endpoints, per-CRD health tracking, and the `/katalog` API that powers the Control Center.

→ [Read: Health Subsystem](health-subsystem/)

---

## Conditionals

[Conditionals](conditional/) are the logic layer — `when:` and `anyOf:` blocks that control when a resource is created, when a status field is written, and how multi-phase async workflows sequence themselves. Works in Katalogs and Motifs.

→ [Read: Conditionals](conditional/)

---

## ONCOP

[ONCOP](oncop/) (Orkestra Native Cross-Operator Protocol) is the cross-binary observation layer. One operator reads another's typed state — health, metrics, or full CR — without hard-coded URLs, with built-in caching, and with the same template surface as same-binary cross: reads.

→ [Read: ONCOP](oncop/)

---

## Declarative Unit Testing

[Declarative Unit Testing](simulate/) is how Orkestra verifies reconciler behavior without a cluster — one `simulate.yaml`, no Kubernetes, no Docker, sub-second. Declare which resources should appear in which cycle. The same reconciler that runs in production runs here.

→ [Read: Declarative Unit Testing](simulate/)

---

## Declarative End-to-End Testing

[Declarative E2E](e2e/) is how Orkestra verifies an operator against a real cluster — one YAML file, no test framework, no Go. Every learning example ships with a runnable `e2e.yaml`. The `imports:` field composes focused per-Katalog tests into suites that a single `ork e2e` command runs end to end.

→ [Read: Declarative End-to-End Testing](e2e/)
