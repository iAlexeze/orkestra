<div align="center">

```
   ___       _              _
  / _ \ _  _| |___ _ _  ___| |_ _ _ __ _
 | (_) | || | / -_) ' \/ -_)  _| '_/ _` |
  \___/ \_,_|_\___|_||_\___|\__|_| \__,_|
          O R K E S T R A
```

**The Kubernetes operator framework that needs no Go.**

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

No Go. No code generation. No controller boilerplate.

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

## Quick start

```bash
# 1. Scaffold a new operator
ork init my-operator
cd my-operator

# 2. Apply the example CRD
kubectl apply -f examples/website/website-crd.yaml

# 3. Start Orkestra
ork run --katalog examples/website/website-katalog.yaml

# 4. Apply a CR — in another terminal
kubectl apply -f examples/website/website-cr.yaml

# 5. Watch Orkestra work
kubectl get deployments
kubectl get services
curl localhost:8080/katalog/website/health | jq
curl localhost:8080/katalog/website | jq
curl localhost:8080/metrics
```

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

The OrkestraRegistry provides resource implementations — `Deployment`,
`Service`, `Secret`, `ConfigMap`, `ServiceAccount`, `Job`, `CronJob`.
Every implementation handles create, update, delete, owner references,
and idempotency. You declare what you want. The registry handles the how.

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
ork init      <n>                    Scaffold a new operator project
ork validate  --katalog <path>          Validate a Katalog or Komposer
ork template  --katalog <path>          Preview the merged, validated Katalog
ork template  --katalog <path> --graph  Show dependency graph
ork template  --katalog <path> --json   Full post-validation state as JSON
ork generate runtime --katalog <path>  Generate runtime wiring (typed CRDs only)
ork run       --katalog <path>          Start the operator runtime
ork version                             Print version information
```

---

## Observability

Every Orkestra operator exposes built-in endpoints with no configuration:

```bash
GET /health                  Liveness probe
GET /ready                   Readiness probe
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

## Katalog reference

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: <n>
  description: <optional>
spec:
  finalizers: [...]             # katalog-level finalizers — inherited by all CRDs
  crds:
    - name: <lowercase-kebab>
      enabled: true|false       # default: true
      namespaced: true|false    # default: true
      workers: <int>
      resync: <duration>
      dependsOn: [<crd-name>, ...]
      apiTypes:
        group: <api-group>
        version: <version>
        kind: <Kind>
        plural: <plural>
        location: <go-import-path>  # optional — omit for dynamic mode
      reconciler:
        default: true|false         # default: true
        finalizers: [...]           # per-CRD finalizers
        onCreate:    { deployments, services, secrets, configMaps, serviceAccounts, jobs, cronJobs }
        onReconcile: { deployments, services, secrets, configMaps, cronJobs }
        onDelete:    { jobs }
        hooks:       { location, function }   # optional Go hooks
        constructor: { location, function }   # required when default: false
      queue:
        maxQueueDepth: <int>
        degradeThreshold: <int>
```

---

## Philosophy

Orkestra is built on three principles:

**Declarative first.** If Kubernetes can express it declaratively, Orkestra should too.

**Composition over code.** Operators should be assembled from declarations,
not programmed from scratch.

**Runtime over build-time.** Behavior should be interpreted at runtime, not
baked into binaries.

---

## Documentation

| Document | Description |
|----------|-------------|
| [Katalog](./docs/katalog.md) | How to declare what you want |
| [Komposer](./docs/komposer.md) | Files, Helm, environment variables, merge rules |
| [Komponents](./docs/komponents.md) | What each part of Orkestra does |
| [OrkestraRegistry](./docs/orkestra-registry.md) | Available resource implementations |
| [Templating](./docs/templating.md) | Template expressions and the resolver |
| [CLI Reference](./docs/cli.md) | All commands and flags |
| [Inspect Live CRD](./docs/inspect-live-crd.md) | CLI for inspecting all CRDs | 
| [Architecture](./docs/architecture.md) | How Orkestra works under the hood |
| [Extending Orkestra](./docs/extending.md) | Adding new CRDs and resource types |
| [Use Cases](./docs/use-cases.md) | Real-world operator patterns |
| [Deployment](./docs/deployment.md) | Step by step guide to deploy Orkestra |
| [Metrics](./docs/metrics.md) | Which metrics to look for |
| [Health Subsystem](./docs/health-subsystem.md) | How Orkestra manages its health and that of CRDs |
| [Dependency Model](./docs/dependency-model.md) | How Orkestra manages dependencies |
| [Roadmap](./ROADMAP.md) | What is coming next |

## Publish

| Document | Description |
|----------|-------------|
| [Why Orkestra](./publish/why-orkestra.md) | The case for declarative operators |
| [Declarative Operators](./publish/declarative-operators-whitepaper.md) | Technical whitepaper |

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