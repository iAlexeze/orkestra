# Why Orkestra?

Kubernetes has always promised declarative infrastructure.
You describe what you want. The platform makes it so.

That promise holds everywhere — until you need to extend Kubernetes itself.

The moment you need a custom resource, you leave the declarative world.
You write Go. You scaffold controllers. You wire informers and schemes.
You manage reconcile loops, retries, and backoff. You build images. You
write deployment manifests for your controller. You maintain a project
whose primary purpose is to watch another project.

Every major operator framework to date has accepted this as the cost of
entry. Kubebuilder, Operator SDK, Metacontroller — they each make the Go
easier, not unnecessary.

This creates a paradox: **to make Kubernetes more declarative, you must
write imperative code.**

Orkestra breaks that paradox.

---

## How we got here

Understanding why Orkestra exists requires understanding how the operator
pattern evolved — not as a criticism of what came before, but as an honest
account of how each improvement brought us closer to where we are now.

**The original operator pattern** (2016) was elegant in concept: a controller
that watches a custom resource and reconciles the cluster toward the desired
state declared in it. The implementation required deep familiarity with
client-go internals — informers, workqueues, schemes, RESTMappers. Most of
that code was identical across operators. Only the business logic differed.

**Kubebuilder and Operator SDK** solved the boilerplate problem. Code
generation, scaffolding, and controller-runtime reduced hundreds of lines
to dozens. The operator developer could focus on the reconcile function
rather than the plumbing. This was real progress.

But something subtle happened: the abstraction layer got thicker. The
operator was now a project with a Makefile, a generated directory, a
controller-runtime dependency, a test suite for framework scaffolding
before any business logic was written. The cost of entry dropped — but
it did not disappear. You still needed to know Go. You still needed to
build and push an image. You still needed a deployment manifest for your
controller.

The fundamental problem was never the programming language, or the
boilerplate, or the build pipeline.

**The fundamental problem was that Kubernetes had no permanent observer
watching what was being provisioned.** Every tool that emerged — operator
frameworks, webhook servers, policy engines — existed to compensate for
that absence.

Orkestra is that observer.

---

## The core idea

Orkestra treats operators the same way Kubernetes treats Deployments — as
a declaration of intent. You write a Katalog. You describe the CRDs you
want managed and the resources that should exist for each CR. You run
`ork run`. The operator is live.

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
spec:
  crds:
    - name: website
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
```

This is a complete operator. It creates child resources. It drift-corrects
when the CR changes. It emits Kubernetes events, exposes Prometheus metrics,
supports leader election, and handles graceful shutdown.

No Go. No code generation. No build pipeline. No separate deployment.

---

## Why this is different

Other frameworks lower the barrier to writing Go operators. Orkestra removes
the barrier entirely for the common case. That is not a reduction in
complexity — it is a different category of tool.

**Accessibility.** Platform engineers, SREs, and application teams can now
build and own operators without learning Go. The person who understands the
domain can write the operator. The institutional knowledge lives in the
Katalog, not in a Go binary that only the original author understands.

**Portability.** A Katalog is YAML. It can be versioned in Git, rendered
with Helm, promoted across environments, diffed in pull requests, and shared
with other teams like any other manifest. Operators become infrastructure
artifacts rather than software projects.

**Composability.** A Komposer composes Katalogs from files, Helm charts,
and remote URLs. Platform teams publish standard CRD definitions. Application
teams consume and override what they need. The same pattern that Helm brought
to deployment manifests now applies to operator behavior.

**Consistency.** Every CRD managed by Orkestra behaves the same way. Same
health API. Same metrics. Same dependency ordering. Same lifecycle. An
organisation running twenty Orkestra operators has twenty operators with
identical operational characteristics — not twenty systems each invented
from scratch.

**Unified observability.** When one runtime watches all your CRDs, you get
one health API, one metrics endpoint, one `ork status` command. The sprawl
of per-operator dashboards collapses into a single view.

---

## One runtime, many CRDs

A single Orkestra instance can manage any number of CRDs. Each CRD gets its
own informer, its own worker pool, its own queue depth, and its own health
endpoint — all from the same runtime.

```yaml
spec:
  crds:
    - name: website
      workers: 3
      resync: 30s
    - name: database
      workers: 2
      dependsOn: [website]
    - name: cache
      workers: 2
      dependsOn: [website, database]
```

This is the opposite of operator sprawl. One runtime replaces N operators.

---

## Dependency ordering

CRDs can declare dependencies:

```yaml
- name: application
  dependsOn:
    - project
```

Orkestra starts CRDs in topological order. Dependents wait for their
dependencies to signal readiness before their workers start. Missing CRDs
retry in the background — healthy CRDs are never blocked.

This is essential for multi-CRD systems where resources depend on each other.
The dependency graph is declared, not coded.

---

## Built‑in resources

A Kubernetes Deployment is a CRD that ships with every cluster. It has a
group (`apps`), version (`v1`), kind (`Deployment`), and plural (`deployments`).
Those four values in a Katalog entry are enough for Orkestra to watch it.

```yaml
- name: deployment-governance
  apiTypes:
    kind: Deployment
  reconciler:
    default: true
    onCreate: []   # watch only — no resources created
```

Orkestra queries the API server at startup to discover the group, version,
plural, and scope for any `kind`. The user does not need to know these values.
The cluster knows them. Orkestra asks.

This means Orkestra can watch and report on any Kubernetes resource with a
single line in a Katalog. The same observability — health endpoints, metrics,
`ork status` — applies to built-in resources and custom CRDs alike.

---

## The bigger picture

The operator pattern is the right abstraction for Kubernetes extensibility.
The requirement to implement it in Go, compiled into a binary deployed
separately for each CRD, has been a constraint of convention rather than
necessity.

When the same runtime can watch any CRD, compose definitions from any source,
and provide unified observability, the operator sprawl disappears. Operators
become data, not code. They are composed, not programmed. They are versioned,
shared, and reused like any other Kubernetes resource.

Kubernetes made infrastructure declarative.
Orkestra makes the operators that extend Kubernetes declarative.
The same principle, applied one level up.

It was always possible.
It just needed someone to build it.

---

**Next:** [Whitepaper](./declarative-operators-whitepaper.md)