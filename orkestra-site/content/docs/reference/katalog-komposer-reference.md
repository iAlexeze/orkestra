---
title: "Katalog Komposer Reference"
weight: 91
---

# Katalog and Komposer Reference

This page is the single-page reference for Orkestra's two core document
kinds — Katalog and Komposer. Use it when you need to look up a specific
field quickly, or when you want to understand how the two documents relate
before reading their individual schema pages.

For full field-by-field documentation with error references, see:

- [Katalog Schema →](./katalog-schema.md)
- [Komposer Schema →](./komposer-schema.md)

---

## The relationship

```
                    ┌─────────────────────────────┐
                    │         Komposer            │
                    │                             │
                    │  sources:                   │
                    │    files: [Katalog, ...]    │
                    │    helm:  [Chart, ...]      │
                    │    registry: [Pattern, ...] │
                    │                             │
                    │  spec.crds: [overrides]     │
                    └──────────────┬──────────────┘
                                   │ merges into
                                   ▼
                    ┌─────────────────────────────┐
                    │     Unified Configuration   │
                    │  (validated, deduplicated)  │
                    └──────────────┬──────────────┘
                                   │
                                   ▼
                    ┌─────────────────────────────┐
                    │      Orkestra Runtime       │
                    │  per-CRD: informer, queue,  │
                    │  workers, reconciler, health│
                    └─────────────────────────────┘
```

A **Katalog** declares CRDs and their behavior. It is the atomic unit —
one operator definition, usable alone or as a source for a Komposer.

A **Komposer** composes Katalogs. It declares where operator definitions
come from, then merges them into one runtime configuration. Inline
`spec.crds` on a Komposer are overrides — they win over any source.

You can run either directly:

```bash
ork run --katalog katalog.yaml     # single Katalog — one or more CRDs
ork run --katalog komposer.yaml    # Komposer — many sources merged
```

---

## Quick-reference tables

### Katalog top-level

| Field | Type | Required | Default |
|---|---|---|---|
| `apiVersion` | `orkestra.konductor.io/v1Alpha` | yes | — |
| `kind` | `Katalog` | yes | — |
| `metadata.name` | string | yes | — |
| `metadata.description` | string | no | `""` |
| `spec.finalizers` | `[]string` | no | `[]` |
| `spec.endpoints.enabled` | bool | no | `true` |
| `spec.endpoints.health` | bool | no | `true` |
| `spec.endpoints.info` | bool | no | `true` |
| `spec.crds` | `[]CRDEntry` | yes | — |

### CRD entry

| Field | Type | Required | Default |
|---|---|---|---|
| `name` | string | yes | — |
| `enabled` | bool | no | `true` |
| `namespaced` | bool | no | `true` |
| `workers` | int | no | `3` |
| `resync` | duration | no | `15s` |
<!-- | `critical` | bool | no | `false` | -->
| `description` | string | no | `""` |
| `dependsOn` | `[]string` | no | `[]` |
| `restrictedNamespaces` | `[]string` | no | `[]` |
| `apiTypes.kind` | string | yes | — |
| `apiTypes.group` | string | yes* | auto-enriched |
| `apiTypes.version` | string | yes* | auto-enriched |
| `apiTypes.plural` | string | yes* | auto-enriched |
| `apiTypes.location` | string | no | — |
| `endpoints.enabled` | bool | no | `true` |
| `endpoints.health` | bool | no | `true` |
| `endpoints.info` | bool | no | `true` |
| `queue.maxQueueDepth` | int | no | `2000` |
| `queue.degradeThreshold` | int | no | `10` |
| `conversion.storageVersion` | string | when conversion used | — |
| `conversion.paths` | `[]ConversionPath` | when conversion used | — |
| `reconciler.default` | bool | no | `true` |
| `reconciler.finalizers` | `[]string` | no | `[]` |
| `reconciler.hooks.location` | string | when hooks used | — |
| `reconciler.hooks.function` | string | when hooks used | — |
| `reconciler.constructor.location` | string | when constructor used | — |
| `reconciler.constructor.function` | string | when constructor used | — |
| `reconciler.onCreate` | object | no | — |
| `reconciler.onReconcile` | object | no | — |
| `reconciler.onDelete` | object | no | — |

\* Auto-enriched for Kubernetes built-in resources. Required for custom CRDs.

### Komposer top-level

| Field | Type | Required | Default |
|---|---|---|---|
| `apiVersion` | `orkestra.konductor.io/v1Alpha` | yes | — |
| `kind` | `Komposer` | yes | — |
| `metadata.name` | string | yes | — |
| `metadata.description` | string | no | `""` |
| `sources` | object | yes | — |
| `spec.crds` | `[]CRDEntry` (overrides) | no | `[]` |

### File source entry

| Field | Type | Required | Default |
|---|---|---|---|
| `url` (or direct string) | string | yes | — |
| `auth.type` | string | no | — |
| `auth.fromEnv` | string | for bearer/github | — |
| `auth.usernameFromEnv` | string | for basic | — |
| `auth.passwordFromEnv` | string | for basic | — |

### Helm source entry

| Field | Type | Required | Default |
|---|---|---|---|
| `repo` | string | yes | — |
| `chart` | string | yes | — |
| `version` | string | yes* | — |
| `path` | string | no | — |
| `valueFiles` | `[]string` | no | `[]` |
| `values` | object | no | `{}` |

### Registry source entry

| Field | Type | Required | Default |
|---|---|---|---|
| `url` | string | yes | — |
| `version` | string | no | `main`/`latest` |
| `oci` | bool | no | `false` |
| `useKomposer` | bool | no | `false` |
| `auth.type` | string | no | — |
| `auth.fromEnv` | string | for bearer/github | — |
| `auth.usernameFromEnv` | string | for basic | — |
| `auth.passwordFromEnv` | string | for basic | — |

### Shared template fields

| Field | Type | Default | All resources |
|---|---|---|---|
| `name` | string | `<cr-name>-<type>` | yes |
| `namespace` | string | CR namespace | yes |
| `reconcile` | bool | `false` | yes |
| `labels` | `[]{ key, value }` | `[]` | yes |
| `annotations` | `[]{ key, value }` | `[]` | yes |
| `when` | `[]Condition` | `[]` | yes |

### Condition operators

| Operator | Shorthand | Description |
|---|---|---|
| `equals` | `equals:` | Field exactly matches value |
| `notEquals` | — | Field does not match value |
| `contains` | — | Field contains value as substring |
| `prefix` | — | Field starts with value |
| `suffix` | — | Field ends with value |
| `exists` | — | Field is present and non-empty |
| `notExists` | — | Field is absent or empty |
| `gt` | — | Field is numerically greater than value |
| `lt` | — | Field is numerically less than value |

---

## Complete examples

### Minimal Katalog

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
          services:
            - port: "80"
              targetPort: "{{ .spec.port }}"
              reconcile: true
```

### Katalog with full options

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: website-operator
  description: Manages Website CRD with Deployment, Service, and cleanup Job

spec:
  finalizers:
    - finalizer.demo.orkestra.io/cleanup

  crds:
    website:
      enabled: true
      workers: 3
      resync: 30s
      description: Web application — creates Deployment and Service

      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites

      dependsOn:
        - project

      restrictedNamespaces:
        - kube-system
        - kube-*

      queue:
        maxQueueDepth: 500
        degradeThreshold: 5

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
              reconcile: true
              when:
                - field: spec.enabled
                  equals: "true"

          services:
            - name: "{{ .metadata.name }}-svc"
              port: "80"
              targetPort: "{{ .spec.port }}"
              reconcile: true

        onDelete:
          jobs:
            - name: "{{ .metadata.name }}-cleanup"
              image: busybox
              command: ["sh", "-c", "echo cleanup"]
              backoffLimit: 3
```

### Minimal Komposer

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: platform-komposer

sources:
  files:
    - ./katalogs/website.yaml

spec:
  crds:
    website:
      workers: 8
```

### Komposer with all source types

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: platform-komposer
  description: Platform-wide operator configuration

sources:
  registry:
    - url: ghcr.io/konduktor-io/orkestra-registry/postgres@v14
      oci: true
    - url: https://github.com/myorg/internal-registry@main
      auth:
        type: github
        fromEnv: GITHUB_TOKEN

  files:
    - ./katalogs/website.yaml
    - https://platform.myorg.io/crds/namespace.yaml
    - url: https://private.myorg.io/internal.yaml
      auth:
        type: bearer
        fromEnv: PLATFORM_TOKEN

  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 2.1.0
      valueFiles:
        - ./values/production.yaml
      values:
        workers: 4

spec:
  crds:
    postgres:
      workers: 8
    website:
      resync: 15s
```
