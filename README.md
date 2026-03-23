<div align="center">

```
   ___       _              _
  / _ \ _  _| |___ _ _  ___| |_ _ _ __ _
 | (_) | || | / -_) ' \/ -_)  _| '_/ _` |
  \___/ \_,_|_\___|_||_\___|\__|_| \__,_|
          O R K E S T R A
```

**CRDs in. Operators out.**

**The Kubernetes operator framework that needs no Programming Language.**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://golang.org/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.28+-326CE5.svg)](https://kubernetes.io/)
[![Release](https://img.shields.io/github/v/release/iAlexeze/orkestra)](https://github.com/iAlexeze/orkestra/releases)

</div>

---

## What is Orkestra?

Orkestra is a declarative Kubernetes operator framework. You declare what you want in a
YAML file — a **Katalog** — and Orkestra manages the full lifecycle of your
CRDs: create, reconcile, drift-correct, and delete.

No Programming language. No code generation. No controller boilerplate.

```yaml
# katalog.yaml — a complete operator in one file
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: website-katalog
spec:
  crds:
    - name: website
      enabled: true
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

That is the entire operator. Apply a `Website` CR and Orkestra creates the
Deployment and Service. Change a field in the CR and Orkestra reconciles.
Delete the CR and Orkestra cleans up.

> **Orkestra turns CRDs into operators — no programming language required.
> If you have a CRD, you already have everything you need.
> The rest is just a Katalog.**
---

## The Orkestra Model

Here is the entire mental model of Orkestra in one diagram:

```mermaid
flowchart LR
 subgraph Input["User Input"]
        A[("Your CRD<br>(YAML schema)")]
        B[("Katalog<br>(YAML logic)")]
  end
 subgraph Output["Orkestra"]
        C[("Orkestra Runtime")]
        D[("OrkestraRegistry")]
  end
    A L_A_C_0@-- schema defines what --> C
    B L_B_C_0@-- logic defines how --> C
    C L_C_D_0@-- uses --> D
    D L_D_C_0@-- provides implementations --> C
    C L_C_K8s_0@-- manages --> K8s["Kubernetes API"]

    style A fill:transparent,stroke:#333,stroke-width:2px
    style B fill:transparent,stroke:#333,stroke-width:2px
    style C fill:#FF6D00,stroke:#333,stroke-width:4px,color:#FFFFFF
    style D fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF
    style K8s fill:#00C853,stroke:#333,stroke-width:2px,color:#FFFFFF

    L_A_C_0@{ animation: fast } 
    L_B_C_0@{ animation: fast } 
    L_C_D_0@{ animation: fast } 
    L_D_C_0@{ animation: fast } 
    L_C_K8s_0@{ animation: fast }
```

Or in words:

> **CRD → Katalog → Operator**  
> If Kubernetes can store it, Orkestra can reconcile it.

---

## Install

### macOS (Homebrew)

```bash
brew tap iAlexeze/tap
brew install ork
```

### Linux / macOS (curl)

```bash
curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/main/install.sh | bash
```

### Options

```bash
# Review before running
curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/main/install.sh -o install.sh
less install.sh
bash install.sh

# Pin to a specific version
curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/main/install.sh | ORK_VERSION=v0.1.1 bash

# Install to a custom directory
curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/main/install.sh | ORK_INSTALL_DIR=~/.local/bin bash
```

### Verify the binary (recommended)

Every release is GPG-signed. To verify the binary before running it:

```bash
# Import the Orkestra public key (one time)
curl -sSL https://github.com/iAlexeze/orkestra/releases/download/v0.1.1/orkestra-public-key.asc | gpg --import

# Download the binary and its signature
curl -sSLO https://github.com/iAlexeze/orkestra/releases/download/v0.1.1/ork_linux_amd64.tar.gz
curl -sSLO https://github.com/iAlexeze/orkestra/releases/download/v0.1.1/ork_linux_amd64.tar.gz.asc

# Verify
gpg --verify ork_linux_amd64.tar.gz.asc ork_linux_amd64.tar.gz
# gpg: Good signature from "Orkestra Releases <releases@orkestra.io>"
```

### Confirm installation

```bash
ork version
```

---

## Quick Start

Declare what you want in a Katalog and run an operator with Orkestra in just a few minutes.

---

## Requirements

You only need two things:

- **A Kubernetes cluster** (1.28+).  
  Works with [kind](https://kind.sigs.k8s.io/), [minikube](https://minikube.sigs.k8s.io/), [k3s](https://k3s.io/), or a managed cluster ([EKS](https://aws.amazon.com/eks/), [GKE](https://cloud.google.com/kubernetes-engine/), [AKS](https://azure.microsoft.com/en-us/products//kubernetes-service/)).

- **The [ork](#install) CLI** – installed via Homebrew or the install script.

Orkestra automatically discovers your cluster from your kubeconfig — no extra setup required.

---

## 1. Create your operator

```bash
ork init my-operator
cd my-operator
```

This scaffolds a clean operator workspace with examples and a ready‑to‑run Katalog.

---

## 2. Connect to your cluster

Copy the example environment file:

```bash
cp .env.example .env
```

Most users don’t need to change anything — Orkestra will pick up your kubeconfig automatically.

---

## 3. Install the CRD your operator manages

For the Website example:

```bash
kubectl apply -f examples/website/website-crd.yaml
```

This tells Kubernetes what a `Website` resource looks like.

---

## 4. Run Orkestra

Start the operator locally:

```bash
ork run --katalog examples/website/website-katalog.yaml
```

Use `--debug` to see every reconcile, event, and template resolution.

---

## 5. Apply a Website resource

In another terminal:

```bash
kubectl apply -f examples/website/website-cr.yaml
```

Orkestra immediately detects the new CR and begins reconciling.

---

## 6. Watch Orkestra work

```bash
ork status -w
kubectl get deployments
kubectl get services
```

Or explore the built‑in endpoints:

```bash
curl localhost:8080/katalog/website/health | jq
curl localhost:8080/katalog/website | jq
curl localhost:8080/metrics
```

You’ll see the operator’s health, metrics, and full runtime state.

---

## What just happened?

When you applied your CR:

1. Kubernetes notified Orkestra  
2. Orkestra queued the reconcile  
3. The CR was loaded from the informer cache  
4. Finalizers were added  
5. Mutation and validation ran  
6. Templates were resolved  
7. Deployments and Services were created  
8. Drift correction was enabled  
9. Metrics and events were emitted  

All from a single YAML file.

---

## How it works

### The Katalog

The Katalog is the single source of truth. It declares what CRDs Orkestra
manages and what resources to create for each one.

```yaml
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
            - image: "{{ .spec.image }}"   # resolved from the CR at reconcile time
              replicas: "{{ .spec.replicas }}"
              reconcile: true              # also apply as drift correction
```

Field values support Go `text/template` expressions evaluated against the live
CR object. Static values are used as-is. Dynamic values reference CR fields.

### The OrkestraRegistry

The OrkestraRegistry is the operator standard library — a growing ecosystem of reusable, versioned, production‑ready operator behaviors. It provides resource implementations — `Deployment`,
`Service`, `Secret`, `ConfigMap`, `ServiceAccount`, `Job`, `CronJob`.
Every implementation handles create, update, delete, owner references,
and idempotency. You declare what you want. The registry handles the how.

This is the foundation for a future where:

> No one writes reconcilers anymore, and Operators are composed, not coded.

_Write it once; Community maintains and uses forever._

### Komposer

A Komposer composes CRD definitions from multiple sources into one runtime.
Where a Katalog declares CRDs, a Komposer declares where to find them:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: platform-komposer
sources:
  files:
    - ./katalogs/project.yaml
    - https://raw.github.com/myorg/crds/main/katalog.yaml
    - $REMOTE_KATALOG_URL
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 1.2.0
```

The merger resolves all sources, deduplicates by CRD name, and produces one
validated runtime configuration. Inline `spec.crds` on a Komposer override
any source definition with the same name — use this for environment-specific
adjustments without forking the source.

### Dependency ordering

CRDs can declare dependencies:

```yaml
- name: application
  dependsOn:
    - project        # application starts only after project is ready
```

Orkestra starts CRDs in topological order. Dependents wait for their
dependencies to signal readiness before their workers start. Missing CRDs
retry in the background — healthy CRDs are never blocked.

---

## CLI

```bash
ork init      <name>                    Scaffold a new operator project
ork validate  --katalog <path>          Validate a Katalog or Komposer
ork template  --katalog <path>          Preview the merged, validated Katalog
ork template  --katalog <path> --graph  Show dependency graph
ork template  --katalog <path> --json   Full post-validation state as JSON
ork generate runtime --katalog <path>  Generate runtime wiring (typed CRDs only)
ork run       --katalog <path>          Start the operator runtime
ork version                             Print version information
ork status                              Displays status of CRDs, workers, errors, reconciles and queue depth
```

---

## Observability

Every Orkestra operator exposes built-in endpoints with no configuration:

```bash
GET /health                 Orkestra Liveness probe
GET /ready                  Orkestra Readiness probe
GET /metrics                 Prometheus metrics
GET /katalog                 All CRDs — health, config, dependency graph
GET /katalog/{crd}           Single CRD — config, stats, reconciler info
GET /katalog/{crd}/health    Single CRD health — 200 healthy / 503 degraded
```

**Metrics:**

| Metric | Type | Description |
|--------|------|-------------|
| `controller_reconcile_total` | Counter | Reconcile count by result (success/error) |
| `controller_reconcile_duration_seconds` | Histogram | Reconcile latency per CRD |
| `controller_queue_depth` | Gauge | Current workqueue depth per CRD |
| `controller_workers_active` | Gauge | Active worker count per CRD |
| `controller_resource_count` | Gauge | Live CR count per CRD |
| `controller_crd_activation_latency_seconds` | Histogram | CRD activation latency |
| `controller_crd_activation_total` | Counter | CRD activation count |

All metrics use the full GVK string as the `crd` label.

---

## Examples

Three examples, each building on the previous one.

| Example | What it shows | Complexity |
|---------|--------------|------------|
| [Website](./examples/website/README.md) | Hello world — Deployment + Service from a CR | ⭐ |
| [Platform Namespace](./examples/platform-namespace/README.md) | Secrets, ConfigMaps, ServiceAccounts — the platform engineering pattern | ⭐⭐ |
| [Komposer](./examples/komposer/README.md) | Composing Katalogs from files, Helm charts, and inline overrides | ⭐⭐⭐ |

---

## Philosophy

Orkestra is built on three principles:

**Declarative first.** If Kubernetes can express it declaratively, Orkestra should too.

**Composition over code.** Operators should be assembled from declarations,
not programmed from scratch.

**Runtime over build-time.** Behavior should be interpreted at runtime, not
baked into binaries.

---

## What Orkestra is NOT

- ❌ Not a replacement for Kubernetes  
- ❌ Not a DSL  
- ❌ Not a templating engine  
- ❌ Not a webhook server  
- ❌ Not a controller framework  
- ❌ Not a policy engine  
- ❌ Not a code generator  

Orkestra is a **runtime** — the missing trusted observer Kubernetes never had.

---

## Documentation

Start with the core concepts:

| Document | Description |
|----------|-------------|
| [Katalog](./docs/katalog.md) | How to declare operator behavior |
| [Komposer](./docs/komposer.md) | Compose multiple Katalogs and sources |
| [CLI Reference](./docs/cli.md) | Commands, flags, and usage |

For all other documentation — including architecture, OrkestraRegistry, metrics, health subsystem, dependency model, extending Orkestra, and more — see:

👉 **[Full Documentation Index](./docs/README.md)**

---

## Publications

| Document | Description |
|----------|-------------|
| [Why Orkestra](./publications/why-orkestra.md) | The case for declarative operators |
| [Declarative Operators](./publications/declarative-operators-whitepaper.md) | Technical whitepaper |
| [Your CRD is Enough](./publications/your-crd-is-enough.md) | You already have all you need to run an operator |
| [Metrics Analysis](./publications/metrics-analysis.md) | See Orkestra metrics managing **170+** CRDs |


---

## Community

Orkestra is built for platform engineers, SREs, infrastructure teams, and
anyone who wants operators without writing operators.

- [GitHub Issues](https://github.com/iAlexeze/orkestra/issues)
- [Discussions](https://github.com/iAlexeze/orkestra/discussions)
- Kubernetes Slack — `#orkestra` _(planned)_

Contributions, bug reports, and Katalog examples are all welcome.

---

## License

[MIT](LICENSE)