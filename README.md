<div align="center">
  <img src="./docs/assets/fav.png" alt="Orkestra" height="96" />
  <h1>Orkestra</h1>
  <p><strong>The Kubernetes operator runtime that needs no programming language.</strong></p>
  <p>
    <a href="https://goreportcard.com/report/github.com/orkestra-sh/orkestra">
      <img src="https://goreportcard.com/badge/github.com/ialexeze/orkestra" alt="Go Report Card" />
    </a>
    <a href="https://github.com/orkestra-sh/orkestra/releases">
      <img src="https://img.shields.io/github/v/release/orkestra-sh/orkestra" alt="Release" />
    </a>
    <img src="https://img.shields.io/badge/Go-1.22+-00ADD8.svg" alt="Go Version" />
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

You have a CRD. Kubernetes stores it, validates it, and serves it. The only missing piece is something that watches it and acts on it — an operator.

Traditionally, that means weeks of Go: informers, workqueues, reconcile loops, code generation, Docker builds, Helm charts. Every operator is a software project.

Orkestra removes that entirely. You declare what the operator should do. Orkestra runs it.

```yaml
# This is a complete, production-grade operator.
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
```

Apply a `Website` CR. Orkestra creates the Deployment and Service, sets owner references, emits events, and exposes a health endpoint — without a single line of Go written.

---

## What you get for free

Every CRD declared in a Katalog receives a complete, isolated operator stack:

| What | Details |
|---|---|
| **Informer** | Watches your exact GVK and API version |
| **Workqueue** | Per-CRD, deduplicated, rate-limited |
| **Worker pool** | Configurable concurrency, isolated from other CRDs |
| **Drift correction** | `reconcile: true` on any resource — corrected on every cycle |
| **Owner references** | All child resources cascade-deleted with the CR |
| **Finalizer management** | CRs protected from dirty deletion automatically |
| **Kubernetes events** | Every reconcile emits a traceable event |
| **Leader election** | Only one instance reconciles; followers maintain warm caches |
| **Health API** | `/katalog/website/health`, `/katalog/website`, `/metrics` |
| **Prometheus metrics** | Reconcile totals, queue depth, error rate — per CRD |
| **Status management** | `Ready` condition + declarative status fields after every reconcile |

Fifteen CRDs. One Orkestra instance. ~47 MB.

---

## Admission policy without a webhook server

Validation and mutation rules live in the Katalog. No webhook server, no TLS certificate management, no separate deployment.

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

When `ENABLE_WEBHOOKS=true`, these rules intercept `kubectl apply` synchronously — rejection happens before the CR is stored. The same rules run at reconcile time for CRs that existed before the webhook was enabled. One declaration. Two enforcement points.

---

## Declarative version conversion

Multi-version CRDs with zero conversion code. No Go. No separate webhook deployment. No TLS to manage beyond the one certificate Orkestra already uses.

```yaml
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
```

**Production result:** 62 conversions. 0 failures. 0.5 ms average latency.

---

## Declarative status

Status fields are declared alongside reconcile templates and resolved from the live CR and its child resources:

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

After every successful reconcile:

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

## Composable operator definitions

The Komposer composes Katalogs from multiple sources — files, Helm charts, Git repositories, OCI registries — into one runtime:

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
      workers: 8        # override for production
```

Platform teams publish patterns. Application teams compose and override. The registry is OCI — distributed and versioned like container images.

---

## Install

```bash
# macOS / Linux (Homebrew)
brew install iAlexeze/tap/ork

# Linux / macOS (curl)
curl -sSL https://raw.githubusercontent.com/orkestra-sh/orkestra/main/install.sh | bash

# Verify
ork version
```

---

## Quick start

```bash
# Scaffold a new operator project
ork init my-operator
cd my-operator

# Install the example CRD
kubectl apply -f examples/website/website-crd.yaml

# Start the operator
ork run --katalog examples/website/website-katalog.yaml

# Apply a CR (in another terminal)
kubectl apply -f examples/website/website-cr.yaml

# Watch it reconcile
ork status
kubectl get deployments
```

---

## The numbers

| | Traditional | Orkestra |
|---|---|---|
| First working operator | 3–6 weeks | < 1 hour |
| Memory for 15 CRDs | 750 MB–3 GB | ~47 MB |
| Conversion latency | 2–5 ms (external webhook) | 0.5 ms (in-process) |
| Admission policy setup | 1 week | One Katalog rule |
| Deployment manifests | 15 (one per operator) | 1 |
| On-call runbooks | 15 (one per operator) | 1 |

---

## When you do need Go

Declarative templates cover the common case. When your operator needs to call an external API, write a typed hook:

```go
func WebsiteHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*apiv1.Website]{
        OnReconcile: func(ctx context.Context, obj *apiv1.Website) error {
            // type-safe struct access
            // call external APIs
            // use OrkestraRegistry for the Kubernetes resources
        },
    }
}
```

Declare it in the Katalog:

```yaml
reconciler:
  hooks:
    location: github.com/myorg/hooks
    function: WebsiteHooks
```

Orkestra still provides the informer, queue, workers, finalizers, events, metrics, and status. You provide only the logic that genuinely requires code.

---

## Safety

- **`safeReconcile`** — panics in any CRD's reconciler are caught and logged. Other CRDs are completely unaffected.
- **Leader election** — warm-cache failover in under 15 seconds on leader failure.
- **Idempotent operations** — every reconcile is safe to retry. No duplicates.
- **Graceful shutdown** — in-flight reconciles complete before the process exits.

Full failure model: [Trust & Failure Model](https://orkestra.readthedocs.io/en/latest/concepts/trust-and-failure-model)

---

## Documentation

| | |
|---|---|
| [Getting Started](https://orkestra.readthedocs.io/en/latest/guides/user-guide/getting-started) | Your first operator in 5 minutes |
| [Katalog Schema](https://orkestra.readthedocs.io/en/latest/reference/katalog-schema) | Complete field reference |
| [Komposer Schema](https://orkestra.readthedocs.io/en/latest/reference/komposer-schema) | Composition reference |
| [Registry](https://orkestra.readthedocs.io/en/latest/orkestra-registry/overview) | Distributing operator patterns |
| [Technical Docs](https://orkestra.readthedocs.io/en/latest/technical-docs) | Internals and architecture |
| [Whitepapers](https://orkestra.readthedocs.io/en/latest/papers) | The case for declarative operators |

---

## Community

- [GitHub Issues](https://github.com/orkestra-sh/orkestra/issues) — bugs and feature requests
- [GitHub Discussions](https://github.com/orkestra-sh/orkestra/discussions) — questions and ideas
- [Contributing](./CONTRIBUTING.md) — how to get involved

---

## License

[MIT](LICENSE)