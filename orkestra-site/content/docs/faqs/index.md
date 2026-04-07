---
title: "Index"
weight: 18
---

# Frequently Asked Questions

---

## Core Concepts

### What is Orkestra?

Orkestra is a declarative operator runtime for Kubernetes. It turns CRDs into
fully functional operators without controllers, reconcilers, or conversion code.

You declare what a CRD should do — create a Deployment and a Service, apply
defaults, validate fields, convert between versions. Orkestra runs the operator.
The code you would have written does not exist.

{{< callout type="tip" title="The one-sentence version" >}}
Every operator framework before Orkestra reduced the code you write.
Orkestra removes the need to write code at all.
{{< /callout >}}

See [Your CRD Is Enough](../blog/your-crd-is-enough.md) for the full picture.

---

### Do I need to write Go code?

No — for the common case.

Orkestra provides these capabilities declaratively, with no Go:

- Informers watching your exact GVK and version
- Workqueue with configurable depth, backoff, and rate limiting
- Worker pool with configurable concurrency
- Drift correction (`reconcile: true` on any template resource)
- Owner references and cascade deletion
- Kubernetes event emission
- Leader election
- Health endpoints and Prometheus metrics
- Multi-version CRD conversion
- Admission-time validation and mutation

Go hooks are available when you need them — external API calls, complex
conditional logic, type-safe struct access. But hooks are additive. The
declarative layer handles everything else.

{{< callout type="note" title="When Go becomes necessary" >}}
The 20% of operator logic that genuinely requires code — creating a user
inside PostgreSQL, calling an external API, reading another cluster's state
— is handled by [hooks](../technical-docs/hooks.md). Hooks coexist with
declarative templates. You do not choose one or the other.
{{< /callout >}}

---

### How does Orkestra differ from Helm or Kustomize?

Different category entirely.

| | Helm | Kustomize | Orkestra |
|---|---|---|---|
| **What it does** | Renders templates once | Patches manifests once | Runs a continuous operator loop |
| **When it runs** | At deploy time | At deploy time | Continuously, while the cluster runs |
| **Drift correction** | No | No | Yes — corrects on every reconcile cycle |
| **Watches CRs** | No | No | Yes — every change event triggers reconcile |
| **Versioning** | Chart versions | Kustomization | Per-CRD operator stacks, declarative conversion |
| **Dependencies** | Chart dependencies | Kustomization bases | `dependsOn` ordering with ready signals |

Orkestra is an operator runtime. Helm and Kustomize are deployment tools. They
solve adjacent problems and compose naturally — you can use a Helm chart as a
Katalog source in a Komposer.

---

### What is a Katalog?

A Katalog is a YAML document that declares how Orkestra should manage one or more
CRDs. It is not a Kubernetes CRD itself — it is a file.

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
```

{{< callout type="note" title="Why Katalog is not a CRD" >}}
Orkestra deliberately keeps Katalog and Komposer as plain YAML files, not
Kubernetes CRDs. See [Why Katalog and Komposer Are Not CRDs](./why-not-crds.md)
for the full reasoning. The short version: your CRD should be the focus,
not Orkestra's management infrastructure.
{{< /callout >}}

See the [Katalog Schema](../reference/katalog-schema.md) for all available fields.

---

### What is a Komposer?

A Komposer composes multiple Katalogs from different sources into one unified
runtime configuration.

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: platform-komposer

sources:
  registry:
    - url: ghcr.io/konduktor-io/orkestra-registry/postgres@v14
      oci: true
  files:
    - ./katalogs/website.yaml
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 2.1.0

spec:
  crds:
    postgres:
      workers: 8      # override for production
```

The `spec.crds` inline block always wins on name conflict — it is the override
mechanism. Platform teams publish Katalogs; application teams compose and override.

See the [Komposer Schema](../reference/komposer-schema.md) for all options.

---

### What is the OrkestraRegistry?

The OrkestraRegistry is two things:

**1. The internal resource library** (`pkg/orkestra-registry/`) — Go implementations
of Create, Update, Delete, and Resolve for every common Kubernetes resource type:
Deployments, Services, Secrets, ConfigMaps, Jobs, CronJobs, Pods, ServiceAccounts.
These are called by the reconciler when it processes declarative templates. You never
call them directly unless you are writing hooks.

**2. The public pattern registry** (`konduktor-io/orkestra-registry`) — versioned
operator patterns distributed as OCI artifacts. Pull a Postgres operator pattern
with one line in a Komposer. No binary. No deployment. Just a Katalog.

{{< callout type="tip" title="The npm analogy" >}}
The OrkestraRegistry is Orkestra's package manager for operator behavior.
Patterns are versioned, composable, and overridable. You import them like
dependencies, not like binaries.
{{< /callout >}}

See [OrkestraRegistry Overview](../orkestra-registry/index.md) for the full picture.

---

## Running Orkestra

### Can Orkestra manage multiple CRDs?

Yes — any number. This is the point.

Each CRD in a Katalog gets its own complete, isolated operator stack:

- Dedicated informer watching its exact GVK and API version
- Dedicated workqueue with independent depth and backoff
- Dedicated worker pool — other CRDs cannot consume its workers
- Dedicated health endpoint at `/katalog/{crd}/health`
- Dedicated Prometheus metrics labeled by GVK

All of these operator stacks run inside one Orkestra process. The isolation is at
the logic level. The shared infrastructure — API server connection, informer factory,
health server, leader election — is paid once.

{{< callout type="tip" title="The economics" >}}
15 separate operator processes: ~750 MB–3 GB memory, 15 health endpoints,
15 metric schemas, 15 upgrade procedures.
{{< /callout >}}

    Orkestra managing 15 CRDs: ~50 MB memory, 1 health server, 1 metric schema,
    1 upgrade procedure.

---

### How do I start Orkestra?

Locally, for development:

```bash
ork run --katalog katalog.yaml
```

In a cluster, via Helm:

```bash
helm install orkestra charts/orkestra \
  --namespace orkestra \
  --create-namespace \
  --set konfig.katalog.path=/katalogs/katalog.yaml
```

See the [Deployment Guide](../guides/user-guide/deployment.md) for full cluster setup including
TLS, RBAC, and production tuning.

---

### What does `ork validate` do?

`ork validate` runs the complete Katalog loading sequence without starting the runtime.
It surfaces every configuration error — bad YAML, unknown kinds, circular dependencies,
missing registry files, empty pattern files — before any cluster changes are made.

```bash
ork validate --katalog katalog.yaml

✓ website
    kind: Website
    group: demo.orkestra.io / version: v1alpha1 / plural: websites
    mode: dynamic / workers: 3 / resync: 15s
    validation: 2 rules / mutation: 1 rule

✗ application
    error: circular dependency: application → namespace → application
```

{{< callout type="tip" title="Run in CI" >}}
`ork validate` exits with a non-zero code on any error. Add it to your CI
pipeline to catch Katalog errors before they reach the cluster:
```yaml
- name: Validate Katalog
  run: ork validate --katalog katalog.yaml
```
It requires no cluster connection — safe to run in any CI environment.
{{< /callout >}}

---

### Does Orkestra require cert-manager?

No. Orkestra needs TLS certificates for its HTTPS server (used by conversion
and admission webhooks) when `ENABLE_CONVERSION=true` or `ENABLE_ADMISSION_WEBHOOK=true`.
Where those certificates come from is your choice.

| Approach | Suitable for |
|---|---|
| Self-signed (via `generate-certs.sh`) | Development and testing |
| cert-manager `Certificate` resource | Production — automated renewal |
| External PKI / corporate CA | Enterprise environments with existing PKI |
| Cloud provider ACM / GCP managed certs | Cloud-native deployments |

The Helm chart includes optional cert-manager integration. Set
`certManager.enabled: true` and the chart creates a `Certificate` resource and
mounts the resulting Secret automatically.

{{< callout type="note" title="Conversion and webhooks share one certificate" >}}
`/convert`, `/validate`, and `/mutate` all run on the same HTTPS server on
`:8443` with the same TLS certificate. One certificate covers all three endpoints.
{{< /callout >}}

---

### What environment variables does Orkestra read?

| Variable | Default | Description |
|---|---|---|
| `ORK_PORT` | `8080` | HTTP server port |
| `ENABLE_CONVERSION` | `false` | Enable the `/convert` HTTPS endpoint |
| `ENABLE_ADMISSION_WEBHOOK` | `false` | Enable `/validate` and `/mutate` (requires `ENABLE_CONVERSION`) |
| `TLS_CERT` | — | Path to TLS certificate |
| `TLS_KEY` | — | Path to TLS key |
| `ORK_REGISTRY` | — | Default registry URL for `sources.registry` entries without explicit URL |
| `DEFAULT_WORKERS` | `3` | Worker count per CRD when not set in Katalog |
| `DEFAULT_RESYNC` | `15s` | Resync interval when not set in Katalog |
| `MAX_QUEUE_DEPTH` | `2000` | Max queue depth when not set in Katalog |
| `LOG_LEVEL` | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `NAMESPACE` | — | Namespace where Orkestra runs — used in webhook configurations |
| `ORKESTRA_SERVICE_NAME` | `orkestra` | Service name for webhook clientConfig |
| `CONVERSION_WINDOW` | `1000` | Rolling window size for conversion and admission latency percentiles |

---

## CRDs and Operators

### What is the super-operator model?

The super-operator model is the principle that each CRD gets a complete, isolated
operator stack while sharing the runtime infrastructure.

In traditional frameworks, one-operator-per-CRD means one binary, one deployment,
one informer factory, one leader election lease per CRD. The isolation is at the
process level — expensive.

In Orkestra, one-operator-per-CRD means one informer, one queue, one worker pool,
one reconciler per CRD — all inside a single process. The isolation is at the logic
level. The runtime infrastructure (API server connection, informer factory, health
server, leader election) is shared.

This gives you the isolation guarantee of the one-operator-per-CRD principle at a
fraction of the resource cost.

{{< callout type="note" title="The kube-controller-manager analogy" >}}
This is exactly how `kube-controller-manager` works. It runs the Deployment
controller, the ReplicaSet controller, the Job controller, and dozens of others
in one process. Each controller is isolated — they share only the infrastructure.
Orkestra applies this proven model to custom resources.
{{< /callout >}}

---

### Can Orkestra manage built-in Kubernetes resources?

Yes. `kind: Deployment`, `kind: Pod`, `kind: Service`, and 30+ other built-in
Kubernetes kinds are supported without declaring group, version, or plural —
Orkestra enriches them automatically from its internal registry:

```yaml
- name: deployment-governance
  apiTypes:
    kind: Deployment   # ← only field needed for built-in kinds
  validation:
    - field: metadata.labels.team
      operator: exists
      message: "all deployments must declare a team owner"
      action: warn
```

{{< callout type="tip" title="Governance without a separate policy engine" >}}
This is how you apply governance to Kubernetes built-in resources without
OPA, Kyverno, or a separate admission controller. Orkestra watches the resource,
validates it at reconcile time, and optionally intercepts at admission time
when `ENABLE_ADMISSION_WEBHOOK=true`.
{{< /callout >}}

Run `ork validate --katalog katalog.yaml` to see exactly what Orkestra resolves
for a kind-only declaration.

---

### Does Orkestra support multi-version CRDs?

Yes — with zero conversion code.

Each CRD version is a separate entry in the Katalog with its own complete operator
stack. Each entry's informer watches its specific GVK — the API server converts
objects to the requested version before delivering them. Conversion rules are
declared alongside reconcile templates and evaluated by the same resolver:

```yaml
- name: website-v1
  conversion:
    storageVersion: v1
    paths:
      - from: v1alpha1
        to: v1
        spec:
          image: "{{ .spec.image }}"
          seo:
            enabled: false   # v1alpha1 has no seo field — supply default
```

**Production results:** 62 conversions, 0 failures, sub-millisecond average latency.

{{< callout type="note" title="No separate webhook deployment" >}}
Conversion runs on Orkestra's own HTTPS server — the same server that serves
`/validate` and `/mutate`. No separate conversion webhook binary. No separate
TLS certificate. No separate deployment.
{{< /callout >}}

See [Versioning](../runtime-manual/concepts/versioning.md) and
[Declarative Conversion](../publications/declarative-conversion.md) for the full design.

---

## Validation and Mutation

### What is the difference between validation and mutation?

**Validation** evaluates rules against a CR and either blocks it (`action: deny`)
or surfaces an advisory (`action: warn`).

**Mutation** applies defaults and normalisations to a CR before it is stored. Fields
declared with `default:` are set only when absent. Fields declared with `override:`
are always set.

Both run at two points:

- **Admission time** — when `ENABLE_ADMISSION_WEBHOOK=true`, synchronously during `kubectl apply`
- **Reconcile time** — every reconcile cycle, regardless of webhook configuration

Declare once, enforced at both points.

{{< callout type="tip" title="Roll out rules safely" >}}
Deploy new validation rules with `action: warn` first. Observe
`controller_admission_validation_violations_total` in Prometheus to understand
which CRs would be affected. When you are confident, change to `action: deny`.
The Katalog change takes effect on the next Orkestra restart.
{{< /callout >}}

---

### Does `ENABLE_ADMISSION_WEBHOOK=true` block the API server if Orkestra is down?

No — by design. The webhook configuration uses `FailurePolicy: Ignore` by default.
If Orkestra is unreachable when the API server calls `/validate` or `/mutate`, the
operation is allowed through. Validation catches violations at reconcile time when
Orkestra restarts.

```yaml
# To change to blocking behaviour (requires high-availability Orkestra deployment):
# Set in Helm values:
webhooks:
  failurePolicy: Fail    # default: Ignore
```

{{< callout type="warning" title="Before setting Fail" >}}
`FailurePolicy: Fail` means Orkestra's availability directly gates all CR
deployments. Set it only with multiple Orkestra replicas, a PodDisruptionBudget,
and confidence that your admission rules are correct. Start with `Ignore`.
{{< /callout >}}

---

## Operations

### How do I debug a CRD in production?

A systematic diagnostic path:

```bash
# 1. Overall health
ork status

# 2. Specific CRD health — 200 or 503
curl localhost:8080/katalog/website/health

# 3. Full CRD detail — stats, active warnings, queue depth
curl localhost:8080/katalog/website | jq

# 4. Recent events on a CR
ork events website my-site

# 5. Full CR detail with events
ork describe website my-site

# 6. Prometheus metrics
curl localhost:8080/metrics | grep website

# 7. Live queue depth and worker state
ork top
```

{{< callout type="tip" title="Port-forwarding in-cluster" >}}
When Orkestra runs in a cluster, port-forward before using the CLI:
```bash
kubectl port-forward svc/orkestra 8080:8080 -n orkestra &
ork status
```
{{< /callout >}}

The most common issues:

| Symptom | Likely cause |
|---|---|
| `/health` returns 503 | CRD degraded — check reconcile error rate in `/katalog/{crd}` |
| Resource not created | `when:` condition not met — check CR fields vs condition |
| Webhook rejection | Validation rule firing — read the error message in `kubectl apply` output |
| Stuck in terminating | `onDelete` Job blocked — check Job status in the CR's namespace |
| Old field values | Reconciler not running — check if CRD is enabled and healthy |

---

### Is Orkestra safe for production?

Yes. Orkestra is designed for and demonstrated in production.

- **Leader election** — only one instance actively reconciles; followers maintain warm caches for instant failover
- **safeReconcile** — panics in any reconciler are caught; other CRDs are unaffected
- **Per-CRD failure domains** — a degraded CRD does not affect others 
- **Graceful shutdown** — in-flight reconciles complete before the process exits
- **Conversion in production** — 62 conversions, 0 failures, sub-millisecond latency

{{< callout type="note" title="Failover time" >}}
Worst-case leader failover is 15 seconds (the lease duration). In practice,
a follower on a healthy node with a warm cache starts reconciling within
16–17 seconds of a leader crash. During this window, CRs are not modified —
they are queued and processed when the new leader starts.
{{< /callout >}}

See [Trust and Failure Model](../publications/trust-and-failure-model.md) for every
failure mode, what it means, and how Orkestra handles it.

---

### What RBAC permissions does Orkestra need?

Orkestra needs a ClusterRole with:

```yaml
rules:
  # Watch and manage every CRD it is configured to handle
  - apiGroups: ["*"]
    resources: ["*"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

  # Leader election
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["get", "create", "update"]

  # Emit Kubernetes events
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["create", "patch"]

  # Webhook configuration (when ENABLE_ADMISSION_WEBHOOK=true)
  - apiGroups: ["admissionregistration.k8s.io"]
    resources:
      - validatingwebhookconfigurations
      - mutatingwebhookconfigurations
    verbs: ["get", "create", "update", "patch"]
```

The `["*"]` rule is broad. Scope it to specific API groups using `restrictedNamespaces`
and targeted ClusterRole rules when running in security-sensitive environments.

The Helm chart generates the correct ClusterRole automatically based on the Katalog
entries and enabled features.

---

## Ecosystem

### How does Orkestra compare to kro?

kro (Kubernetes Resource Orchestrator) was announced in 2024 by Google, Microsoft,
and AWS. It allows declaring `ResourceGraphDefinitions` that compose Kubernetes
resources declaratively. The core insight — operator behavior should be a declaration
— is the same insight Orkestra is built on.

The differences are significant:

| | kro | Orkestra |
|---|---|---|
| **Per-CRD isolation** | No — shared reconcile context | Yes — dedicated informer, queue, workers |
| **Multi-version CRDs** | No | Yes — declarative conversion paths |
| **Registry/distribution** | No | Yes — OCI artifacts, Artifact Hub |
| **Admission webhooks** | No | Yes — validation and mutation |
| **Health API** | No | Yes — per-CRD endpoints and Prometheus |
| **CLI** | No | Yes — `ork status`, `ork events`, `ork describe`, `ork top` |
| **Hooks for external logic** | No | Yes — typed and dynamic Go hooks |

kro is a composability layer. Orkestra is a runtime. The fact that three major cloud
providers independently arrived at the same insight validates the direction. Orkestra
is the complete version of what they were reaching for.

---

### Can Orkestra manage third-party CRDs?

Yes — any CRD that Kubernetes accepts, Orkestra can watch and reconcile. No fork, no
reverse engineering, no changes to the CRD definition needed.

```yaml
- name: prometheus
  apiTypes:
    group: monitoring.coreos.com
    version: v1
    kind: Prometheus
    plural: prometheuses
  reconciler:
    default: true
    onCreate:
      # governance, companion resources, defaults
```

This is how governance patterns work — you apply Orkestra's validation and mutation
model to CRDs you did not write and cannot modify.

---

### What is the path to Kubernetes core?

See [Orkestra: The Universal Observer That Belongs in Kubernetes Core](../publications/universal-observer-whitepaper.md)
for the full argument and roadmap.

The short version: Orkestra is building toward CNCF Sandbox, then a Kubernetes
Enhancement Proposal, then alpha/beta/GA integration into `kube-controller-manager`.
The target timeline is five years. The prerequisite is production adoption at multiple
organisations, with metrics.

The Katalog and Komposer becoming native Kubernetes kinds — `kubectl get katalogs` —
is the end state. At that point, every cluster ships with a meta-controller that
understands declarative operator definitions. Platform teams write Katalogs. Kubernetes
manages them.