<div align="center">
  <img src="./docs/assets/logo.png" alt="Orkestra" height="96" />

  <h1>Orkestra</h1>
  <p><strong>A runtime for Kubernetes operators.</strong></p>
  <h3><em>Declare. Run.</em></h3>

  <p>
    <a href="https://goreportcard.com/report/github.com/orkspace/orkestra"><img src="https://goreportcard.com/badge/github.com/orkspace/orkestra" alt="Go Report Card" /></a>
    <a href="https://github.com/orkspace/orkestra/releases"><img src="https://img.shields.io/github/v/release/orkspace/orkestra" alt="Release" /></a>
    <img src="https://img.shields.io/badge/Go-1.22+-00ADD8.svg" alt="Go" />
    <img src="https://img.shields.io/badge/Kubernetes-1.28+-326CE5.svg" alt="Kubernetes" />
    <img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License" />
  </p>

  <p>
    <a href="https://orkestra.readthedocs.io">Docs</a> ·
    <a href="https://docs.orkestra.sh/getting-started">Quick Start</a> ·
    <a href="https://github.com/orkspace/orkestra/discussions">Discussions</a>
  </p>
</div>

---

You have a CRD. Kubernetes stores it, validates it, and serves it.

The only missing piece is something that watches it and acts on it.

Traditionally, that means Go. Informers, workqueues, reconcile loops, code generation, Dockerfiles, Helm charts. A software project per operator. Most engineers never start.

Orkestra removes that entirely.

```yaml
# Declare
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
      operatorBox:               # isolated environment for this operator in the runtime
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
# Run
ork run -f katalog.yaml
kubectl apply -f website.yaml
```

Orkestra creates the Deployment and Service, sets owner references, writes status, emits events, corrects drift, exposes metrics and a control center — without a single line of Go.

**See Control Center:**
```bash
ork control start

# → localhost:8081
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

<!-- Fifteen CRDs. One process. [~47 MB](https://cc.orkestra.sh). -->

---
## Getting started

```bash
# Install (macOS)
brew install orkspace/tap/ork orkspace/tap/orkcc

# Install (Linux)
curl -sSL https://get.orkestra.sh | bash

# Initialize an operator
ork init my-operator
cd my-operator

# Apply the CRD
kubectl apply -f examples/beginner/01-hello-website/crd.yaml

# Run the operator runtime
ork run --file examples/beginner/01-hello-website/katalog.yaml

# Apply a CustomResource
kubectl apply -f examples/beginner/01-hello-website/cr.yaml
```

---

### Watch live on Control Center
In another terminal, run:

```bash
ork control start

# → localhost:8081
```

For production, deploy with Helm:

```bash
helm install orkestra orkestra/orkestra \
  --set runtime.katalog.existingConfigMap=my-platform-katalog \
  --namespace orkestra-system \
  --create-namespace
```

The same Katalog you ran locally is what runs in production.

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

## Orkestra Registry

The registry is the package manager operators never had.

Traditional operators are binaries. One per CRD. One deployment per operator. Ten operators, ten processes, gigabytes of memory, ten release cycles to maintain. The ecosystem grew this way because the pattern demanded it.

Orkestra changes the unit of distribution. Operators are **Katalogs** — YAML **patterns** packaged as OCI artifacts. The runtime is already installed. The artifact tells it what to do.

```yaml
sources:
  registry:
    - url: ghcr.io/orkspace/registry/postgres:v14
      oci: true
    - url: ghcr.io/orkspace/registry/redis:v7
      oci: true
```

That is two complete operators. No binaries. No deployments. One Orkestra process runs both.

Operators are now **assembled** from the registry, **composed** with local overrides, **upgraded** by changing a version tag, and **shared** by pushing a Katalog to any OCI-compatible registry.

```yaml
spec:
  crds:
    postgres:
      workers: 8   # production override — everything else from the registry
```

Full documentation: [Orkestra Registry](https://docs.orkestra.sh/orkestra-registry/)

---
## Operator Autoscaler

Every CRD in Orkestra runs inside an isolated **operatorBox:** with its own workers, queue, and metrics. The **Operator Autoscaler** lets you scale these dynamically — no Go code, no external controller.

```yaml
operatorBox:
  autoscale:
    interval: 30s   # how often to evaluate conditions
    cooldown: 2m    # how long conditions must be false before reverting

    conditions:
      when:
        # Scale based on this operator's own metrics
        - field: metrics.queueDepth
          greaterThan: "300"

        # Or scale based on another operator's metrics (Cross IPC)
        - field: cross.db.metrics.queueDepth
          greaterThan: "500"
        - field: cross.db.metrics.workersBusyPercent
          greaterThan: "70"

    do:
      workers: 12
      queueDepth: 1000
```

When conditions are true, Orkestra applies the overrides immediately.  
When they’re false — and the cooldown has elapsed — it restores the CRD’s baseline.

Autoscaling is **declarative, safe, and fully reversible**.

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

With `ENABLE_ADMISSION_WEBHOOK=true`, or `security.webhooks.admission.enabled=true` these intercept `kubectl apply` synchronously at the API server. Without it, they run on every reconcile. One declaration. Two enforcement points.

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
        value: "{{ get .children.deployment "status" "readyReplicas" }}"
```

Status fields are resolved from the live CR and its children after every reconcile. No `updateStatus` calls. No diff logic. Declare what the status should contain. Orkestra writes it.

---

## Multi-version CRD conversion

When your schema evolves, Orkestra gives you two declarative options.

**Option 1 — Kubernetes conversion webhook (built-in)**

The same process that runs your operators serves the `/convert` endpoint. No separate webhook deployment. No additional TLS.

```yaml
conversion:
  storageVersion: v2
  paths:
    - from: v1
      to: v2
      spec:
        schedule: "{{ cronToMap .spec.schedule }}"

    - from: v2
      to: v1
      spec:
        schedule: "{{ cronFromMap .spec.schedule }}"
```

<!-- **In production:** [100,000+ conversions](https://cc.orkestra.sh). 0 failures. ~ 2ms average latency. -->

**Option 2 — Internal normalization (no webhook)**

For simple or single-direction schema evolution, `normalize:` canonicalizes field values inside the operatorBox: pipeline — no webhook deployment, no TLS, no `admissionregistration` API call. Ideal when you want a single storage representation without wiring up the Kubernetes conversion machinery.

```yaml
normalize:
  spec:
    chedule: "{{ cronFromAny .spec.schedule }}"  # orkestra note 'cronFromAny'
```

Runs before `onCreate`/`onReconcile`. The CR is patched with the normalized value before any resources are created.

---

## Cross-operator IPC

Operators observe each other's state explicitly. No shared caches. No hidden coupling.

```yaml
operatorBox:
  default: true
  cross:
    - crd: managed-database
      selector:
        name: "{{ .metadata.name }}-db"
      as: db
  onReconcile:
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        env:
          DB_HOST:
            value: "{{ get .cross.db "status" "endpoint" }}"
        when:
          - field: "{{ phase .cross.db }}"
            equals: Ready
```

The `Deployment` is not created until the database CR is Ready. When it is, the endpoint is injected automatically. No polling. No coordination code.

---

## State machine

Declarative phase progressions without a single line of Go. `when:` conditions gate each step; the resync loop is the clock.

```yaml
operatorBox:
  onCreate:
    jobs:
      # Step 1 — start build when no phase yet
      - name: "{{ .metadata.name }}-build"
        image: "{{ .spec.image }}"
        when:
          - field: status.phase
            operator: notExists
        reconcile: false     # Job is terminal — create once

      # Step 2 — run tests after build succeeds
      - name: "{{ .metadata.name }}-test"
        image: "{{ .spec.image }}"
        when:
          - field: status.phase
            equals: "Running/build"
          - field: "{{ jobSucceeded .children.job }}"
            equals: "true"

      # Step 3 — notify after tests pass
      - name: "{{ .metadata.name }}-notify"
        image: "{{ .spec.image }}"
        when:
          - field: status.phase
            equals: "Running/test"
          - field: "{{ jobSucceeded .children.job }}"
            equals: "true"

  status:
    fields:
      - path: phase
        value: "Running/build"
        when:
          - field: "{{ name children.job }}"
            hasSuffix: "-build"
      - path: phase
        value: "Succeeded"
        when:
          - field: status.phase
            equals: "Running/notify"
          - field: "{{ jobSucceeded .children.job }}"
            equals: "true"
```

Each reconcile advances one step and writes one state. The queue fires again on the next resync. This is level-triggered reconciliation — idempotent by design.

---

## Environment variables

Inject environment variables into Deployments from **literals**, **Secrets**, **ConfigMaps**, or **any mix of sources**.  
All values are template expressions resolved against the **live CR** at reconcile time.

Orkestra also lets you **create** the Secret/ConfigMap in the same `operatorBox:` before consuming them — no extra manifests, no extra controllers.

```yaml
operatorBox:
  onCreate:
    # Secret derived from the CR
    secrets:
      - name: "{{ .metadata.name }}-creds"
        once: true                   # Create once - prevents creation on every resync 
        rotateAfter: 30d             # Automatic rotation (no manual rotation needed)
        data:
          username: "{{ .spec.username }}"
          password: "{{ randomAlphanumeric 16 }}"     # Use orkestra note

    # ConfigMap derived from the CR
    configMaps:
      - name: "{{ .metadata.name }}-cfg"
        data:
          region: "{{ .spec.region }}"
          image: "{{ .spec.image }}"

    # Deployment consuming both
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        env:
          USERNAME:
            secretKeyRef:
              name: "{{ .metadata.name }}-creds"
              key: username
          PASSWORD:
            secretKeyRef:
              name: "{{ .metadata.name }}-creds"
              key: password
          REGION:
            configMapKeyRef:
              name: "{{ .metadata.name }}-cfg"
              key: region

        # Or make all envs available to deployment
        envFrom:
          - configMapRef: "{{ .metadata.name }}-cfg"
          - secretRef: "{{ .metadata.name }}-creds"
```

- All values are evaluated at reconcile time, so updates to the CR flow naturally into the Deployment.

---

## External gating

Gate resource creation on an HTTP call. The response status, body, and error are available as `.external.<name>.*` in all `when:` conditions and template expressions.

```yaml
operatorBox:
  onCreate:
    external:
      - name: healthCheck
        url: "{{ .spec.serviceUrl }}/health"
        method: GET
        expectedStatus: 200
        continueOnError: false
        timeout: 5s

      - name: featureFlags
        url: "{{ .spec.serviceUrl }}/flags/{{ .metadata.name }}"
        method: GET
        continueOnError: true
        timeout: 3s

    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        when:
          - field: external.healthCheck.status
            equals: "200"
        reconcile: true

    configMaps:
      - name: "{{ .metadata.name }}-flags"
        data:
          flags: "{{ .external.featureFlags.body }}"
        when:
          - field: external.featureFlags.called
            equals: "true"
          - field: external.featureFlags.error
            operator: notExists
        reconcile: true
```

`continueOnError: false` blocks the entire reconcile if the call fails. `continueOnError: true` lets the rest of the pipeline proceed — the error is available in `.external.<name>.error`.

---

## Composition

Pull Katalogs from files, Helm, Git, or OCI registries:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: platform
sources:
  registry:
    - url: ghcr.io/orkspace/orkestra-registry/postgres@v14
      oci: true
  files:
    - ./katalogs/website.yaml
    - ./katalogs/pipeline.yaml
  helm:
    - repo: ghcr.io/orkspace/registry/platform
      chart: platform-example
      version: 0.1.0
      valueFiles:
        - values/overrides.yaml
spec:
  crds:
    postgres:
      workers: 8
```

One command starts the entire platform.

---

## Security

Deletion protection, namespace protection, admission webhooks, and conversion webhooks all share one certificate. One block. No separate TLS setup.

```yaml
security:
  deletionProtection:
    enabled: true             # protects your CRDs and Orkestra deployment from kubectl delete
    cleanupOnShutdown: true   # Tells orkestra to cleanup deletionProtection webhooks and certs on shutdown
  
  namespaceProtection:
    enabled: true             # Orkestra blocks creation of custom resources in restrictedNamespaces at apply time and creation of child resources at reconcile time. One declaration. Two enforcement points.

  webhooks:
    admission:
      enabled: true        # intercepts kubectl apply at the API server
    failurePolicy: Fail

  conversion:
    enabled: true          # serves /convert for multi-version CRDs
```

With `deletionProtection` enabled, Orkestra registers a validating webhook that rejects `DELETE` requests to delete protected CRDs as well as Orkestra deployment, service or ingress. No separate webhook server. The same process that runs your operators handles it.

---
> [!important]
> Features in development

### Automatic rollbacks

Orkestra provides two rollback models: zero‑config recovery and custom rollback templates. Both approaches restore the last known good spec after repeated reconcile failures. Rollback is declarative and idempotent; no additional controllers or resource types are introduced.

#### Zero‑config rollback

A single field enables automatic recovery. When the operator encounters consecutive reconcile failures, Orkestra enters a rollback phase and reapplies the previous desired state. The rollback templates are derived from the existing reconcile declarations; no onRollback block is required.

```yaml
operatorBox:
  default: true

  rollBackOnError: true
```

When enabled, Orkestra:

- captures the previous spec before applying a new one  
- tracks consecutive failures  
- rolls back automatically after the threshold is reached  (3 consecutive failures in 10 minutes)
- blocks normal reconciliation until the spec is corrected  

This is the simplest and safest rollback path.

#### Custom rollback

For operators that require explicit rollback behavior, a custom rollback block can be declared. This allows full control over which resources are restored and how the previous spec is applied.

```yaml
operatorBox:
  rollback:
    trigger:
      consecutiveFailures: 3
      withinDuration: 5m

    onRollback:
      deployments:
        - name: "{{ .metadata.name }}"
          image: myorg/stable-deployment:v1
          replicas: "{{ .previous.spec.replicas }}"
          reconcile: true
```

Custom rollback provides:

- explicit control over rollback templates  
- conditional triggers  
- access to `.previous.spec.*` for restoring prior values  
- full integration with the existing reconcile pipeline  

Rollback is not transactional undo. It is re‑declaration of the last known good state. Existing Update functions handle idempotent re‑application.

---

### Providers
Declare infrastructure dependencies at the Katalog level. Orkestra registers only the providers listed here — per-CRD blocks for anything else are silently skipped.

```yaml
providers:
  - name: aws
    required: true
    auth:
      accessKeyId: "$AWS_ACCESS_KEY_ID"
      secretAccessKey: "$AWS_SECRET_ACCESS_KEY"
      region: "$AWS_REGION"
  - name: mongodb
    required: true
    auth:
      mongoUri: "$MONGODB_URL"
```

Then reference them inside any `operatorBox:`:

```yaml
operatorBox:
  providers:
    aws:
      - s3:
          bucket: "{{ .metadata.name }}-assets"
          region: "{{ .spec.region }}"
    mongodb:
      - database:
          name: "{{ .metadata.name }}"
      - user:
          name: "{{ .spec.dbUser }}"
          database: "{{ .metadata.name }}"
```

<!-- ---

## In production

| | |
|---|---|
| Live resources under management | 13,220 |
| Active operatorBox:es | 3 Katalogs, 113 workers |
| Reconcile error rate | **0.0%** |
| Conversion failures | **0** |
| Memory (15 CRDs) | ~47 MB | -->

---

## Documentation

| | |
|---|---|
| [Getting Started](https://docs.orkestra.sh/getting-started) | First operator in under an hour |
| [Katalog Reference](https://docs.orkestra.sh/reference/katalog-schema) | Complete field reference |
| [Examples](./examples/) | Beginner → advanced, all verified |
| [Concepts](https://docs.orkestra.sh/concepts) | Architecture and mental model |
| [Papers](https://docs.orkestra.sh/papers) | The case for declarative operators |

---

## Community

[Issues](https://github.com/orkspace/orkestra/issues) · [Discussions](https://github.com/orkspace/orkestra/discussions) · [Contributing](./CONTRIBUTING.md)

---

Apache 2.0 — see [LICENSE](./LICENSE)