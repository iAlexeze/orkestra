# Declarative Operators: A New Model for Kubernetes Extensibility

*Orkestra Project — March 2026*

---

## Abstract

Kubernetes operators encode domain knowledge as reconciliation logic. Every
major operator framework to date requires this logic to be written in a
programming language, compiled into a binary, and deployed as a separate
long-running process — one per CRD. The result, at scale, is operator sprawl:
dozens of binaries each with independent lifecycle, observability, and
resource consumption.

This paper presents a different model. We argue that the operator pattern
is fundamentally correct, but its implementation has been conflated with its
mechanism. The pattern requires one reconciler per CRD with proper isolation.
It does not require one binary per CRD. When a shared runtime provides the
isolation, each CRD becomes a complete, independent operator — with its own
informer, worker pool, workqueue, health endpoint, and metrics — while the
operational overhead of running them collapses to a single process.

We describe Orkestra, a runtime for declarative operators built on this
principle. Users declare CRDs and their reconcile behavior in a YAML Katalog.
The runtime interprets these declarations, provides full operator lifecycle,
and composes multiple Katalogs through a Komposer model. We demonstrate that
this approach eliminates code generation, build pipelines, and per-operator
deployments while preserving and strengthening the isolation properties that
operator frameworks seek.

---

## 1. The Operator Pattern and Its Costs

### 1.1 The Original Model

The operator pattern, introduced in 2016, proposed encoding operational
knowledge as a reconciliation loop: a controller that watches a custom
resource and continuously drives the cluster toward the desired state declared
in it. The pattern was correct. Its implementation required intimate
familiarity with client-go internals — informers, workqueues, REST mappers,
and scheme registration. The business logic was a small fraction of the total
code. The remainder was identical across every operator ever written.

### 1.2 Frameworks Address Boilerplate

Kubebuilder, Operator SDK, and controller-runtime addressed the boilerplate
by generating the common parts. The operator developer could focus on the
reconcile function rather than its surrounding infrastructure. This was
genuine progress.

The cost did not disappear. The generated project still required Go, a build
pipeline, an image registry, and a deployment manifest. Adding a new CRD
meant adding a new Go type, running code generation, rebuilding the binary,
pushing the image, and rolling the deployment. The development loop was
compressed but not eliminated.

More significantly, the framework design encoded an assumption that would
compound over time: one operator per CRD, one binary per operator.

### 1.3 The Operator Sprawl Problem

The one-binary-per-CRD assumption produces predictable operational overhead
at scale. A production cluster running Prometheus, Cert Manager, External
Secrets, an Ingress controller, a service mesh, and a collection of internal
CRDs routinely runs twenty to fifty operator processes. Each consumes memory
and CPU even when idle. Each maintains its own informer cache, duplicating
watch traffic against the API server. Each has its own RBAC configuration,
its own metrics endpoint (if any), its own upgrade cadence, and its own
health story.

Platform engineers managing these clusters do not have one observability
problem. They have fifty. Understanding why an application failed requires
consulting multiple dashboards, multiple log streams, multiple health
endpoints — each with its own format, its own conventions, and its own
operational vocabulary.

This is operator sprawl. It is not a consequence of the operator pattern.
It is a consequence of the assumption that operator equals binary.

---

## 2. Reframing: The Super-Operator Model

The Kubernetes community has long held that the correct design is one operator
per CRD. This paper agrees — and argues that Orkestra fulfills this principle
more completely than previous frameworks.

The confusion lies in what "one operator" means. In traditional frameworks,
it means one process, one binary, one deployment. The per-CRD isolation is
an architectural intention, but it is implemented at the process boundary —
not within the runtime.

Orkestra makes the isolation explicit and structural. Each CRD in Orkestra
receives its own isolated operator stack:

- **Informer** — watches exactly one GVK with its own resync interval
- **Workqueue** — independent depth, backoff, and rate limiting
- **Worker pool** — dedicated goroutines; no other CRD can consume them
- **Reconciler** — interprets this CRD's templates and hooks only
- **Health endpoint** — `/katalog/{crd}/health` per CRD
- **Metrics** — five Prometheus metrics, all labeled by GVK
- **Failure domain** — a panic in one reconciler is recovered; others continue

These components are hosted by a shared runtime that provides the
infrastructure — API server connections, leader election, dependency ordering,
the informer factory, the queue registry. The runtime is infrastructure. The
operator stack per CRD is the tenant.

This is the super-operator model: each CRD becomes a complete, production-grade
operator. They share infrastructure the way microservices share a Kubernetes
cluster — not by merging their logic, but by running on common platforms.

---

## 3. Orkestra

### 3.1 The Katalog

A Katalog is a YAML document that declares one or more CRDs and how they
should be reconciled. It is the unit of operator definition in Orkestra.

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: website-operator
spec:
  crds:
    website:
      workers: 3
      resync: 30s
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites
      reconciler:
        default: true
        onCreate:
          deployments:
            - image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              reconcile: true
          services:
            - port: "80"
              targetPort: "{{ .spec.port }}"
              reconcile: true
```

This is a complete operator declaration. `ork run --katalog katalog.yaml`
starts the runtime. Every `Website` CR triggers a reconcile that creates
and drift-corrects a Deployment and Service. Deletion cascades via owner
references. Finalizers ensure cleanup completes before CR removal.

The field `reconcile: true` on each resource means it is also corrected on
every reconcile cycle — not just created. Drift is detected and corrected
without additional configuration.

### 3.2 The Dynamic Client Model

Orkestra operates on unstructured CRDs by default — the same
`*unstructured.Unstructured` representation Kubernetes uses internally.
The API types (`apiTypes.location`) field is optional, needed only when
Go hooks require concrete type assertions. This distinction matters for
three reasons.

First, it eliminates the code generation step for the common case. The
cluster already holds the CRD schema. Orkestra reads it at startup via the
discovery API. The user does not need to replicate it in Go structs.

Second, it enables watching any CRD — including ones the operator does not
own. A Katalog entry with `kind: Deployment` and no `apiTypes.location` is
sufficient for Orkestra to watch all Deployments in the cluster. The cluster
knows the schema. Orkestra asks for it.

Third, it is the mechanism that makes multi-version CRDs tractable. Each
version of a CRD is registered as a separate Katalog entry. Each version
gets its own complete operator stack. The conversion logic is declared
alongside the reconcile logic, interpreted by the same template resolver.

### 3.3 The Template Resolver

Orkestra's template resolver evaluates Go `text/template` expressions
against the live CR object at reconcile time. This is the mechanism
through which declarative templates become Kubernetes API calls.

```yaml
deployments:
  - name: "{{ .metadata.name }}"
    image: "{{ .spec.image }}"
    replicas: "{{ .spec.replicas }}"
    reconcile: true
```

The resolver resolves each field against the CR's unstructured map before
calling the OrkestraRegistry. The OrkestraRegistry handles the Kubernetes
API calls — create, update, delete, owner references, idempotency — for
each resource type. Adding a new resource type to the registry is a single
file addition.

### 3.4 Dependency Ordering

CRDs can declare dependencies:

```yaml
crds:
  project:
    dependsOn: []
  namespace:
    dependsOn: [project]
  application:
    dependsOn: [project, namespace]
```

Orkestra computes the topological order from the dependency graph and starts
CRDs in that order. Each CRD waits for its dependencies to signal readiness
before its workers start. Missing CRDs — declared but not yet installed in
the cluster — are retried in the background without blocking healthy CRDs.

This capability is structurally impossible with separate operators. Separate
processes have no coordination mechanism. Orkestra provides it as a declared
property of the Katalog.

Shutdown runs in reverse dependency order. No partial reconciliations. No
orphaned resources.

### 3.5 The Komposer

A Komposer composes Katalogs from multiple sources into one runtime:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: platform-komposer
sources:
  files:
    - ./katalogs/website.yaml
    - https://platform.myorg.io/crds/database.yaml
    - url: https://private.myorg.io/crds/internal.yaml
      auth:
        type: bearer
        fromEnv: PLATFORM_TOKEN
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 2.1.0
  registry:
    - katalog:
        application:
          version: v1.4.0
spec:
  crds:
    # Inline override — wins on name conflict with any source
    application:
      workers: 8
```

Orkestra's in-built merger resolves all sources, deduplicates by CRD name, and produces one
validated configuration. Inline `spec.crds` are merged last and override
source definitions — the mechanism for environment-specific configuration
without forking source Katalogs.

This is the pattern that Helm brought to deployment manifests, applied to
operator behavior. Platform teams publish Katalogs. Application teams
compose and selectively override.

### 3.6 Observability

Every CRD managed by Orkestra automatically exposes:

```
GET /katalog                   All CRDs — health, dependency graph, stats
GET /katalog/{crd}             Single CRD — config, reconcile stats
GET /katalog/{crd}/health      200 healthy / 503 degraded
GET /metrics                   Prometheus metrics for all CRDs
```

Five metrics, all per-CRD, all labeled by full GVK:

```
controller_reconcile_total{crd, result}
controller_reconcile_duration_seconds{crd}
controller_queue_depth{crd}
controller_workers_active{crd}
controller_resource_count{crd}
```

This unified observability is a structural consequence of the single-runtime
model. Separate operators cannot provide it.

---

## 4. Multi-Version CRDs: Declarative Conversion

### 4.1 The Standard Approach

Kubernetes CRDs support multiple API versions through a conversion webhook.
When a client requests a version different from the storage version, the API
server sends a `ConversionReview` request to the webhook. The webhook must
return the converted objects.

The standard implementation requires writing conversion functions in Go,
deploying a separate webhook server, managing TLS certificates, configuring
the CRD to point to the webhook, and maintaining conversion logic as versions
evolve. For a change as simple as adding a field, this infrastructure overhead
frequently exceeds the development cost of the change itself.

### 4.2 Conversion as a Consequence of the Super-Operator Model

The super-operator model makes declarative conversion architecturally natural.
Each version of a CRD is a separate Katalog entry with its own complete
operator stack — its own informer watching that specific version, its own
workers, its own reconciler. The version boundary is a first-class concept
in the runtime.

Conversion rules are declared alongside reconcile templates using the same
template resolver:

```yaml
- name: website-v1
  apiTypes:
    group: demo.orkestra.io
    version: v1
    kind: Website
  conversion:
    storageVersion: v1
    paths:
      - from: v1alpha1
        to: v1
        spec:
          image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
          seo:
            enabled: false   # default — v1alpha1 has no seo field
      - from: v1
        to: v1alpha1
        spec:
          image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
          theme: "default"   # default — v1 has no theme field
```

Orkestra's HTTPS server serves the `/convert` endpoint. The conversion
handler resolves each path's spec template against the source object and
returns the converted objects. The CRD's `conversion` block points to this
endpoint.

### 4.3 Production Results

The following is from a live deployment managing both versions of the
Website CRD:

```json
{
  "name": "website-v1alpha1",
  "conversion": {
    "enabled": true,
    "total": 62,
    "success": 62,
    "failures": 0,
    "avgLatencyMs": 0.5,
    "p95LatencyMs": 1.2
  }
}
```

```
orkestra_conversion_requests_total{kind="Website",from="v1alpha1",to="v1",result="success"} 14
orkestra_conversion_requests_total{kind="Website",from="v1",to="v1alpha1",result="success"} 17
orkestra_conversion_duration_seconds_sum{from="v1alpha1",to="v1"} 0.007
```

62 successful conversions. Zero failures. Sub-millisecond average latency.
Zero lines of Go written for the conversion. Zero additional deployments.
Zero TLS certificates to manage.

---

## 5. Addressing the One-Operator-Per-CRD Principle

### 5.1 The Principle is Correct

The Operator SDK best practices document states: "Avoid a design solution
where more than one Kind is reconciled by the same controller." The concerns
articulated are encapsulation, Single Responsibility, cohesion, and
preventing unexpected side effects between CRDs.

These concerns are valid. Orkestra does not disagree with them.

### 5.2 The Principle Refers to Reconciler Logic, Not Process Boundaries

The Operator SDK's concerns are about reconciler logic — that a reconciler
for `Website` should not contain logic for `Database`. Orkestra enforces
this more strictly than any previous framework.

In Orkestra, the reconciler for `Website` is a closure that closes over
exactly the Katalog entry for `Website`. It has no access to the reconciler
for `Database`. It cannot affect the `Database` workqueue. It cannot observe
the `Database` informer. The isolation is structural.

What is shared is the orchestration infrastructure: the API server connection,
the informer factory, the queue registry, the health server, the leader
election lease. This is analogous to how the Kubernetes
`kube-controller-manager` runs the Deployment controller, the ReplicaSet
controller, the Endpoint controller, and dozens of others in a single process —
each isolated, each maintaining the Single Responsibility Principle, all
sharing infrastructure.

No one argues that `kube-controller-manager` violates good design. Orkestra
applies the same principle to user-defined operators.

### 5.3 The Shared Failure Domain

The legitimate remaining concern is that a single process is a single failure
domain. If the Orkestra process crashes, all CRD operators stop together.

This is addressed the same way Kubernetes addresses it for
`kube-controller-manager`: multiple replicas with leader election and warm
informer caches. Followers are not idle — they maintain synced informer
caches. When the leader fails, a follower takes over within the lease renewal
period, already warm.

Within the process, Orkestra's `safeReconcile` wrapper recovers from panics
in individual reconcilers. A `nil` pointer dereference in the `Website`
reconciler does not crash the `Database` reconciler.

---

## 6. What This Replaces

| Traditional | Orkestra |
|---|---|
| Go operator binary per CRD | Katalog YAML |
| Kubebuilder scaffolding | `ork init` |
| One deployment per CRD | One runtime per operator surface |
| Per-operator health endpoints | Unified `/katalog/*` API |
| Per-operator metrics (or none) | Unified per-CRD metrics |
| Manual dependency management | Declared `dependsOn` |
| Helm chart per operator | Komposer |
| Conversion webhook binary + TLS | Conversion rules in Katalog |
| Operator sprawl | One runtime |

---

## 7. Conclusion

The operator pattern is the right abstraction for Kubernetes extensibility.
The requirement to implement one binary per CRD has been a constraint of
implementation convention, not of the pattern itself.

Orkestra demonstrates that when a runtime provides per-CRD isolation
structurally — through dedicated informers, workqueues, worker pools, and
failure domains — the isolation properties the community seeks are preserved
and strengthened. Each CRD becomes a complete, production-grade operator.
They share only the infrastructure that makes this possible.

The consequences extend beyond simplification. When each CRD version is its
own operator entry, multi-version conversion becomes a declaration rather
than an infrastructure project. When operators are YAML declarations, they
become composable through the same mechanisms as any other Kubernetes
resource. When one runtime watches all managed CRDs, unified observability
is a structural consequence, not an integration project.

Operators become data, not code.
They are composed, not programmed.
They are versioned, shared, and reused like any other Kubernetes resource.

Kubernetes made infrastructure declarative.
Orkestra makes the operators that extend Kubernetes declarative.
The same principle, applied one level up.

---

*Orkestra — Declarative Operators for Kubernetes*
*March 2026 — https://github.com/iAlexeze/orkestra*