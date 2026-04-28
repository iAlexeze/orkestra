# Why Orkestra?

Kubernetes has always promised declarative infrastructure.
You describe what you want. The platform makes it so.

That promise holds everywhere — until you need to extend Kubernetes itself.

The moment you need a custom resource, you leave the declarative world.
You write Go. You scaffold controllers. You wire informers and schemes.
You manage reconcile loops, retries, and backoff. You build and push images.
You maintain a project whose entire purpose is to watch another project.

Every major operator framework to date has accepted this as the cost of entry.
Kubebuilder, Operator SDK, Metacontroller — they each make the Go easier.
None of them make it unnecessary.

This creates a paradox: **to make Kubernetes more declarative, you must
write imperative code.**

Orkestra breaks that paradox.

---

## How we got here

Understanding why Orkestra exists requires understanding how the operator
pattern evolved — not to criticise what came before, but to trace honestly
how each improvement brought its own new problem.

**The original operator pattern** (2016) was elegant in concept: a controller
that watches a custom resource and continuously reconciles the cluster toward
the desired state declared in that resource. The implementation required deep
familiarity with client-go internals — informers, workqueues, schemes, REST
mappers. Most of that implementation was identical across operators. The
business logic was a small fraction of the total code.

**Kubebuilder and Operator SDK** solved the boilerplate problem. Code
generation, scaffolding, and controller-runtime reduced hundreds of lines
to dozens. The cost of entry dropped meaningfully.

The cost did not reach zero. The generated project still required Go, a
build pipeline, an image registry, and a deployment manifest. Adding a new
CRD meant a new type, new code generation, a new binary, a new deployment.

The more pressing problem was what this model produced at scale. A typical
production cluster runs operators for Prometheus, Cert Manager, Ingress,
External Secrets, internal CRDs, and more — each a separate binary, a
separate deployment, a separate failure domain, a separate metrics story,
a separate upgrade cadence. The operator became a software project. The CRD
became an afterthought.

The fundamental problem was never the boilerplate, or the language, or the
build pipeline. **The fundamental problem was that every CRD demanded its
own operator, and every operator was invented from scratch.**

Orkestra is the shared infrastructure that changes this.

---

## The reframe: your CRD is a super-operator

The Kubernetes community has long held that the right design is one operator
per CRD. This is correct — and Orkestra agrees with it completely.

The reframe is not about sharing a reconciler across CRDs. It is about what
"one operator" actually means.

In traditional operator frameworks, "one operator" means one Go binary, one
deployment, one informer, one workqueue, one set of RBAC, one set of metrics.
When you have twelve CRDs, you have twelve of everything. The per-CRD
isolation exists, but so does twelve times the operational overhead.

In Orkestra, each CRD is still its own operator — fully isolated, fully
independent. It has its own informer, its own worker pool, its own workqueue,
its own health endpoint, its own metrics, its own reconciler, its own
dependency ordering, its own failure domain. A panic in one CRD's reconciler
does not affect any other.

What is shared is not the operator logic. What is shared is the
**orchestration infrastructure** — the runtime that hosts these operators,
starts them in order, manages their lifecycle, and provides the observability
layer that makes them all look consistent.

Your CRD does not get a lightweight reconciler. It gets a full operator — with
every capability that would have taken weeks to write — for free, as a
consequence of being declared in a Katalog.

This is what we mean by the super-operator model: each CRD becomes a
complete, production-grade operator. They just share the infrastructure that
makes that possible.

---

## What this looks like in practice

You write a Katalog:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
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
      operatorBox:
        default: true
        onCreate:
          deployments:
            - image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              reconcile: true
```

You run:

```bash
ork run --katalog katalog.yaml
```

The `Website` CRD now has:

- Its own informer watching `demo.orkestra.io/v1alpha1`
- 3 dedicated workers — no other CRD can consume them
- Its own workqueue with independent backoff and depth limits
- A Deployment created and drift-corrected for every CR
- Cascade deletion via owner references
- Finalizers ensuring cleanup completes before CR removal
- A health endpoint at `/katalog/website/health`
- Five Prometheus metrics labeled by its GVK
- Warning events on every failure, Normal events on every success
- Leader election — only one pod reconciles, others hold warm caches

That is a complete, isolated, production-grade operator. You wrote twelve
lines of YAML.

---

## Multi-version CRDs: the proof

The super-operator model is most clearly demonstrated by how Orkestra handles
multi-version CRDs.

The standard approach requires writing conversion functions in Go, deploying
a webhook server, managing TLS certificates, and registering webhook
configuration objects. This is weeks of infrastructure work for what is
conceptually a field mapping problem.

In Orkestra, each version of a CRD is registered as a separate entry. Each
version gets its own complete operator stack — its own informer, its own
workers, its own reconciler. The conversion rules are declared in the Katalog
alongside the reconcile templates.

```yaml
crds:
  website-v1alpha1:
    apiTypes:
      group: demo.orkestra.io
      version: v1alpha1
      kind: Website
    operatorBox:
      default: true
      onCreate:
        deployments: [...]

  website-v1:
    apiTypes:
      group: demo.orkestra.io
      version: v1
      kind: Website
    operatorBox:
      default: true
      onCreate:
        deployments: [...]
    conversion:
      storageVersion: v1
      paths:
        - from: v1alpha1
          to: v1
          spec:
            image: "{{ .spec.image }}"
            replicas: "{{ .spec.replicas }}"
            seo:
              enabled: false
        - from: v1
          to: v1alpha1
          spec:
            image: "{{ .spec.image }}"
            replicas: "{{ .spec.replicas }}"
            theme: "default"
```

This runs in production today. Two CRD versions, each with their own full
operator stack, conversion handled declaratively, conversion metrics
automatically available:

```
orkestra_conversion_requests_total{kind="Website",from="v1alpha1",to="v1",result="success"} 14
orkestra_conversion_requests_total{kind="Website",from="v1",to="v1alpha1",result="success"} 17
```

No Go. No webhook binary. No TLS certificates to manage.

---

## The composability story

When your operator is a Katalog, it inherits all the properties of data.
It can be versioned, diffed, templated, and promoted like any other manifest.
It can be composed from multiple sources through a Komposer.

```yaml
kind: Komposer
sources:
  files:
    - https://platform.myorg.io/crds/website.yaml
    - https://platform.myorg.io/crds/database.yaml
    - $SECURITY_KATALOG_URL
  registry:
    - katalog:
        application:
          version: v2.1.0
spec:
  crds:
    # Production environment override
    application:
      workers: 8
```

Platform teams publish Katalogs. Application teams compose and override.
Environments are different configurations of the same operator definitions.
The same model that Helm brought to deployment manifests, applied to operators.

---

## The bigger picture

The operator pattern is the right abstraction for Kubernetes extensibility.
The requirement to implement it in Go, built and deployed separately for each
CRD, has been a constraint of convention rather than necessity.

When the same runtime can host any number of CRDs — each isolated, each
complete, each observable through a consistent interface — the operator
becomes something you declare, not something you build.

Kubernetes made infrastructure declarative.
Orkestra makes the operators that extend Kubernetes declarative.
The same principle, applied one level up.
It was always possible.
It just needed someone to build it.