---
title: "Orkestra Registry"
weight: 71
---

# The Missing Package Manager for Kubernetes Operators

*OrkestraRegistry — Declarative Distribution for Operator Behavior*

  *Orkestra Project — March 2026*

---

## Abstract

<!-- Kubernetes operator ecosystems have historically been ecosystems of binaries.
Each operator is a separate software project, compiled into a binary, deployed
as a separate process, and maintained independently. OperatorHub.io lists
hundreds of such operators — each with its own resource footprint, its own
observability story, and its own upgrade cycle. The cumulative operational
cost of maintaining a cluster's operator layer has become a significant and
underappreciated burden for platform teams.

This paper introduces OrkestraRegistry — a registry of declarative operator
patterns distributed as OCI artifacts. Rather than distributing binaries,
OrkestraRegistry distributes behavior: YAML Katalogs that declare how a CRD
should reconcile, what resources it should create, and how it should convert
between versions. These patterns are interpreted by the Orkestra runtime,
which can compose multiple patterns into a single process. We describe the
architecture of the registry, the distribution model, the promotion path from
typed extensions to declarative core patterns, and the long-term vision of a
composable, versioned operator ecosystem. We argue that this model does for
operators what package managers did for software dependencies: it makes
reuse the default. -->

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

## 1. The Binary Operator Ecosystem

### 1.1 The Current State

OperatorHub.io lists operators for databases, message queues, monitoring
stacks, security tools, and platform infrastructure. Each entry represents
a binary that a platform team must install, configure, maintain, and upgrade.
A cluster running PostgreSQL, Redis, Prometheus, Cert Manager, External
Secrets, and a collection of internal CRDs routinely runs ten to twenty
operator processes.

The operational overhead is predictable. Each process consumes memory —
typically 50 to 200 MB. Each maintains its own informer cache, duplicating
API server watch traffic. Each has its own metrics endpoint with its own
format and conventions. Each has its own upgrade path, its own release notes,
its own breaking changes to navigate.

Platform teams spend significant time on work that is not central to their
business. They are not building competitive advantage by maintaining the
Postgres operator. They are keeping a dependency current.

### 1.2 Why Distribution Matters

The operator ecosystem grew this way because the tooling for sharing operator
behavior did not exist. OperatorHub distributes binaries because binaries
are what operators produce. The Helm Hub distributes charts because charts
are what Helm produces. Distribution follows production.

If operators can be produced as declarations, they can be distributed as
declarations. The economics of the ecosystem change at the point of
production.

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

The `katalog.yaml` is a standard Orkestra Katalog. It declares the CRD's
API types, reconcile templates, conversion rules, and dependency ordering.
When a consumer imports this pattern, Orkestra interprets the Katalog and
provides a full operator stack for the CRD — dedicated informer, worker pool,
workqueue, health endpoint, and metrics — with no additional code.

```yaml
# postgres/v14/katalog.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: postgres-v14

spec:
  crds:
    postgres:
      apiTypes:
        group: postgres.orkestra.io
        version: v1
        kind: Postgres
        plural: postgreses
      workers: 2
      resync: 1m
      operatorBox:
        default: true
        onCreate:
          deployments:
            - image: "postgres:{{ .spec.version | default \"14\" }}"
              replicas: "1"
              reconcile: true
          services:
            - port: "5432"
              targetPort: "5432"
              reconcile: true
```

### 2.2 Distribution as OCI artifacts

Patterns are published as OCI artifacts — the same distribution format as
container images. This is a deliberate choice with consequences for the
entire ecosystem.

OCI registries are already operated by every cloud provider, every enterprise
IT organization, and most development teams. The tooling — authentication,
access control, replication, scanning, retention policies — is mature. The
semantics — immutable tags, semantic versioning, digest pinning — are well
understood.

By using OCI as the distribution format, OrkestraRegistry inherits all of
this infrastructure without building it. Platform teams that already control
what container images their clusters can pull from apply the same controls
to operator patterns. Private registries that already host internal container
images can host internal operator patterns. The security posture is consistent.

```bash
# The same tooling that manages images manages patterns
oras pull ghcr.io/orkestra-sh/registry/postgres:v14
crane copy ghcr.io/orkestra-sh/registry/postgres:v14 \
  registry.myorg.com/operators/postgres:v14
```

### 2.3 Consuming patterns

A Komposer references OCI patterns in its `sources.oci` block:

```yaml
sources:
  registry:
    - url: ghcr.io/orkestra-sh/registry/postgres:v14
      oci: true
    - url: ghcr.io/orkestra-sh/registry/postgres:v14
      oci: true
  files:
    - ./internal-crds.yaml
spec:
  crds:
    # Production overrides — win on name conflict with any source
    postgres:
      workers: 4
      resync: 30s
```

When `ork run` starts, it resolves OCI references, fetches uncached artifacts
to `~/.orkestra/registry/`, and merges them into the runtime configuration.
Inline `spec.crds` overrides are applied last. The resulting configuration
is validated and used to start the runtime.

Two patterns, merged with local overrides, running as one process. One
health endpoint. One metrics endpoint. One deployment to monitor.

### 2.4 Discoverability through Artifact Hub

Patterns published to public OCI registries are indexed by Artifact Hub —
the discovery layer the cloud-native ecosystem uses for Helm charts, OPA
policies, Falco rules, and Kyverno policies.

Each pattern's Artifact Hub entry surfaces the CRD it manages, the versions
available, the fields it accepts, and recommended production overrides. This
makes operator discovery consistent with how teams already find Helm charts.
The difference is what they receive: not a binary, but a declaration they
can compose and override.

---

## 3. The Three Layers

OrkestraRegistry has a layered architecture. Each layer has a distinct
role.

### 3.1 Internal resource library

The foundation is a Go library inside the Orkestra codebase
(`pkg/orkestra-registry/`). It implements create, update, delete, and resolve
functions for Deployments, Services, Secrets, ConfigMaps, Jobs, CronJobs,
Pods, and more. Every function is idempotent. Every function sets
owner references. Every function applies consistent system labels.

This library is stable, tested, and maintained as part of Orkestra core.
It is the machinery that runs when a Katalog declares a resource. The
declarative interface abstracts it completely — consumers never call these
functions directly.

### 3.2 Public Katalog registry

The public registry is a Git repository (`orkestra-sh/registry`)
with two sections.

**Core Katalogs** (`orkestra-core/`) are complete, declarative operator
patterns. They require no Go code from the consumer. They declare the CRD's
reconcile behavior, conversion rules, and dependency ordering in YAML.
Core Katalogs are the primary artifact the ecosystem produces and consumes.

**Typed extensions** (`typed-extensions/`) are Go hooks for use cases that
cannot yet be expressed declaratively. They are versioned Go modules.
Consumers reference them in Katalogs via `hooks.location`. Typed extensions
are first-class contributors to the ecosystem — they are not a workaround.
They are how the ecosystem handles complexity at the frontier of what YAML
can express.

### 3.3 OCI distribution

A CI pipeline in the registry repository publishes core Katalogs as OCI
artifacts on every commit to main. Tags are immutable — once
`postgres:v14.0.0` is published, it cannot be overwritten. This guarantees
that version-pinned Komposers never change behaviour without a version bump.

---

## 4. The Promotion Path

The registry is designed around a progression from typed to declarative.

**Typed extension.** The first version of a complex pattern may require Go
hooks — for database user creation, external API calls, or logic that cannot
be expressed in template expressions. The typed extension is published and
used.

**Core Katalog.** As the pattern matures and the use cases are understood,
a declarative version is developed. The typed extension is deprecated with
a pointer to the core Katalog. The Go code remains available for consumers
on older versions.

**Built-in feature.** If the pattern is general enough — if many independent
Katalogs express the same template logic — the pattern becomes a built-in
Orkestra feature. Future Katalogs express it in one line where they previously
needed ten.

This promotion ladder is how Orkestra's declarative capability grows. The
ecosystem exercises the frontier. Successful patterns illuminate the next
feature. Orkestra core absorbs them.

The result is a virtuous cycle: community patterns drive the runtime forward.
The runtime becomes more capable. Patterns that needed Go no longer do.
Operators become more declarative over time, not less.

---

## 5. Private Registries

The OCI distribution model means OrkestraRegistry is not a centralised
service with a single registry. Any OCI-compatible registry can host
operator patterns.

Platform teams at large organisations publish curated sets of approved,
internally tested patterns to private registries. Application teams import
from the private registry rather than the public one. The security model
is identical to container image policy: control which registries the cluster
can pull from.

```yaml
sources:
  registry:
    - url: registry.myorg.com/operators/postgres:v14-hardened
      oci: true
      auth:
        type: basic
        usernameFromEnv: REGISTRY_USER
        passwordFromEnv: REGISTRY_PASSWORD
```

This enables platform teams to maintain a bill of materials for their
operator layer — exactly the same governance model they apply to their
container image layer.

---

## 6. The ork registry CLI

Full CLI support for pattern management is in active development.

The planned `ork registry` subcommand will provide:

```bash
ork registry login ghcr.io                    # authenticate
ork registry push <registry>/<n>:<v> ./dir    # publish a pattern
ork registry pull <registry>/<n>:<v> ./dir    # inspect locally
ork registry list <registry>                  # list available patterns
ork registry search <keyword>                 # search Artifact Hub
ork registry info <registry>/<n>:<v>          # metadata without pulling
```

OCI patterns are fully consumable today via direct `oci:` references in
Komposers using standard OCI tooling. The CLI is being built to provide
a first-class experience within the `ork` tool family. Release notes will
mark the stable milestone.

---

## 7. What This Changes

### For pattern consumers

Adopting a third-party operator becomes a Komposer entry and two lines of
YAML. No Helm release. No RBAC to configure. No separate deployment to
monitor. The pattern is fetched, merged with local overrides, and starts
contributing to an already-running Orkestra process.

Upgrading a pattern becomes a version bump in the Komposer. Declarative
conversion rules handle breaking field changes between versions. The upgrade
path is documented in the pattern's Katalog. The pattern author declares it
once. Consumers apply it at their own pace.

### For pattern authors

Publishing an operator pattern becomes equivalent to publishing a Helm chart.
There is no binary to compile or image to build. Write the Katalog, structure
the directory with the five required files, push to an OCI registry. The
pattern is live and consumable.

Maintaining the pattern is maintaining YAML. Field additions are additive.
Breaking changes are accompanied by conversion rules. Consumers can override
any field without forking the pattern.

### For the ecosystem

The Kubernetes operator ecosystem has needed a package manager for operator
behavior for years. The infrastructure existed for binaries — OperatorHub,
OLM, Helm — but not for declarations, because declarations were not how
operators were built.

OrkestraRegistry is that package manager. OCI is the distribution format.
Artifact Hub is the discovery layer. The Orkestra runtime is the executor.
The three pieces were independently available. OrkestraRegistry assembles
them into an ecosystem.

---

## 8. Conclusion

The operator ecosystem will not remain an ecosystem of binaries indefinitely.
The operational cost is too high. The barrier to contribution is too steep.
The reuse rate is too low.

OrkestraRegistry demonstrates the alternative: operator behavior distributed
as declarations, composed at runtime, overridden without forking, upgraded
through declarative conversion rules. The distribution infrastructure is
OCI — mature, universal, already operated by every team that runs containers.
The discovery layer is Artifact Hub — already the standard for cloud-native
artifacts. The runtime is Orkestra — already managing CRDs declaratively
in production.

The remaining work is content: populating the registry with patterns that
matter to platform teams, validating them in production, promoting typed
extensions to declarative Katalogs, and building the CLI that makes the
experience complete.

The infrastructure is ready. The ecosystem follows the patterns.

---

*Orkestra — Declarative Operators for Kubernetes*
*March 2026 — https://github.com/orkspace/orkestra*
