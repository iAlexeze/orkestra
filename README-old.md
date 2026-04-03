<div align="center">
  <img src="./docs/assets/logo.png" alt="Orkestra" height="96" />
  <h1>Orkestra</h1>
  <p><strong>The Kubernetes operator runtime that needs no programming language.</strong></p>
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

Building a Kubernetes operator means writing Go — informers, workqueues, reconcile loops, code generation, Dockerfiles, Helm charts. Then doing it again for the next CRD. And the next.

Orkestra removes that entirely.

You write a CRD and a Katalog. Orkestra runs the operator.

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: website-operator
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
          services:
            - port: "80"
              targetPort: "{{ .spec.port }}"
              reconcile: true
```

```bash
ork run --katalog katalog.yaml
kubectl apply -f website.yaml
```

Orkestra creates the Deployment and Service, sets owner references, emits Kubernetes events, exposes a health endpoint, writes status, and corrects drift — without a single line of Go.

**Your CRD is enough. The rest is just a Katalog.**

---

## Getting started

```bash
# Install
brew install iAlexeze/tap/ork
# or
curl -sSL https://raw.githubusercontent.com/orkestra-sh/orkestra/main/install.sh | bash

# Scaffold
ork init my-operator && cd my-operator

# Run
kubectl apply -f examples/website/website-crd.yaml
ork run --katalog examples/website/website-katalog.yaml

# Apply a CR
kubectl apply -f examples/website/website-cr.yaml

# Watch
ork status
```

---

## What every Katalog entry gets

Every CRD declared in a Katalog receives a complete, isolated operator stack — for free:

| | |
|---|---|
| Informer | Watches your exact GVK and API version |
| Workqueue | Deduplicated, rate-limited, per-CRD |
| Worker pool | Configurable concurrency, isolated from other CRDs |
| Drift correction | `reconcile: true` re-applies desired state on every cycle |
| Owner references | Child resources cascade-deleted with the CR |
| Finalizer management | CRs protected from dirty deletion |
| Kubernetes events | Every reconcile emits a traceable event |
| Leader election | Warm-cache failover in under 15 seconds |
| Status | `Ready` condition + declarative status fields |
| Health API | `/katalog/{crd}/health`, `/katalog/{crd}`, `/metrics` |
| Prometheus metrics | Reconcile totals, queue depth, error rate — per CRD |

Fifteen CRDs. One Orkestra process. ~47 MB.

---

## Admission policy

Validation and mutation rules live in the Katalog. No webhook server. No separate TLS. No separate deployment.

```yaml
validation:
  - field: spec.image
    prefix: "myorg/"
    message: "images must come from the internal registry"
    action: deny

mutation:
  - field: spec.replicas
    default: "2"
```

With `ENABLE_ADMISSION_WEBHOOK=true`, these intercept `kubectl apply` synchronously. Without webhooks, they run on every reconcile cycle. One declaration. Two enforcement points.

---

## Declarative status

Status fields are declared in the Katalog and resolved from the live CR and its child resources after every successful reconcile:

```yaml
status:
  fields:
    - path: phase
      value: "Running"
    - path: readyReplicas
      value: "{{ .children.deployment.status.readyReplicas }}"
    - path: endpoint
      value: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
```

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
  phase: Running
  readyReplicas: "2"
  endpoint: my-site.default.svc.cluster.local
```

---

## Multi-version CRDs

Zero conversion code. No separate webhook deployment.

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

**In production:** 62 conversions. 0 failures. 0.5 ms average latency.

---

## Composition

The Komposer pulls Katalogs from files, Helm charts, Git, and OCI registries into one runtime:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: platform
sources:
  registry:
    - url: ghcr.io/konduktor-io/orkestra-registry/postgres@v14
      oci: true
  files:
    - ./katalogs/website.yaml
spec:
  crds:
    - name: postgres
      workers: 8
```

---

## When you do need Go

For external API calls, complex conditionals, or type-safe struct access — write a hook:

```go
func WebsiteHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*apiv1.Website]{
        OnReconcile: func(ctx context.Context, obj *apiv1.Website) error {
            // obj.Spec.Image — type-safe
            // call external APIs
            // use OrkestraRegistry for Kubernetes resources
        },
    }
}
```

```yaml
reconciler:
  hooks:
    location: github.com/myorg/hooks
    function: WebsiteHooks
```

Orkestra still provides the informer, queue, workers, finalizers, events, metrics, and status. You provide only the logic that genuinely requires code.

---

## By the numbers

| | Traditional | Orkestra |
|---|---|---|
| First working operator | 3–6 weeks | < 1 hour |
| Memory for 15 CRDs | 750 MB–3 GB | ~47 MB |
| Conversion latency | 2–5 ms (external webhook) | 0.5 ms (in-process) |
| Admission policy | 1 week | One Katalog rule |
| Deployment manifests | 15 | 1 |
| On-call runbooks | 15 | 1 |

---

## Safety

- **`safeReconcile`** — panics in any CRD's reconciler are caught and logged. Other CRDs are unaffected.
- **Leader election** — warm-cache failover in under 15 seconds on leader failure.
- **Idempotent operations** — every reconcile is safe to retry. No duplicates.
- **Graceful shutdown** — in-flight reconciles drain before the process exits.

→ [Trust and Failure Model](https://orkestra.readthedocs.io/en/latest/concepts/trust-and-failure-model)

---

## Documentation

| | |
|---|---|
| [Getting Started](https://orkestra.readthedocs.io/en/latest/getting-started) | First operator in under an hour |
| [Katalog Schema](https://orkestra.readthedocs.io/en/latest/reference/katalog-schema) | Complete field reference |
| [Concepts](https://orkestra.readthedocs.io/en/latest/concepts) | Architecture, patterns, philosophy |
| [Examples](./examples/) | Progressive examples from beginner to advanced |
| [Technical Docs](https://orkestra.readthedocs.io/en/latest/technical-docs) | Internals and architecture |
| [Papers](https://orkestra.readthedocs.io/en/latest/papers) | The case for declarative operators |

---

## Community

- [GitHub Issues](https://github.com/orkestra-sh/orkestra/issues) — bugs and feature requests
- [GitHub Discussions](https://github.com/orkestra-sh/orkestra/discussions) — questions and ideas
- [Contributing](./CONTRIBUTING.md) — how to get involved

---

[MIT License](LICENSE)