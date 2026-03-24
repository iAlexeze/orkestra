# Declarative Operators: A New Model for Kubernetes Extensibility

*Orkestra Project — March 2026*

---

## Abstract

Kubernetes operators encode domain knowledge as reconciliation logic. Every major operator framework to date requires this logic to be written in Go, compiled into a binary, and deployed as a separate long-running process. The result is a pattern that is increasingly expensive to operate — each new CRD brings a new binary, a new deployment, a new set of metrics, a new health story.

This paper argues that the operator itself can be declarative. We describe Orkestra — a runtime for declarative operators that watches CRDs and reconciles according to YAML templates. The same runtime that reconciles custom resources can watch built-in Kubernetes resources, compose operator definitions from multiple sources, and provide unified observability. The result is a model where operators are data, not code.

---

## 1. The evolution of the operator pattern

### 1.1 The original model

The operator pattern, introduced in 2016, proposed encoding operational knowledge as a reconciliation loop: a controller that watches a custom resource and continuously drives the cluster toward the desired state declared in it. The concept was sound. The implementation required intimate familiarity with client-go internals — informers, workqueues, REST mappers, schemes. Most of that implementation was identical across operators. The business logic was a small fraction of the total code.

### 1.2 Frameworks reduce boilerplate

Kubebuilder and Operator SDK addressed this by generating the plumbing. Scaffolding commands produced a working controller skeleton. controller-runtime wrapped client-go into a higher-level interface. The operator developer could focus on the reconcile function. The cost of entry dropped meaningfully.

The cost did not reach zero. The generated project still required Go, a build pipeline, an image registry, and a deployment manifest. Adding a new CRD meant adding a new type, running code generation, rebuilding the binary, pushing the image, and rolling the deployment. The development loop was compressed but not eliminated.

### 1.3 The operator sprawl problem

As organizations adopted operators, the operational overhead multiplied. A typical platform runs operators for Prometheus, Cert Manager, Istio, External Secrets, Crossplane providers, and internal CRDs — each with its own binary, its own deployment, its own RBAC, its own metrics endpoint, its own upgrade cadence. Each operator consumes memory and CPU. Each operator duplicates the same informer logic, the same workqueue, the same leader election code.

The problem is not that operators are heavy. The problem is that the operator pattern was never designed to be shared across many CRDs.

---

## 2. The Orkestra model

### 2.1 The operator as a declaration

Orkestra introduces two document kinds: **Katalog** and **Komposer**.

A Katalog declares one or more CRDs — their API types and how they should be reconciled. A Komposer composes Katalogs from multiple sources into one runtime.

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
          services:
            - port: "80"
              targetPort: "{{ .spec.port }}"
              reconcile: true
```

This declaration is the complete operator. Apply the `Website` CRD to the cluster, run `ork run --katalog katalog.yaml`, and:

- Every `Website` CR is watched by an informer
- A Deployment and Service are created for each CR and kept in sync
- Deletion cascades via owner references
- Finalizers ensure cleanup completes before the CR is removed
- A health API, Prometheus metrics, and Kubernetes events are provided automatically

No Programming language. No code generation. No separate deployment. No duplicate operator binary.

### 2.2 One runtime, many CRDs

A single Orkestra instance can manage any number of CRDs. Each CRD gets its own informer, its own worker pool, its own queue depth, and its own health endpoint — all from the same runtime.

```yaml
spec:
  crds:
    - name: website
      workers: 3
      resync: 30s
      # ... templates
    - name: database
      workers: 2
      dependsOn: [website]
      # ... templates
    - name: cache
      workers: 2
      dependsOn: [website, database]
```

This is the opposite of operator sprawl. One runtime replaces N operators.

### 2.3 Dependency ordering

CRDs can declare dependencies:

```yaml
- name: application
  dependsOn:
    - project
```

Orkestra starts CRDs in topological order. Dependents wait for their dependencies to signal readiness before their workers start. Missing CRDs retry in the background — healthy CRDs are never blocked.

This is essential for multi-CRD systems where resources depend on each other. The dependency graph is declared, not coded.

### 2.4 Built-in Kubernetes resources

A Kubernetes Deployment is a CRD that ships with every cluster. It has a group (`apps`), version (`v1`), kind (`Deployment`), and plural (`deployments`). Those four values in a Katalog entry are enough for Orkestra to watch it.

```yaml
- name: deployment-governance
  apiTypes:
    kind: Deployment
  reconciler:
    default: true
    onCreate: []
```

Orkestra queries the API server at startup to discover the group, version, plural, and scope for any `kind`. The user does not need to know these values. The cluster knows them. Orkestra asks.

This capability means Orkestra can watch and report on any Kubernetes resource with a single line in a Katalog. The same observability — health endpoints, metrics, `ork status` — applies to built-in resources and custom CRDs alike.

---

## 3. Composition at scale

### 3.1 The Komposer model

A Komposer resolves CRD definitions from multiple sources — files, Helm charts, remote URLs — into one validated runtime configuration. Sources are merged by CRD name. Inline `spec.crds` on a Komposer override source definitions.

```yaml
sources:
  files:
    - ./katalogs/project.yaml
    - https://platform.myorg.io/crds/katalog.yaml
    - url: https://private.myorg.io/crds/secure-katalog.yaml
      auth:
        type: bearer
        fromEnv: PLATFORM_KATALOG_TOKEN
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 2.1.0
```

### 3.2 Multi-team ownership

Each team owns their Katalog. The platform Komposer composes them. Environment-specific overrides are declared inline. The same Katalog runs in development with `workers: 2` and in production with `workers: 8` — the override is declared, not hardcoded.

This is the pattern that Helm brought to deployment configuration, applied to operator behavior. Platform teams publish operator definitions. Application teams consume and selectively override. The inheritance is explicit, auditable, and composable.

---

## 4. Observability

### 4.1 Built-in health API

Every CRD managed by Orkestra automatically exposes:

```
GET /katalog                 All CRDs — health, config, dependency graph
GET /katalog/{crd}           Single CRD — config, stats, reconciler info
GET /katalog/{crd}/health    Single CRD health — 200 healthy / 503 degraded
```

### 4.2 Built-in metrics

Five Prometheus metrics, all per-CRD:

| Metric | Type | Description |
|--------|------|-------------|
| `controller_reconcile_total` | Counter | Reconcile count by result (success/error) |
| `controller_reconcile_duration_seconds` | Histogram | Reconcile latency per CRD |
| `controller_queue_depth` | Gauge | Current workqueue depth per CRD |
| `controller_workers_active` | Gauge | Active worker count per CRD |
| `controller_resource_count` | Gauge | Live CR count per CRD |

### 4.3 CLI status

`ork status` shows the state of every managed CRD in a single view:

```
Orkestra Operator Status
Operator:            platform-operator
Health:              ● healthy
CRDs:                5 total, 5 enabled

CRD              WORKERS   QUEUE     HEALTH       RECONCILES   ERR%   RESOURCES
website             3/3       0        ●        1,247        0.0%   3
database            2/2       0        ●        412          0.0%   6
cache               2/2       3        ●        8,891        0.2%   12
```

This is the unified observability that the operator sprawl makes impossible.

---

## 5. The OrkestraRegistry

### 5.1 The standard library of operators

The OrkestraRegistry provides production-ready implementations for Deployments, Services, Secrets, ConfigMaps, Jobs, and CronJobs. Every implementation handles create, update, delete, owner references, and idempotency.

Users declare what they want. The registry handles how.

### 5.2 The future: a shared ecosystem

The registry is the foundation for a global ecosystem of reusable operator patterns. A registry entry looks like this:

```yaml
# registry entry: postgres@v14
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: postgres
crds:
  - name: postgres
    group: postgres.io
    version: v1
    kind: Postgres
    plural: postgreses
templates:
  postgres:
    onCreate:
      deployments:
        - image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
```

A Komposer imports it:

```yaml
sources:
  registry:
    - postgres@v14
    - monitoring@v2
    - backup@v1
```

The same pattern that made package management work for applications now works for operators. The operator sprawl collapses into a registry of reusable patterns.

---

## 6. The Kubernetes-native future

### 6.1 What the architecture implies

Every capability in Orkestra is a declaration interpreted at runtime. The composition model — Katalog and Komposer as first-class YAML documents — makes operator distribution work like package management.

The logical conclusion is that this observer belongs inside Kubernetes, not outside it.

### 6.2 A native meta-controller

If Katalog and Komposer became Kubernetes-native resource kinds, registered by the cluster itself, the Orkestra runtime could run as a core controller inside `kube-controller-manager`. Every cluster would have an operator runtime without installation. Platform teams would write Katalogs and Kubernetes would manage them.

The CRD definition provides the schema at cluster apply time. The Katalog entry declares Group, Version, Kind, and Plural — the same information the cluster already has. Orkestra reads it from the discovery API. Nothing is duplicated.

### 6.3 The path there

This is not immediate. It is the direction. The path runs through production adoption, through CNCF Sandbox, through a Kubernetes Enhancement Proposal, through alpha and beta behind a feature gate, to a future release where `kubectl apply -f my-katalog.yaml` is the complete interaction with an operator runtime that ships with every cluster.

Every Katalog written today is evidence for the proposal. Every platform team that simplifies their operator surface is evidence that the direction is right.

The solution will speak for itself.

---

## 7. What this replaces

| Traditional | Orkestra equivalent |
|-------------|---------------------|
| Go operator binary | Katalog YAML |
| Kubebuilder scaffolding | `ork init` |
| One deployment per CRD | One runtime for all CRDs |
| Separate health endpoints | Unified health API |
| Per-operator metrics | Unified per-CRD metrics |
| Manual dependency management | Declared `dependsOn` |
| Helm chart per operator | Komposer |
| Operator sprawl | One runtime |

---

## 8. Conclusion

The operator pattern is the right abstraction for Kubernetes extensibility. The requirement to implement it in Go, compiled into a binary deployed separately for each CRD, has been a constraint of convention rather than necessity.

When the same runtime can watch any CRD, compose definitions from any source, and provide unified observability, the operator sprawl disappears. Operators become data, not code. They are composed, not programmed. They are versioned, shared, and reused like any other Kubernetes resource.

Kubernetes made infrastructure declarative.
Orkestra makes the operators that extend Kubernetes declarative.
The same principle, applied one level up.
It was always possible.
It just needed someone to build it.

---

*Orkestra — Declarative Operators for Kubernetes*
*March 2026*
*https://github.com/iAlexeze/orkestra*


- **Next:** [Production Metrics](./metrics-analysis.md)