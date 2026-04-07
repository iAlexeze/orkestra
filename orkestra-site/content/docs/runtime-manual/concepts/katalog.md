---
title: "Katalog"
weight: 118
---

# Katalog

### A Declarative Bundle for CRDs, Behavior, and Runtime Orchestration

The **Katalog** is the heart of Orkestra. It is a **declarative, composable, dependency‑aware bundle** that describes:

- what CRDs exist — built‑in or custom
- how they behave — workers, resync intervals, dependencies
- what resources they create — Deployments, Services, Secrets, ConfigMaps, and more
- how they validate and mutate incoming CRs

It is the single source of truth for your operator. You write one YAML file. Orkestra does the rest.

If the Kubernetes API is the "data plane," the **Katalog is the control plane** for your operator.

{{< callout type="tip" title="katalog is not a CRD" >}}
A katalog is not a CRD and [here is why](../../faqs/why-not-crds.md).
{{< /callout >}}

---

## What Is a Katalog?

A **Katalog** is a YAML file that declares one or more CRD entries. Each entry describes a Kubernetes resource — built‑in or custom — and how Orkestra should manage it.

**Built‑in resources** (Pod, Deployment, Secret, ConfigMap, ServiceAccount) require only `kind`. Orkestra queries the Kubernetes API to discover the group, version, plural, and scope automatically.

**Custom resources** require `group`, `version`, `kind`, and `plural`. You may also provide an `apiTypes.location` for typed mode, but it is optional — dynamic mode works with any CRD without code generation.

```yaml
# Built‑in — Kubernetes knows the rest
- name: pod-governance
  apiTypes:
    kind: Pod

# Custom — declare the API metadata
- name: website
  apiTypes:
    group: demo.orkestra.io
    version: v1alpha1
    kind: Website
    plural: websites
```

**No Go. No code generation. No controller‑runtime scaffolding.** Just YAML.

---

## A Simple Katalog

The simplest Katalog manages a single CRD with no reconciliation logic — just watches and reports health.

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: simple-observer
spec:
  crds:
    pod-observer:
      apiTypes:
        kind: Pod
```

Orkestra:
- Enriches `kind: Pod` to `core/v1`, `pods`, `namespaced`
- Creates informer, workers, health endpoints
- Exposes `/katalog/pod-observer/health` and Prometheus metrics
- No resources are created — just observation

---

## A Complete Katalog

A complete Katalog declares CRDs, reconciliation templatess and conditional creation.

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: website-katalog
  description: Manages the Website CRD with Deployment and Service

spec:
  finalizers:
    - finalizer.demo.orkestra.io/website

  crds:
    # ── Built‑in resources ─────────────────────────────────────────────
    # Orkestra enriches automatically. No group/version/plural needed.
    pod-governance:
      apiTypes:
        kind: Pod

    deployment-watcher:
      apiTypes:
        kind: Deployment
      endpoints:
        info: false          # hide the /info endpoint for this CRD

    secrets-manager:
      apiTypes:
        kind: Secret
      endpoints:
        info: false

    # ── Custom CRDs ─────────────────────────────────────────────────────
    backend:
      apiTypes:
        group: orkestra.konduktor.io
        version: v1alpha1
        kind: OrkApp
        plural: orkapps
      endpoints:
        enabled: false            # disable all endpoints for this CRD

    frontend:
      enabled: false              # disables this CRD and orkestra ignores it on startup
      namespaced: true
      description: Manages a web application Deployment and Service pair
      workers: 2
      resync: 30s
      dependsOn:
        - backend
      queue:
        maxQueueDepth: 500
        degradeThreshold: 10
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites

      # ── Reconciliation templates ─────────────────────────────────────
      # onCreate runs on every reconcile. Idempotent — skipped if exists.
      # reconcile: true also applies as drift correction.
      reconciler:
        default: true
        finalizers:
          - finalizer.demo.orkestra.io/website

        onCreate:
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              port: "{{ .spec.port }}"
              namespace: "{{ .metadata.namespace }}"
              reconcile: true
              labels:
                - key: app
                  value: "{{ .metadata.name }}"
                - key: managed-by
                  value: orkestra

          services:
            - name: "{{ .metadata.name }}-svc"
              type: "{{ .spec.serviceType }}"
              port: "80"
              targetPort: "{{ .spec.port }}"
              namespace: "{{ .metadata.namespace }}"
              reconcile: true
              # Conditional creation — only when exposePublicly is true
              when:
                - field: spec.exposePublicly
                  equals: "true"
              labels:
                - key: app
                  value: "{{ .metadata.name }}"
                - key: managed-by
                  value: orkestra

        # onDelete — owner references handle Deployment + Service cleanup.
        # No explicit onDelete needed for most cases.
```

---

## Conditional Creation

Resources can be conditionally created using `when` blocks. The `when` field accepts a list of conditions; all must match for the resource to be created.

```yaml
services:
  - name: "{{ .metadata.name }}-svc"
    when:
      - field: spec.exposePublicly
        equals: "true"
      - field: spec.environment
        equals: "production"
```

Available operators: `equals`, `notequals`, `exists`, `notexists`.

---

## Built‑in Resource Enrichment

Orkestra queries the Kubernetes API at startup to discover metadata for built‑in resources. You only need to specify `kind`.

```yaml
- name: pod-governance
  apiTypes:
    kind: Pod

# Orkestra discovers:
#   group: "" (core)
#   version: v1
#   plural: pods
#   namespaced: true
```

This works for any built‑in resource: Pod, Deployment, Secret, ConfigMap, ServiceAccount, Job, CronJob, and more.

---

## What Makes a Katalog

| Element | Purpose |
|---------|---------|
| `apiVersion` | Must be `orkestra.konduktor.io/v1Alpha` |
| `kind` | Must be `Katalog` |
| `metadata.name` | Unique identifier |
| `spec.finalizers` | Katalog‑level finalizers (inherited) |
| `spec.crds[]` | List of CRD entries |

### CRD Entry Fields

| Field | Default | Description |
|-------|---------|-------------|
| `name` | required | Unique identifier, lowercase kebab‑case |
| `enabled` | `true` | Include in runtime |
| `namespaced` | `true` | Scope: true = namespace, false = cluster |
| `workers` | `0` | Worker count (0 = use default) |
| `resync` | `0` | Resync interval (0 = use default) |
| `dependsOn` | `[]` | CRDs that must start first |
| `critical` | `false` | If true, CRD degradation degrades whole operator |
| `endpoints.enabled` | `true` | Disable all endpoints for this CRD |
| `endpoints.health` | `true` | Disable `/health` endpoint |
| `endpoints.info` | `true` | Disable `/info` endpoint |
| `reconciler.default` | `true` | Use GenericReconciler (zero code) |
| `reconciler.finalizers` | `[]` | Per‑CRD finalizers |
| `reconciler.onCreate` | `{}` | Resources to create |
| `reconciler.onReconcile` | `{}` | Drift correction resources |
| `reconciler.onDelete` | `{}` | Cleanup resources |
| `queue.maxQueueDepth` | `0` | Max queue depth (0 = use default) |
| `queue.degradeThreshold` | `0` | Failures before degraded (0 = use default) |

---

## CLI Integration

```bash
# Validate a Katalog
ork validate --katalog katalog.yaml

# Preview the merged result
ork template --katalog katalog.yaml --graph
ork template --katalog katalog.yaml --json

# Run the operator
ork run --katalog katalog.yaml

# Check status
ork status
```

---

## Summary

The Katalog transforms operator engineering from boilerplate into orchestration.

- **Declarative** – define, don't code
- **Built‑in aware** – Kubernetes knows the rest
- **Dependency‑aware** – topological ordering
- **Observable** – health endpoints, metrics, status
- **Extensible** – hooks, custom reconcilers

Use it to define your CRDs. Use it to wire your controllers. Use it to build multi‑CRD systems with elegance and clarity.

---

> For a complete list of all configurable options, see the **[Katalog and Komposer Reference](../../reference/katalog-komposer-reference.md)**.  
> For real‑world examples, see the **[Use Cases](../../use-cases/index.md)** documentation.