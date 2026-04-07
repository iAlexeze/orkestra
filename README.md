
<div align="center">
  <img src="./docs/assets/logo.png" alt="Orkestra" height="96" />
  <h1>Orkestra</h1>

  <p><strong>CRDs in. Operators out.</strong></p>

  <p><strong>A declarative, zero-Go operator runtime for Kubernetes.</strong></p>

  <p>
    <a href="https://goreportcard.com/report/github.com/orkestra-sh/orkestra"><img src="https://goreportcard.com/badge/github.com/ialexeze/orkestra" alt="Go Report Card" /></a>
    <a href="https://github.com/orkestra-sh/orkestra/releases"><img src="https://img.shields.io/github/v/release/orkestra-sh/orkestra" alt="Release" /></a>
    <img src="https://img.shields.io/badge/Go-1.22+-00ADD8.svg" alt="Go" />
    <img src="https://img.shields.io/badge/Kubernetes-1.28+-326CE5.svg" alt="Kubernetes" />
    <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License" />
  </p>

  <p>
    <a href="https://orkestra.readthedocs.io">Documentation</a> ·
    <a href="https://orkestra.readthedocs.io/en/latest/getting-started">Getting Started</a> ·
    <a href="https://github.com/orkestra-sh/orkestra/discussions">Discussions</a>
  </p>
</div>

---

## The Problem

Building a Kubernetes operator means writing Go.

Not just Go — but informers, workqueues, reconcile loops, CRD clients, code generation, Dockerfiles, RBAC, Helm charts.

Then doing it again.
And again.
For every CRD.

Most engineers stop before they start.

---

## The Shift

**What if your CRD was already enough?**

Orkestra turns CRDs into operators — **without writing code**.

You define your API.
You declare your intent.
Orkestra runs the operator.

---

## The 10-Minute Operator

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: website-operator

spec:
  crds:
    website:
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

```bash
ork run --katalog katalog.yaml
kubectl apply -f website.yaml
```

That’s it.

No Go.
No controller-runtime.
No scaffolding.

---

## What You Get (For Free)

Every CRD becomes a full operator:

* Informer (your exact GVK)
* Workqueue (rate-limited, deduplicated, isolated from other CRDs)
* Worker pool (isolated per CRD)
* Drift correction (`reconcile: true`, isolated reconciler per CRD)
* Owner references + garbage collection
* Finalizers
* Kubernetes events
* Leader election
* Status updates
* Health API
* Prometheus metrics

**15 CRDs → 1 process → ~47 MB**

---

## Not a Framework — A Runtime

Traditional tools help you *write operators*.

Orkestra **removes the need to write them at all**.

You’re not building controllers.
You’re declaring systems.

---

## Declarative Everything

### Validation & Mutation

```yaml
validation:
  - field: spec.image
    prefix: "myorg/"
    action: deny

mutation:
  - field: spec.replicas
    default: 2
```

* Admission-time (optional)
* Reconcile-time (always)

**One rule. Continuous enforcement.**

---

### Multi-Version CRDs (No Webhooks to Build)

```yaml
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

**Production:**

* 62 conversions
* 0 failures
* ~0.5 ms latency
* 0 lines of Go

---

### Declarative Status

```yaml
status:
  fields:
    - path: readyReplicas
      value: "{{ .children.deployment.status.readyReplicas }}"
```

Your API reflects real state — automatically.

---

## Composition at Scale

Combine multiple sources into one runtime:

```yaml
kind: Komposer
sources:
  files:
    - ./katalogs/website.yaml

  registry:
    - url: ghcr.io/orkestra/registry/postgres@v14
      oci: true

spec:
  crds:
    - name: postgres
      workers: 8
```

* Git
* OCI
* Helm
* Internal APIs

**One runtime. Many domains.**

---

## Security by Design

* Minimal RBAC (generated from your Katalog)
* No wildcard permissions required
* No credentials in YAML (`fromEnv` only)
* Shared TLS for all webhooks
* Optional admission layer

```bash
ork generate rbac -k katalog.yaml -o rbac.yaml -n orkestra-system
```

---

## When You *Do* Need Code

For external APIs or complex logic not suppported currently in Orkestra, write a hook for just that:

```go
// Do this onReconcile
func WebsiteHooks() domain.AnyReconcileHooks {
  return domain.ReconcileHooks[*apiv1.Website]{
    OnReconcile: func(ctx context.Context, obj *apiv1.Website) error {
      // your logic
      return nil
    },
  }
}
```

Plug it in:

```yaml
reconciler:
  hooks:
    location: github.com/myorg/hooks
    function: WebsiteHooks
```

Everything else stays declarative.

---

## By the Numbers

|                     | Traditional      | Orkestra |
| ------------------- | ---------------- | -------- |
| First operator      | 3–6 weeks        | < 1 hour |
| Memory (15 CRDs)    | 750MB–3GB        | ~47MB    |
| Conversion          | Webhook infra    | Built-in |
| Admission           | Separate service | Built-in |
| Operators to manage | N                | 1        |
| Mental models       | N                | 1        |

---

## Philosophy

Orkestra is built on a simple idea:

> **Operators are not programs.
> They are declarations.**

* CRDs are APIs → APIs should be stable
* Reconciliation is data → not code
* Infrastructure should disappear → not expand

---

## Getting Started

```bash
brew install iAlexeze/tap/ork
# or
curl -sSL https://raw.githubusercontent.com/orkestra-sh/orkestra/main/install.sh | bash

ork init my-operator
cd my-operator

kubectl apply -f examples/website/website-crd.yaml
ork run --katalog examples/website/website-katalog.yaml

kubectl apply -f examples/website/website-cr.yaml
ork status
```

---

## Learn by Example

Structured from zero → advanced:

```
examples/
  beginner/
  intermediate/
  advanced/
```

Start here → **[Examples](./examples/README.md)**

---

## The Bigger Picture

Most engineers never build operators.

Not because they lack ideas.
Because the barrier is too high.

Orkestra removes that barrier.

**If you can write a CRD, you can build an operator.**

---

## Community

* GitHub Issues — bugs & features
* Discussions — ideas & questions
* Contributions welcome

---

## Final Thought

You already know how to describe your system.

Orkestra just runs it.

---

**CRDs in. Operators out.**
