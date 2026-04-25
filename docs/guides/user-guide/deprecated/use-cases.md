# Orkestra Use Cases

<!-- Orkestra is a declarative operator runtime. Every operator is a Katalog —
a YAML file that declares what CRDs to manage and how. This page shows what
becomes possible when your operator is a file rather than a codebase.

---

## Zero-code operators

The most immediate use case. Write a CRD definition, write a Katalog, run.

```yaml
# website-katalog.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: website-katalog
spec:
  crds:
    website:
      enabled: true
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
          services:
            - port: "80"
              targetPort: "{{ .spec.port }}"
              type: "{{ .spec.serviceType }}"
              reconcile: true
```

```bash
kubectl apply -f website-crd.yaml
ork run --katalog website-katalog.yaml
```

A Deployment and a Service are created for every `Website` CR. Owner
references are set. Drift is corrected. Cascade deletion works. The
health API is live. Metrics are flowing. No Go written.

---

## Platform namespace provisioning

Every platform team at every company writes some version of this controller.
Create a namespace, copy a pull secret, add a ConfigMap, create a ServiceAccount.

```yaml
operatorBox:
  default: true
  onCreate:
    configMaps:
      - name: "{{ .metadata.name }}-config"
        namespace: "{{ .spec.targetNamespace }}"
        data:
          ENVIRONMENT: "{{ .spec.environment }}"
          LOG_LEVEL: "{{ .spec.logLevel }}"
          TEAM: "{{ .spec.team }}"
        reconcile: true

    secrets:
      - name: registry-pull-secret
        fromSecret: docker-registry-creds
        fromNamespace: platform
        namespace: "{{ .spec.targetNamespace }}"
        reconcile: true

    serviceAccounts:
      - name: "{{ .spec.team }}-sa"
        namespace: "{{ .spec.targetNamespace }}"
```

One CR creates a fully provisioned namespace. Change `spec.logLevel` and
the ConfigMap updates on the next reconcile. Delete the CR and every
child resource is cleaned up automatically.

This is what ClusterSecret, Namespace Configurator, and custom namespace
controllers all do separately. In Orkestra it is one Katalog entry.

---

## Secret distribution across namespaces

A common platform requirement — one source Secret, many namespace copies.
When the source rotates, every copy updates automatically.

```yaml
secrets:
  - name: db-credentials
    fromSecret: master-db-creds
    fromNamespace: platform
    toNamespaces:
      - "{{ .metadata.namespace }}"
      - monitoring
      - staging
    reconcile: true
```

Orkestra reads the source once and writes to every namespace. Owner
references mean cleanup is automatic when the CR is deleted.

---

## Multi-CRD dependency ordering

Operators with multiple CRDs that depend on each other start in the right
order every time.

```yaml
crds:
  project:
    dependsOn: []

  managednamespace:
    dependsOn: [project]      # starts after project is ready

  application:
    dependsOn:
      - project
      - managednamespace      # starts after both are ready
```

If a CRD is missing from the cluster at startup, Orkestra retries in
the background without blocking healthy CRDs. When it appears, it starts
automatically and signals its dependents.

---

## Centralised operator configuration

Katalogs are files. Files live in Git. When your CRD configuration is in
Git, it becomes a GitOps artifact — versioned, reviewed, and auditable.

**Platform teams publish standard configurations:**

```
https://internal.company.com/platform/crds/standard-katalog.yaml
```

**Every team consumes it:**

```bash
ork run --katalog https://internal.company.com/platform/crds/standard-katalog.yaml
```

Changes to the Katalog propagate to every cluster that consumes it on
the next Orkestra restart. No binary rebuilds. No deployments. One file
change, everywhere updated.

---

## Environment-specific tuning

Different environments need different settings. A Komposer lets you
compose the shared Katalog with environment-specific overrides without
forking it.

```yaml
# production-komposer.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: production-komposer

sources:
  files:
    - https://internal.company.com/platform/crds/standard-katalog.yaml

spec:
  crds:
    # Override — production needs more workers
    application:
      workers: 10
      resync: 30s
      apiTypes:
        group: platform.myorg.io
        version: v1alpha1
        kind: Application
        plural: applications
      operatorBox:
        default: true
```

Development uses `workers: 2`. Production uses `workers: 10`. The same
source Katalog, two different overrides, no fork.

---

## Helm-driven operator configuration

Teams that already manage infrastructure with Helm can ship Orkestra
Katalog definitions as Helm chart templates. The chart becomes the
distribution mechanism.

```yaml
# charts/platform-crds/templates/katalog.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
spec:
  crds:
    {{- range .Values.crds }}
    - name: {{ .name }}
      apiTypes:
        group: {{ $.Values.apiGroup }}
        version: v1alpha1
        kind: {{ .kind }}
        plural: {{ .plural }}
      operatorBox:
        default: true
    {{- end }}
```

```yaml
# komposer.yaml
sources:
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 1.2.0
      valueFiles:
        - ./values/production.yaml
```

The chart renders a Katalog. Orkestra reads it. The team's existing Helm
workflow remains intact.

---

## Multi-team composition

Large organisations have platform teams, application teams, and security
teams — each owning their own CRDs. A Komposer brings them together.

```yaml
# platform-komposer.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: platform-komposer

sources:
  files:
    # Platform team — infrastructure CRDs
    - ./katalogs/namespaces.yaml

    # Application team — workload CRDs from their repository
    - https://raw.github.com/myorg/app-crds/main/katalog.yaml

    # Security team — policy CRDs
    - $SECURITY_KATALOG_URL
```

Each team owns their Katalog. The Komposer composes them. No team needs
access to another team's repository.

---

## Progressive rollout

Because Katalogs are URLs, you can run different versions on different
clusters and measure the difference.

```bash
# 90% of clusters — stable
ork run --katalog https://config.company.com/stable/katalog.yaml

# 10% of clusters — candidate
ork run --katalog https://config.company.com/candidate/katalog.yaml
```

Compare error rates, reconcile latency, and resource usage between the
two versions. Roll forward when confidence is high. Revert by pointing
back to the stable URL.

This is the same pattern as canary deployments but for operator configuration.

---

## Disaster recovery

A cluster can be fully restored from a Katalog. Point Orkestra at the
same file and the same operators start with the same configuration.

```bash
# Restore on a new cluster
ork run --katalog https://git.company.com/platform/crds/prod-katalog.yaml
```

No binary rebuild. No configuration migration. The Katalog is the
disaster recovery plan.

---

## Air-gapped and restricted environments

Katalogs are files. Files work offline.

```bash
# Downloaded to disk before the air-gap
ork run --katalog /etc/orkestra/crds/production-katalog.yaml
```

For organisations in air-gapped environments — government, finance,
healthcare — the operator configuration is bundled with the deployment
artifact. Nothing is fetched at runtime.

---

## Observability built in

Every Orkestra operator exposes the same endpoints regardless of what
is in the Katalog.

```bash
GET /health                          # orkestra liveness — 200 or 500
GET /ready                           # orkestra readiness — 200 or 503
GET /metrics                         # Prometheus metrics
GET /katalog                         # all CRDs with health and dependency graph
GET /katalog/{crd}                   # single CRD config and stats
GET /katalog/{crd}/health            # 200 healthy / 503 degraded
```

```
# Prometheus metrics
controller_reconcile_total{crd="...",result="success"}
controller_reconcile_total{crd="...",result="error"}
controller_reconcile_duration_seconds{crd="..."}
controller_queue_depth{crd="..."}
controller_workers_active{crd="..."}
controller_resource_count{crd="..."}
controller_crd_activation_latency_seconds{crd="..."}
controller_crd_activation_total{crd="..."}
```

These are not add-ons. They are part of the runtime. Every operator
you build with Orkestra has them from day one.

---

## When to use Go hooks

Declarative templates handle resource creation and drift correction without
any Go. Go hooks are the next step up — you keep GenericReconciler managing
the lifecycle, finalizers, events, and metrics, but you write the reconcile
logic in Go with full type-safe access to the CR.

```yaml
operatorBox:
  default: true
  hooks:
    location: github.com/myorg/hooks
    function: WebsiteHooks
```

**Reach for this when you need:**

1. **Type-safe spec access** — template expressions only see field names as
strings evaluated at runtime. A Go hook has compiled access to
`obj.Spec.Image`, `obj.Spec.Replicas`, `obj.Spec.DatabaseURL`. If your
reconcile logic depends on the shape of the spec being correct at compile
time, use hooks.

2. **External API calls alongside Kubernetes resources** — your reconcile
creates a Deployment and also registers the service in an external
service registry, or calls a cloud API to provision supporting
infrastructure. Templates only know about Kubernetes resources.

3. **Status writes** — computing derived status fields from the current
state of child resources and writing them back to `obj.Status`. The
GenericReconciler doesn't touch status — your hook does.

4. **Calling OrkestraRegistry directly** — hooks call `orkdeploy.Create`,
`orksvc.Create`, `orksecrets.CopyToNamespaces` directly. You get all the
registry implementations with none of the YAML declaration overhead.

Go hooks still run inside GenericReconciler. Finalizers, events, metrics,
and the workqueue are all handled for you.

---

## When to use a custom constructor

A custom constructor means you own the entire reconcile loop. Set
`reconciler.default: false` and declare where your constructor lives.
Orkestra wires it into the dependency graph, starts it in the correct
order, and surrounds it with the health API, metrics, and leader election.
The reconcile implementation is entirely yours.

```yaml
operatorBox:
  default: false
  constructor:
    location: github.com/myorg/reconcilers
    function: NewApplicationReconciler
```

**Reach for this when you need:**

1. **Complex state machines** — your CR progresses through multiple phases
with branching logic based on external state. `Pending → Provisioning →
WaitingForDNS → Ready → Degraded`. Each phase has different behaviour,
different retry strategies, different events. This is a state machine, not
a reconcile loop, and it needs to be written as one.

2. **Custom retry and backoff strategies** — the default workqueue retries
with exponential backoff. Some operators need domain-specific logic — back
off immediately on quota errors, retry faster on transient failures, give
up after N attempts for certain error classes. The constructor gives you
full control over what goes back on the queue and when.

3. **Custom finalizer orchestration** — when deletion requires a specific
sequence of external cleanup steps that must succeed in order before
finalizers are removed, and that sequence is too complex to express in
`onDelete` templates or a single hook function.

3. **Replacing an existing controller** — migrating a hand-written controller
into Orkestra without changing its reconcile logic. Wrap the existing
implementation in the constructor signature and Orkestra handles everything
around it.

4. **Integrating with controller-runtime or another framework** — some teams
have existing reconcilers built with other frameworks. The constructor path
accepts anything that satisfies `domain.Reconciler`. Wrap it and compose.

> You still get everything Orkestra provides — the informer, workqueue, dependency graph, health API, metrics, and leader election.

> You own the reconcile function. That is the only difference.

---

## The pattern that ties it together

Every use case above shares the same shape:

```
Declare what you want in a Katalog or Komposer
         ↓
ork run --katalog <path>
         ↓
Orkestra manages the lifecycle
```

The Katalog is the operator. It is the deployment artifact, the
configuration, and the documentation in one file. When something goes
wrong, you read the Katalog. When something needs to change, you change
the Katalog. When you need to reproduce a failure, you run the Katalog.

This is the simplification that Orkestra exists to provide.

**Whats Next?**
  - See [Templating Engine](../../runtime-manual/concepts/templating.md) to learn more. -->
