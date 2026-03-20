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
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.28+-326CE5.svg)](https://kubernetes.io/)
[![Release](https://img.shields.io/github/v/release/ialexeze/orkestra)](https://github.com/ialexeze/orkestra/releases)

</div>

---

## What is Orkestra?

Orkestra is a Declarative Kubernetes operator framework. You declare what you want in a
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

```bash
curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/refs/heads/main/install.sh | bash
```

Verify:
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
curl localhost:8080/katalog/metrics               # Ready metrics
curl localhost:8080/katalog/website | jq          # CRD Info
curl localhost:8080/katalog/website/health | jq   # Health check
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

A Komposer is a meta-katalog and can compose CRD definitions from multiple sources:

```yaml
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
validated runtime configuration.

### Dependency ordering

CRDs can declare dependencies:

```yaml
- name: application
  dependsOn:
    - project        # application starts only after project is ready
```

Orkestra starts CRDs in topological order. Dependents wait for their
dependencies to signal readiness before their workers start.

---

## CLI

```bash
ork run       --katalog <path>          Start the operator runtime
ork validate  --katalog <path>          Validate a Katalog
ork template  --katalog <path>          Preview the merged, validated Katalog
ork template  --katalog <path> --graph  Show dependency graph
ork template  --katalog <path> --json   Full post-validation state as JSON
ork generate runtime --katalog <path>  Generate runtime wiring (typed CRDs only)
ork init      <name>                    Scaffold a new operator project
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
- `controller_reconcile_total` — reconcile count by result (success/error)
- `controller_reconcile_duration_seconds` — reconcile latency histogram
- `controller_queue_depth` — current workqueue depth per CRD
- `controller_workers_active` — active worker count per CRD
- `controller_resource_count` — live CR count per CRD

---

## Examples

Three examples, each building on the previous one.

| Example | What it shows | Complexity |
|---------|--------------|------------|
| [Website](./examples/website/README.md) | Hello world — Deployment + Service from a CR | ⭐ |
| [Platform Namespace](examples/platform-namespace/README.md) | Secrets, ConfigMaps, ServiceAccounts — the platform engineering pattern | ⭐⭐ |
| [Komposer](examples/komposer/README.md) | Composing Katalogs from files, Helm charts, and inline overrides | ⭐⭐⭐ |

---

## Standards

### Katalog file

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: <name>
  description: <optional>
sources:         # optional
  files: [...]
  helm:  [...]
spec:
  finalizers: [...]
  crds:
    - name: <lowercase-kebab>
      enabled: true|false
      namespaced: true|false
      workers: <int>
      resync: <duration>
      dependsOn: [<crd-name>, ...]
      apiTypes:
        group: <api-group>
        version: <version>
        kind: <Kind>
        plural: <plural>
        location: <go-import-path>   # optional — omit for dynamic mode
      reconciler:
        default: true|false
        finalizers: [...]
        onCreate:    { deployments, services, secrets, configMaps, ... }
        onReconcile: { deployments, services, secrets, configMaps, ... }
        onDelete:    { jobs, ... }
        hooks:       { location, function }     # optional Go hooks
        constructor: { location, function }     # required when default: false
      queue:
        maxQueueDepth: <int>
```
---
## Philosophy

Orkestra is built on three principles:

1. **Declarative First**  
2. **Composition Over Code**  
3. **Runtime Over Build‑Time**

Operators should be assembled, not programmed.

---

## ❤️ Community

Orkestra is built for:
- platform engineers  
- SREs  
- infrastructure teams  
- Kubernetes contributors  
- anyone who wants operators without writing operators  

Contributions, issues, and discussions are welcome.

---

## Documentation

- [Katalogs](./docs/katalog.md) — how to declare what you want
- [Komposer](./docs/komposer.md) — files, Helm, environment variables, merge rules
- [OrkestraRegistry](./docs/orkestra-registry.md) — available resource implementations
- [CLI Reference](./docs/cli.md) — all commands and flags
- [Observability](./docs/observability.md) — health API, metrics, Prometheus
- [Use Cases](./docs/use-cases.md) — explore different Orkestra use cases
- [Architecture Deep Dive](./docs/architectural-deep-dive.md) —  How does Orkestra work under the hood?
- [Component Deep Dive](./docs/component-deep-dive.md) — What does each part of Orkestra do?
- [Extending OrKestra](./docs/extending-orkestra.md) — How do I build on top of Orkestra?
- [Startup & Shutdown](./docs/startup-shutdown.md) — What happens when Orkestra starts and stops?

---

## License

[MIT](LICENSE)