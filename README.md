<div align="center">
  <img src="./docs/assets/logo.png" alt="Orkestra" height="96" />

  <h1>Orkestra</h1>
  <p><strong>A runtime for Kubernetes operators.</strong></p>

  <p>
    <a href="https://goreportcard.com/report/github.com/orkestra-sh/orkestra"><img src="https://goreportcard.com/badge/github.com/ialexeze/orkestra" alt="Go Report Card" /></a>
    <a href="https://github.com/orkspace/orkestra/releases"><img src="https://img.shields.io/github/v/release/orkestra-sh/orkestra" alt="Release" /></a>
    <img src="https://img.shields.io/badge/Go-1.22+-00ADD8.svg" alt="Go" />
    <img src="https://img.shields.io/badge/Kubernetes-1.28+-326CE5.svg" alt="Kubernetes" />
    <img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License" />
  </p>

  <p>
    <a href="https://orkestra.readthedocs.io">Docs</a> ·
    <a href="https://orkestra.readthedocs.io/en/latest/getting-started">Quick Start</a> ·
    <a href="https://github.com/orkestra-sh/orkestra/discussions">Discussions</a>
  </p>
</div>

---

You have a CRD. Kubernetes stores it, validates it, and serves it.

The only missing piece is something that watches it and acts on it.

Traditionally, that means Go. Informers, workqueues, reconcile loops, code generation, Dockerfiles, Helm charts. A software project per operator. Most engineers never start.

Orkestra removes that entirely.

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
      operatorBox:
        onCreate:
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              reconcile: true
          services:
            - name: "{{ .metadata.name }}"
              port: 80
              targetPort: "{{ .spec.port }}"
              reconcile: true
```

```bash
ork run -k katalog.yaml
kubectl apply -f website.yaml
```

Orkestra creates the Deployment and Service, sets owner references, writes status, emits events, corrects drift, exposes metrics and a control center — without a single line of Go.

**See Control Center:**
```bash
ork control start

# → localhost:8090
```

**Your CRD is enough. The rest is just a Katalog.**

---

## What every CRD gets

Every CRD declared in a Katalog becomes a complete, isolated operator:

| | |
|---|---|
| **Informer** | Watches your exact GVK. In-memory cache. Zero API calls on read. |
| **Workqueue** | Per-CRD. Rate-limited. Deduplicated. Isolated from every other CRD. |
| **Worker pool** | Configurable. A panic in one CRD does not affect any other. |
| **Drift correction** | `reconcile: true` — desired state is enforced on every cycle. |
| **Owner references** | Child resources deleted when the CR is deleted. |
| **Finalizers** | CRs protected from dirty deletion automatically. |
| **Events** | Every reconcile is a traceable Kubernetes event. |
| **Leader election** | One active instance. Followers hold warm caches. Failover < 15s. |
| **Status** | `Ready` condition + declarative status fields after every reconcile. |
| **Health API** | `/katalog/{crd}/health`, `/katalog/{crd}/cr`, `/metrics`. |
| **Prometheus metrics** | Reconcile totals, queue depth, error rate — all per CRD. |

Fifteen CRDs. One process. ~47 MB.

---
## Getting started

```bash
# Install
brew install iAlexeze/tap/ork
# or
curl -sSL https://raw.githubusercontent.com/orkestra-sh/orkestra/main/install.sh | bash

# Initialize an operator
ork init my-operator && cd my-operator

# Run
kubectl apply -f examples/website/website-crd.yaml
ork run --katalog examples/website/website-katalog.yaml

# Apply a CR
kubectl apply -f examples/website/website-cr.yaml

# Watch live on Control Center
ork control start

# → localhost:8090
```

For production, deploy with Helm:

```bash
helm install orkestra orkestra/orkestra \
  --set katalog.configMap=my-platform-katalog \
  --namespace orkestra-system \
  --create-namespace
```

The same Katalog you ran locally is what runs in production.

---

## Validation and mutation

Rules live in the Katalog. No separate webhook server. No TLS configuration.

```yaml
validation:
  rules:
    - field: spec.image
      prefix: "myorg/"
      message: "images must come from the internal registry"
      action: deny

mutation:
  mutateFirst: true
  rules:
    - field: spec.replicas
      default: "2"
    - field: spec.port
      default: "8080"
```

With `ENABLE_ADMISSION_WEBHOOK=true`, these intercept `kubectl apply` synchronously at the API server. Without it, they run on every reconcile. One declaration. Two enforcement points.

---

## Conditional provisioning

Resources are created only when conditions are met. No if/else in Go. No custom controllers.

```yaml
operatorBox:
  default: true
  onReconcile:
    services:
      - name: "{{ .metadata.name }}-lb"
        type: LoadBalancer
        when:
          - field: spec.environment
            equals: production
    configMaps:
      - name: "{{ .metadata.name }}-debug"
        when:
          - field: spec.environment
            notEquals: production
```

The `LoadBalancer` Service exists only in production. The debug `ConfigMap` exists everywhere else. The operator responds to spec changes without redeployment.

---

## Status

```yaml
operatorBox:
  default: true
  status:
    fields:
      - path: phase
        value: "{{ ternary .spec.suspend \"Suspended\" \"Active\" }}"
      - path: endpoint
        value: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
      - path: readyReplicas
        value: "{{ .children.deployment.status.readyReplicas }}"
```

Status fields are resolved from the live CR and its children after every reconcile. No `updateStatus` calls. No diff logic. Declare what the status should contain. Orkestra writes it.

---

## Multi-version CRD conversion

When your schema evolves, Orkestra handles the conversion. No separate webhook deployment. No additional TLS. The same process that runs your operators runs the `/convert` endpoint.

```yaml
conversion:
  storageVersion: v2
  paths:
    - from: v1
      to: v2
      spec:
        schedule: "{{ cronFromMap .spec.schedule }}"

    - from: v2
      to: v1
      spec:
        schedule: "{{ cronNormalize .spec.schedule }}"
```

**In production:** 11,279 conversions. 0 failures. 0.59 ms average latency.

---

## Cross-operator IPC

Operators observe each other's state explicitly. No shared caches. No hidden coupling.

```yaml
operatorBox:
  default: true
  cross:
    - kind: managed-database
      selector:
        name: "{{ .metadata.name }}-db"
      as: db
  onReconcile:
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        env:
          DB_HOST:
            value: "{{ .cross.db.status.endpoint }}"
        when:
          - field: cross.db.status.phase
            equals: Ready
```

The `Deployment` is not created until the database CR is Ready. When it is, the endpoint is injected automatically. No polling. No coordination code.

---

## Composition

Pull Katalogs from files, Helm, Git, or OCI registries:

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
    - ./katalogs/pipeline.yaml
spec:
  crds:
    postgres:
      workers: 8
```

One command starts the entire platform.

---

## In production

| | |
|---|---|
| Live resources under management | 13,220 |
| Active operatorboxes | 3 Katalogs, 113 workers |
| Reconcile error rate | **0.0%** |
| Conversion failures | **0** |
| Memory (15 CRDs) | ~47 MB |

---

## By the numbers

| | Traditional | Orkestra |
|---|---|---|
| First operator | Days to weeks | Under 1 hour |
| Lines of Go | 400+ per operator | 0 |
| Memory (15 operators) | 750 MB – 3 GB | ~47 MB |
| Conversion webhook | Separate deployment | Built-in |
| Admission webhook | Separate deployment | Built-in |
| Deployments to manage | One per operator | One |

---

## Documentation

| | |
|---|---|
| [Getting Started](https://orkestra.readthedocs.io/en/latest/getting-started) | First operator in under an hour |
| [Katalog Reference](https://orkestra.readthedocs.io/en/latest/reference/katalog-schema) | Complete field reference |
| [Examples](./examples/) | Beginner → advanced, all verified |
| [Concepts](https://orkestra.readthedocs.io/en/latest/concepts) | Architecture and mental model |
| [Papers](https://orkestra.readthedocs.io/en/latest/papers) | The case for declarative operators |

---

## Community

[Issues](https://github.com/orkestra-sh/orkestra/issues) · [Discussions](https://github.com/orkestra-sh/orkestra/discussions) · [Contributing](./CONTRIBUTING.md)

---

Apache 2.0 — see [LICENSE](./LICENSE)