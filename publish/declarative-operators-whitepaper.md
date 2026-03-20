# Declarative Operators: A New Model for Kubernetes Extensibility

*Orkestra Project — March 2026*

---

## Abstract

Kubernetes operators encode domain knowledge as reconciliation logic.
Every major operator framework to date requires this logic to be written
in Go, compiled into a binary, and deployed as a long-running process.
This paper argues that the reconciliation model itself — not the language
or the framework — is the right abstraction, and that the implementation
can be separated from it. We introduce declarative operators: operators
whose reconcile behavior is expressed as structured declarations interpreted
at runtime, with no compiled implementation required. We describe the
Orkestra runtime, the primitives it introduces (Katalog, Komposer, the
OrkestraRegistry), and the architectural patterns that emerge from treating
operators as data rather than code.

---

## 1. The operator model and its costs

The Kubernetes operator pattern, introduced in 2016, extends the Kubernetes
API with domain-specific resources and encodes the operational knowledge
required to manage them. An operator watches Custom Resource (CR) events
and reconciles the actual state of the cluster toward a desired state
declared in the CR.

The pattern is powerful. It has enabled a generation of platform tooling —
database operators, certificate managers, service mesh controllers, and more.
But the implementation model has remained fixed: every operator is a Go
binary, built around an informer, a workqueue, and a reconcile function.

The cost of this model is consistently underestimated:

**Cognitive load.** Building an operator requires deep familiarity with
Kubernetes client-go internals — schemes, informers, RESTMappers, workqueues,
finalizers, and owner references. These are not domain concepts. They are
infrastructure for implementing domain concepts. A developer building a
database operator spends more time on the infrastructure than on the
database knowledge.

**Operational sprawl.** Modern platforms run dozens of operators. Each
consumes resources, requires an independent update cycle, exposes its own
metrics (inconsistently), and implements its own health checking. Platform
teams spend significant effort managing the operators that manage their
platform.

**Inaccessibility.** Operators encode domain knowledge. Domain experts are
not always Go engineers. The requirement to write Go excludes the people
most qualified to define what an operator should do.

**Duplication.** Every operator reimplements the same patterns: ensuring a
Deployment exists, copying a Secret to another namespace, setting owner
references, handling deletion. These patterns are not operator-specific.
They are resource management primitives that belong in a shared library.

---

## 2. The reconciliation loop as an abstraction

The core of every operator is the reconcile function:

```
observe current state
compare to desired state
apply changes to close the gap
```

This pattern is not specific to Go. It is an algorithm. The desired state
comes from the CR spec. The current state comes from the informer cache.
The changes are API calls to create, update, or delete resources.

What if the reconcile function could be declared rather than implemented?

```yaml
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

This declaration is fully equivalent to a reconcile function that ensures
a Deployment and a Service exist with the specified configuration and
corrects drift on every loop. The template expressions resolve against the
live CR at reconcile time. The `reconcile: true` flag means the resource
is also updated on every reconcile, not just created.

The runtime interprets this declaration. The developer writes nothing else.

---

## 3. The Orkestra model

Orkestra is a runtime for declarative operators. It introduces two document
kinds:

**Katalog** — declares one or more CRDs and how they should be reconciled.
A Katalog is a complete operator definition. Just run
`ork run --katalog ./katalog.yaml` and the operator is live.

**Komposer** — composes Katalogs from multiple sources into one runtime.
A Komposer declares where CRD definitions come from — local files, remote
URLs, Helm charts — and optionally overrides specific definitions for the
current environment.

The separation between Katalog and Komposer mirrors the separation between
a Deployment manifest and a Helm chart. The Katalog defines what an operator
does. The Komposer defines how definitions are composed and distributed.

### 3.1 The Katalog

A Katalog entry for a dynamic CRD:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: platform-namespace-katalog
spec:
  crds:
    - name: platformnamespace
      enabled: true
      namespaced: false
      workers: 2
      resync: 1m
      apiTypes:
        group: platform.orkestra.io
        version: v1alpha1
        kind: PlatformNamespace
        plural: platformnamespaces
      reconciler:
        default: true
        finalizers:
          - finalizer.platform.orkestra.io/platformnamespace
        onCreate:
          configMaps:
            - name: "{{ .metadata.name }}-config"
              namespace: "{{ .spec.targetNamespace }}"
              data:
                ENVIRONMENT: "{{ .spec.environment }}"
                LOG_LEVEL: "{{ .spec.logLevel }}"
                TEAM: "{{ .spec.team }}"
              reconcile: true
          secrets:
            - name: registry-pull-secret
              fromSecret: docker-registry-creds
              fromNamespace: platform
              namespace: "{{ .spec.targetNamespace }}"
              reconcile: true
          serviceAccounts:
            - name: "{{ .spec.team }}-sa"
              namespace: "{{ .spec.targetNamespace }}"
```

This is a complete namespace provisioning operator. For every
`PlatformNamespace` CR, Orkestra creates a ConfigMap with environment
configuration, copies the registry pull secret into the provisioned
namespace, and creates a ServiceAccount. Owner references are set on all
child resources. Deletion of the CR cascades automatically. The ConfigMap
is kept in sync on every reconcile — if `spec.logLevel` changes in the CR,
the ConfigMap updates.

No Go was written.

### 3.2 CRD modes

A CRD entry is either dynamic or typed.

**Dynamic** (default) — no `apiTypes.location`. The runtime uses the dynamic
Kubernetes client and stores objects as `*unstructured.Unstructured`. Template
expressions have full access to all spec fields at runtime. No compiled Go
types needed.

**Typed** — `apiTypes.location` set. The runtime uses compiled Go types
registered at startup via `ork generate runtime`. The reconcile function
receives a typed object — `obj.Spec.Image` instead of a map lookup. Required
when Go hooks need compile-time type safety.

### 3.3 The reconcile implementation hierarchy

Orkestra's GenericReconciler selects the reconcile implementation in
priority order:

1. **Go hooks** — user-provided function with typed CR access. Handles
   cases requiring conditional logic, external API calls, or status writes.
   Registered via `ork generate runtime`.

2. **Declarative templates** — runtime interpretation of
   `onCreate`/`onReconcile`/`onDelete` blocks. No code generation needed.
   The common path for dynamic operators.

3. **No-op** — GenericReconciler manages finalizers, events, and metrics
   without any reconcile logic. Useful for lifecycle tracking only.

When `reconciler.default: false`, the user provides a custom constructor
and owns the entire reconcile function. Orkestra still provides the
informer, workqueue, dependency ordering, health API, and leader election.

---

## 4. The OrkestraRegistry

The OrkestraRegistry is a library of Kubernetes resource implementations.
Each implementation handles create, update, delete, owner references,
idempotency, and drift detection for one resource type.

Current implementations: Deployment, Service, Secret, ConfigMap,
ServiceAccount, Job, CronJob, Pod.

The registry provides two patterns specific to platform engineering:

**Cross-namespace distribution.** Secrets and ConfigMaps can be declared
with `toNamespaces` — the registry reads the source once and writes copies
to every listed namespace. When the source changes, `reconcile: true`
keeps all copies in sync.

```yaml
secrets:
  - name: registry-pull-secret
    fromSecret: docker-registry-creds
    fromNamespace: platform
    toNamespaces:
      - "{{ .metadata.namespace }}"
      - monitoring
      - staging
    reconcile: true
```

**Merge with override.** ConfigMaps support `fromConfigMap` — the registry
copies data from a source ConfigMap and applies declared `data` keys as
overrides. This provides a base-plus-override pattern without forking the
source.

```yaml
configMaps:
  - name: app-config
    fromConfigMap: base-app-config
    fromNamespace: platform
    data:
      LOG_LEVEL: "{{ .spec.logLevel }}"  # overrides the base value
    reconcile: true
```

The registry is extensible. Adding a new resource type requires a type
definition, a resolver method, a registry package (Create, Update, Delete,
Resolve), a runner file in the reconciler package, and one call in
`runTemplateReconcile`. The Katalog accepts the new resource type
immediately.

---

## 5. Composition through Komposers

Platform engineering at scale requires distributing and composing operator
definitions across teams and environments.

A Komposer addresses this directly:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: platform-komposer
sources:
  files:
    - ./katalogs/website.yaml
    - https://raw.github.com/myorg/app-crds/main/katalog.yaml
    - $SECURITY_KATALOG_URL
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 2.1.0
      valueFiles:
        - ./values/production.yaml
spec:
  crds:
    - name: application
      workers: 4            # production override
      apiTypes:
        group: platform.myorg.io
        version: v1alpha1
        kind: Application
        plural: applications
      reconciler:
        default: true
```

The merger resolves all sources, deduplicates by CRD name, and produces
one validated set of CRD entries. Inline `spec.crds` on a Komposer are
overrides — they win on name conflict with any source. This allows consuming
a shared Katalog and overriding specific CRD configuration for the current
environment without forking the source.

**Merge rules** are enforced:
- Duplicate names across independent sources are errors with source attribution
- Inline overrides are silent and intentional
- Inline self-duplicates are errors
- A Komposer cannot source another Komposer — composition chains are kept
  shallow and predictable

---

## 6. Dependency-aware lifecycle

Operators managing multiple CRDs face a fundamental problem: resources that
depend on each other must be started and stopped in the right order.
Traditional operators handle this implicitly and inconsistently.

Orkestra makes dependencies explicit:

```yaml
crds:
  - name: project
    dependsOn: []

  - name: managednamespace
    dependsOn: [project]

  - name: application
    dependsOn: [project, managednamespace]
```

The runtime builds a directed acyclic graph from these declarations,
validates it for cycles and missing references, and computes topological
order. CRDs start in dependency order. Shutdown runs in reverse.
Dependents wait for dependencies to signal readiness before their workers
start.

Missing CRDs at startup — declared but not yet installed on the cluster —
are retried in the background without blocking healthy CRDs.

---

## 7. Observability as infrastructure

Every Orkestra operator exposes a consistent set of endpoints and metrics
regardless of what CRDs it manages.

**HTTP endpoints** (automatic, no configuration):

```
GET /health                       liveness probe
GET /ready                        readiness probe
GET /metrics                      Prometheus metrics
GET /katalog                      all CRDs — health, config, dependency graph
GET /katalog/{crd}                single CRD — config, reconcile stats
GET /katalog/{crd}/health         200 healthy / 503 degraded
```

**Prometheus metrics** (per CRD, labelled by GVK):

```
controller_resource_count                     live CR count from informer cache
controller_reconcile_total                    success/error count
controller_reconcile_duration_seconds         latency histogram
controller_queue_depth                        current queue backlog
controller_workers_active                     active worker count
controller_crd_activation_latency_seconds     CRD activation latency histogram for missing CRDs 
controller_crd_activation_total               CRD activation count for missing CRDs
```

An organisation running ten Orkestra operators has ten operators with
identical operational characteristics. The same Grafana dashboard, the same
alert rules, the same runbooks apply to all of them.

---

## 8. High availability

Orkestra uses Kubernetes leader election (via the konductor package).
All replicas run informers — their caches are warm on every pod. Only the
elected leader runs workers. On leadership loss the lease is released
immediately and a follower takes over with an already-warm cache.
Failover takes milliseconds, not the time to sync a cold informer.

---

## 9. The operator sprawl problem

Modern platforms commonly run fifteen to thirty operators: Prometheus,
cert-manager, external-secrets, crossplane providers, database operators,
and more. Each is a separate binary, a separate deployment, a separate
metrics endpoint (if it has one at all), and a separate operational concern.

Orkestra's multi-CRD runtime addresses this directly. One runtime manages
many CRDs. The dependency graph keeps them correctly ordered. The health API
provides a unified view. The metrics follow a consistent schema.

What previously required fifteen deployments becomes one runtime with
fifteen Katalog entries.

This is not just operational simplification. It is architectural
simplification. Teams no longer need to understand fifteen different operator
frameworks, fifteen different configuration formats, and fifteen different
health models.

---

## 10. The escape hatches

Declarative operators cover the majority of operator use cases. The following
patterns require code:

**Complex state machines.** CRs that progress through multiple phases with
branching logic based on external state — provisioning pipelines, multi-step
migrations — require Go hooks or a custom constructor.

**External API calls.** Operators that provision cloud resources, call DNS
APIs, or interact with external systems need the custom path.

**Status writes.** Writing derived state back to `obj.Status` requires
reading child resource state and making conditional decisions that template
declarations cannot express.

For these cases Orkestra provides Go hooks (user-provided function, framework
manages everything else) and custom constructors (user owns the reconcile
function, framework manages informer, workqueue, and lifecycle).

The escape hatches are additive. A dynamic CRD can be migrated to Go hooks
without changing anything else. The Katalog entry gains a `hooks` declaration.
The rest is unchanged.

---

## 11. Implications

### Operators as artifacts

A Katalog is a versioned, diffable, promotable artifact. It can live in Git,
be rendered with Helm, be reviewed in a pull request, and be deployed with
the same tooling as any other manifest. Operators become part of the
infrastructure-as-code story rather than a special category requiring
separate processes.

### Operator distribution

The Komposer model enables operator distribution. Platform teams publish
Katalogs as Helm charts or at well-known URLs. Application teams consume
them with a single source declaration. Overrides are explicit and localized.
This mirrors how Helm enabled application distribution — the same pattern
applied to operator behavior.

### Non-engineer accessibility

The person who understands a domain — how namespaces should be provisioned,
how databases should be backed up — can now write the operator. They need
YAML and an understanding of their domain. They do not need Go, client-go,
or controller-runtime.

### Velocity

A declarative operator is written in minutes and modified in seconds.
A YAML change takes effect on the next `ork run`. There is no build,
no image push, no deployment rollout in the development loop.

---

## 12. Conclusion

The operator pattern is the right abstraction for Kubernetes extensibility.
The requirement to implement that abstraction in Go has been a constraint of
convention, not necessity.

Declarative operators demonstrate that the reconciliation model can be
separated from its implementation. A runtime that interprets reconciliation
declarations provides the same correctness guarantees as a compiled reconciler —
idempotency, drift correction, cascade deletion, finalizer management — with
none of the implementation burden.

Orkestra is the first runtime to fully realise this model. The Katalog
primitive, the Komposer composition model, the OrkestraRegistry, and the
dependency-aware lifecycle provide a complete foundation for declarative
operators at any scale.

Operators are no longer software projects. They are infrastructure
definitions. Orkestra is the runtime that makes this possible.

---

*Orkestra — Declarative Operators for Kubernetes*
*March 2026*
*https://github.com/iAlexeze/orkestra*