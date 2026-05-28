# Concepts

---

## What is Orkestra?

Orkestra is a declarative operator runtime for Kubernetes. It turns CRDs into
fully functional operators without controllers, reconcilers, or conversion code.

You declare what a CRD should do — create a Deployment and a Service, apply
defaults, validate fields, convert between versions. Orkestra runs the operator.
The code you would have written does not exist.

!!! tip "The one-sentence version"
    Every operator framework before Orkestra reduced the code you write.
    Orkestra removes the need to write code at all.

See [Your CRD Is Enough](/blog/your-crd-is-enough/) for the full picture.

---

## Do I need to write Go code?

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

Go hooks are available when you need them — complex external API calls not covered by the `external:` block, complex
conditional logic not covered by the `when:` and `anyOf:` blocks. But hooks are additive. The
declarative layer handles everything else.

!!! note "When Go becomes necessary"
    The 20% of operator logic that genuinely requires code — creating a user
    inside PostgreSQL, reading another cluster's state
    — is handled by hooks. Hooks coexist with declarative templates. You do not
    choose one or the other.

---

## How does Orkestra differ from Helm or Kustomize?

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

## What is a Katalog?

A Katalog is a YAML document that declares how Orkestra should manage one or more
CRDs. It is not a Kubernetes CRD itself — it is a file.

```yaml
apiVersion: orkestra.orkspace.io/v1
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
        default: true
        onCreate:
          deployments:
            - image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              reconcile: true
```

!!! note "Why Katalog is not a CRD"
    Orkestra deliberately keeps Katalog and Komposer as plain YAML files, not
    Kubernetes CRDs. See [Why Katalog and Komposer Are Not CRDs](./05-why-not-crds.md)
    for the full reasoning.

See the [Katalog Schema](../reference/schema/02-katalog/01-katalog.md) for all available fields.

---

## What is a Komposer?

A Komposer composes multiple Katalogs from different sources into one unified
runtime configuration.

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: platform-komposer

imports:
  registry:
    - url: ghcr.io/orkspace/orkestra-registry/postgres@v14
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

See the [Komposer Schema](../reference/schema/03-komposer/index.md) for all options.

---

## What is the OrkestraRegistry?

The OrkestraRegistry is two things:

**1. The internal resource library** (`pkg/orkestra-registry/`) — Go implementations
of Create, Update, Delete, and Resolve for every common Kubernetes resource type:
Deployments, Services, Secrets, ConfigMaps, Jobs, CronJobs, Pods, ServiceAccounts.
These are called by the reconciler when it processes declarative templates.

**2. The public pattern registry** (`orkspace/orkestra-registry`) — versioned
operator patterns distributed as OCI artifacts. Pull a Postgres operator pattern
with one line in a Komposer. No binary. No deployment. Just a Katalog.

!!! tip "The npm analogy"
    The OrkestraRegistry is Orkestra's package manager for operator behavior.
    Patterns are versioned, composable, and overridable. You import them like
    dependencies, not like binaries.

---

## What is the super-operator model?

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

!!! note "The kube-controller-manager analogy"
    This is exactly how `kube-controller-manager` works. It runs the Deployment
    controller, the ReplicaSet controller, the Job controller, and dozens of others
    in one process. Each controller is isolated — they share only the infrastructure.
    Orkestra applies this proven model to custom resources.

---

## Does Orkestra support multi-version CRDs?

Yes — with zero conversion code.

Each CRD version is a separate entry in the Katalog with its own complete operator
stack. Each entry's informer watches its specific GVK — the API server converts
objects to the requested version before delivering them. Conversion rules are
declared alongside reconcile templates:

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

!!! note "No separate webhook deployment"
    Conversion runs on Orkestra Gateway — the same server that serves
    `/validate` and `/mutate`. No separate conversion webhook binary. No separate
    TLS certificate. No separate deployment.

---

## Next

- **[Running](./02-running.md)** — setup, configuration, and operations
- **[Patterns](./03-patterns.md)** — validation, mutation, built-in kinds
- **[Ecosystem](./04-ecosystem.md)** — comparisons and the Kubernetes roadmap
