---
title: "Katalog Schema"
weight: 1
description: "A Katalog declares one or more CRDs and defines how Orkestra should manage"
---

A Katalog declares one or more CRDs and defines how Orkestra should manage
them: what resources to create, how to reconcile, how to convert between
versions, and what hooks or constructors to use for complex logic.

The Katalog is the unit of operator definition. Every operator in Orkestra
starts as a Katalog — written directly or pulled from a registry.

---

## Document structure

```yaml
apiVersion: orkestra.konductor.io/v1Alpha    # required
kind: Katalog                                # required

metadata:
  name: string        # required — unique identifier for this Katalog
  description: string # optional — shown in ork status and /katalog endpoint

spec:
  finalizers:         # optional — applied to all CRDs in this Katalog
    - string

  endpoints:          # optional — control health and info endpoint exposure
    enabled: bool
    health: bool
    info: bool

  crds:               # required — at least one CRD entry
    - ...
```

---

## `metadata`

### `metadata.name`

**Type:** `string` | **Required:** yes

Unique identifier for this Katalog. Used in `ork status`, log messages, and
the `/katalog` endpoint. Lowercase kebab-case by convention.

```yaml
metadata:
  name: website-operator
```

### `metadata.description`

**Type:** `string` | **Required:** no

Human-readable description. Shown in `ork describe` output and the
`/katalog/{crd}` JSON response. Encouraged for any Katalog that will be
shared or registry-distributed.

---

## `spec.finalizers`

**Type:** `[]string` | **Required:** no | **Default:** `[]`

Finalizers applied to every CR managed by every CRD in this Katalog.
These are merged with CRD-level finalizers — both sets apply.

```yaml
spec:
  finalizers:
    - finalizer.demo.orkestra.io/cleanup
```

!!! note
    Orkestra always adds its own internal finalizer (`orkestra.konductor.io/cleanup`)
    to every CR it manages, regardless of this field. Declared finalizers are
    additional, domain-specific finalizers that you control.

---

## `spec.endpoints`

**Type:** `object` | **Required:** no

Controls whether Orkestra exposes health and info endpoints globally for
this Katalog. Individual CRD entries can override these settings.

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `true` | If false, disables all endpoints for this Katalog |
| `health` | bool | `true` | Controls the `/katalog/{crd}/health` endpoint |
| `info` | bool | `true` | Controls the `/katalog/{crd}` info endpoint |

---

## `spec.crds[]` — CRD entry

The core of the Katalog. Each entry defines one CRD operator.

### Identity fields

| Field | Type | Default | Required | Description |
|---|---|---|---|---|
| `name` | string | — | yes | Unique identifier. Lowercase kebab-case. Used in `dependsOn`, `ork status`, and endpoint paths. |
| `enabled` | bool | `true` | no | If false, this CRD is parsed but not started. Useful for temporarily disabling a CRD without removing its definition. |
| `description` | string | `""` | no | Human-readable description shown in the health API. |

### `apiTypes`

Declares which Kubernetes resource this CRD entry watches.

| Field | Type | Default | Required | Description |
|---|---|---|---|---|
| `kind` | string | — | yes | The Kind to watch. For Kubernetes built-ins (Pod, Deployment, etc.), this is all that is needed. |
| `group` | string | — | yes* | API group. Empty string for core group resources. |
| `version` | string | — | yes* | API version: `v1`, `v1alpha1`, etc. |
| `plural` | string | — | yes* | Plural resource name used in API paths: `websites`, `deployments`. |
| `location` | string | — | no | Go import path for the typed struct. Required only when Go hooks need concrete type assertions (`obj.(*MyType)`). Omit for dynamic mode. |

\* For Kubernetes built-in resources, `group`, `version`, and `plural` are
auto-enriched from the cluster's discovery API. Declare `kind` only:

```yaml
name: deployment-governance
apiTypes:
  kind: Deployment   # Orkestra enriches group, version, plural automatically
```

!!! tip "Built-in kind enrichment"
    Run `ork validate --katalog katalog.yaml` to see what Orkestra resolves
    for a kind-only declaration:

Response:
```txt
✓ deployment-governance
    kind: Deployment → enriched from built-in registry
    group: apps / version: v1 / plural: deployments / scope: Namespaced
```

### Runtime configuration

| Field | Type | Default | Required | Description |
|---|---|---|---|---|
| `namespaced` | bool | `true` | no | Whether the CRD is namespace-scoped. Auto-detected when `apiTypes` is enriched. |
| `workers` | int | `3` | no | Number of dedicated reconcile workers for this CRD. More workers = higher throughput. Each CRD's workers are isolated — they cannot be consumed by other CRDs. |
| `resync` | duration | `15s` | no | How often to re-trigger reconciliation for all CRs of this type, even without a change event. Format: `"30s"`, `"5m"`, `"1h"`. |

### `dependsOn`

**Type:** `[]string` | **Required:** no | **Default:** `[]`

Names of other CRD entries that must be started and ready before this CRD's
workers start. Orkestra computes a topological order from the dependency graph
and starts CRDs in that order.

```yaml
- name: application
  dependsOn:
    - project
    - namespace
```

!!! warning "Circular dependencies"
    Circular `dependsOn` declarations are detected during `ork validate` and
    are a fatal startup error. A CRD that depends on itself, directly or
    transitively, will produce:

    ```
    error: circular dependency detected: application → namespace → application
    ```

!!! note
    A CRD declared in `dependsOn` that is not present in the Katalog is
    retried in the background — it does not block healthy CRDs from starting.
    This allows partial deployments where some CRDs are installed later.

### `restrictedNamespaces`

**Type:** `[]string` | **Required:** no | **Default:** `[]`

Namespaces where Orkestra will not create child resources, regardless of what
a CR's spec requests. Applied before templates and hooks run.

Supports exact names and simple wildcard patterns:

```yaml
restrictedNamespaces:
  - kube-system
  - cert-manager
  - kube-*        # all namespaces starting with kube-
  - "*-system"    # all namespaces ending in -system
```

!!! warning "Additive across composition levels"
    Restrictions declared at the Komposer level cannot be removed by a
    CRD-level declaration. More specific levels add restrictions — they
    never remove them. A namespace restricted at the Komposer level is
    restricted for all CRDs, permanently.

### `endpoints`

Per-CRD endpoint control. Overrides the Katalog-level `spec.endpoints`
settings for this specific CRD.

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `true` | Disables all endpoints for this CRD when false |
| `health` | bool | `true` | Controls `/katalog/{crd}/health` |
| `info` | bool | `true` | Controls `/katalog/{crd}` info response |

### `queue`

| Field | Type | Default | Description |
|---|---|---|---|
| `maxQueueDepth` | int | `2000` | Maximum items in the workqueue before new items are dropped |
| `degradeThreshold` | int | `10` | Consecutive reconcile failures before this CRD is marked degraded |

---

## `spec.crds[].conversion`

Declarative version conversion rules. Declares how Orkestra should translate
objects between API versions when the Kubernetes API server sends a
`ConversionReview` request.

Requires `ENABLE_CONVERSION=true` and valid TLS certificates.

```yaml
conversion:
  storageVersion: v1
  paths:
    - from: v1alpha1
      to: v1
      spec:
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        seo:
          enabled: false   # default — v1alpha1 has no seo field
    - from: v1
      to: v1alpha1
      spec:
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        theme: "default"   # default — v1 has no theme field
```

| Field | Type | Required | Description |
|---|---|---|---|
| `storageVersion` | string | yes | The version Kubernetes uses for internal storage. All objects are stored in this version. |
| `paths` | array | yes | One entry per (from, to) version pair. |
| `paths[].from` | string | yes | Source version — bare string, e.g. `v1alpha1` |
| `paths[].to` | string | yes | Target version — bare string, e.g. `v1` |
| `paths[].spec` | object | yes | The output spec in the target version's format. Values support Go template expressions. |

!!! note "From and to are bare version strings"
    Kubernetes sends `desiredAPIVersion` as a full string
    (`demo.orkestra.io/v1alpha1`). Orkestra extracts the bare version before
    path lookup. Declare `from` and `to` without the group prefix:

```yaml
# Correct
- from: v1alpha1
  to: v1

# Wrong — group prefix not needed
- from: demo.orkestra.io/v1alpha1
  to: demo.orkestra.io/v1
```

!!! tip "Each version is its own operator entry"
    Multi-version CRDs in Orkestra have one Katalog entry per version. Each
    version gets its own informer, worker pool, and reconciler. Conversion
    rules live in the storage version's entry. See [Versioning](../runtime-manual/concepts/versioning.md).

---

## `spec.crds[].reconciler`

The reconciler block declares how Orkestra should reconcile CRs of this type.

### `reconciler.default`

**Type:** `bool` | **Default:** `true`

When true, Orkestra uses the GenericReconciler — the built-in reconcile loop
that interprets `onCreate`, `onReconcile`, and `onDelete` templates.

Set to false only when providing a fully custom reconciler via `constructor`.

### `reconciler.finalizers`

**Type:** `[]string` | **Default:** `[]`

Per-CRD finalizers. Merged with Katalog-level `spec.finalizers` — both sets
apply.

### `reconciler.hooks`

Go hooks for logic that cannot be expressed in templates — external API calls,
complex conditional logic, or type-safe struct access.

| Field | Type | Required | Description |
|---|---|---|---|
| `location` | string | yes | Go module import path: `github.com/myorg/hooks` |
| `function` | string | yes | Exported function name that returns `domain.AnyReconcileHooks` |
| `alias` | string | no | Alias for use in template expressions |

```yaml
hooks:
  location: github.com/myorg/website-hooks
  function: WebsiteHooks
```

!!! note
    Hooks and declarative templates are not mutually exclusive. A CRD can
    declare both `hooks` and `onCreate` templates. Hooks run first; templates
    run for resources not covered by hooks.

### `reconciler.constructor`

A fully custom reconciler that replaces the GenericReconciler entirely.
Use when you need complete control over the reconcile loop.

| Field | Type | Required | Description |
|---|---|---|---|
| `location` | string | yes | Go module import path |
| `function` | string | yes | Exported constructor function |
| `alias` | string | no | Alias for identification |

!!! warning
    When `constructor` is declared, `onCreate`, `onReconcile`, and `onDelete`
    template blocks are ignored. The custom reconciler is responsible for all
    reconcile logic.

### `reconciler.onCreate` / `onReconcile` / `onDelete`

Declarative template blocks. Each declares which resources to create, update,
or clean up at the corresponding lifecycle phase.

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      replicas: "{{ .spec.replicas }}"
      reconcile: true     # also correct drift on every onReconcile cycle

onDelete:
  jobs:
    - name: "{{ .metadata.name }}-cleanup"
      image: busybox
      command: ["sh", "-c", "cleanup.sh"]
```

**`reconcile: true` on any template resource** adds it to the `onReconcile`
phase automatically. You do not need to declare it in both `onCreate` and
`onReconcile` unless you need different configuration for each.

Available resource types:

| Key | Resource |
|---|---|
| `deployments` | Deployment |
| `services` | Service |
| `configMaps` | ConfigMap |
| `secrets` | Secret |
| `jobs` | Job |
| `cronJobs` | CronJob |
| `pods` | Pod |
| `serviceAccounts` | ServiceAccount |

---

## Template fields (shared across all resource types)

All template declarations share these common fields:

| Field | Type | Default | Description |
|---|---|---|---|
| `name` | string | `<cr-name>-<resource>` | Resource name. Supports template expressions. |
| `namespace` | string | `{{ .metadata.namespace }}` | Target namespace. Defaults to the CR's namespace. |
| `reconcile` | bool | `false` | Include this resource in drift correction on every reconcile. |
| `labels` | `[]{ key, value }` | `[]` | Labels applied to the resource. Values support template expressions. |
| `annotations` | `[]{ key, value }` | `[]` | Annotations applied to the resource. Values support template expressions. |
| `when` | `[]Condition` | `[]` | Conditions that must pass for this resource to be created. |

### `when` conditions

```yaml
when:
  - field: spec.environment
    equals: production

  - field: spec.replicas
    operator: gt
    value: "0"

  - field: spec.enabled
    operator: exists
```

| Field | Type | Required | Description |
|---|---|---|---|
| `field` | string | yes | Dot-notation path to the CR field: `spec.environment`, `metadata.labels.tier` |
| `equals` | string | no | Shorthand for `operator: equals`. Most common case. |
| `operator` | string | no | Comparison operator (see table below) |
| `value` | string | no | Comparison value. Not used for `exists`/`notExists`. |

| Operator | Description |
|---|---|
| `equals` | Field exactly matches value |
| `notEquals` | Field does not match value |
| `contains` | Field contains value as substring |
| `prefix` | Field starts with value |
| `suffix` | Field ends with value |
| `exists` | Field is present and non-empty |
| `notExists` | Field is absent or empty |
| `gt` | Field is numerically greater than value |
| `lt` | Field is numerically less than value |

All conditions in a `when` block are AND'd. An empty `when` block always passes.

---

## Resource template schemas

### DeploymentTemplate

| Field | Type | Default | Description |
|---|---|---|---|
| `name` | string | `<cr-name>-deployment` | Deployment name |
| `image` | string | — | Container image. Required. Supports templates. |
| `replicas` | string \| int | `1` | Replica count. Supports templates. |
| `port` | string \| int | — | Container port. Optional. |
| `namespace` | string | CR namespace | Target namespace |
| `reconcile` | bool | `false` | Correct drift on every reconcile |
| `labels` | `[]{ key, value }` | `[]` | Applied to Deployment and pod template |
| `annotations` | `[]{ key, value }` | `[]` | Applied to Deployment metadata |
| `resources.requests.cpu` | string | — | CPU request: `"100m"` |
| `resources.requests.memory` | string | — | Memory request: `"128Mi"` |
| `resources.limits.cpu` | string | — | CPU limit |
| `resources.limits.memory` | string | — | Memory limit |
| `when` | `[]Condition` | `[]` | Creation conditions |

### ServiceTemplate

| Field | Type | Default | Description |
|---|---|---|---|
| `name` | string | `<cr-name>-svc` | Service name |
| `type` | string | `ClusterIP` | `ClusterIP`, `NodePort`, or `LoadBalancer` |
| `port` | string \| int | — | Service port. Required. |
| `targetPort` | string \| int | same as `port` | Container port to route to |
| `namespace` | string | CR namespace | Target namespace |
| `reconcile` | bool | `false` | Correct port drift on every reconcile |
| `labels` | `[]{ key, value }` | `[]` | Applied to Service metadata |
| `when` | `[]Condition` | `[]` | Creation conditions |

!!! note
    The Service selector is always `orkestra-owner: <cr-name>` — it routes
    automatically to pods created by the Deployment registry for the same CR.
    Do not declare a custom selector.

### ConfigMapTemplate

| Field | Type | Default | Description |
|---|---|---|---|
| `name` | string | `<cr-name>-config` | ConfigMap name |
| `namespace` | string | CR namespace | Target namespace |
| `data` | `map[string]string` | `{}` | Key-value configuration data. Values support template expressions. |
| `fromConfigMap` | string | — | Copy data from this source ConfigMap |
| `fromNamespace` | string | — | Namespace of the source ConfigMap |
| `toNamespaces` | `[]string` | — | Distribute to multiple namespaces. Supports template expressions. |
| `reconcile` | bool | `false` | Re-sync with source on every reconcile |
| `when` | `[]Condition` | `[]` | Creation conditions |

### SecretTemplate

| Field | Type | Default | Description |
|---|---|---|---|
| `name` | string | `<cr-name>-secret` | Secret name |
| `namespace` | string | CR namespace | Target namespace |
| `data` | `map[string]string` | `{}` | Key-value secret data. Values are base64-encoded automatically. |
| `fromSecret` | string | — | Copy from this source Secret |
| `fromNamespace` | string | — | Namespace of the source Secret |
| `toNamespaces` | `[]string` | — | Distribute to multiple namespaces |
| `reconcile` | bool | `false` | Re-sync with source on every reconcile |
| `when` | `[]Condition` | `[]` | Creation conditions |

### JobTemplate

| Field | Type | Default | Description |
|---|---|---|---|
| `name` | string | `<cr-name>-job` | Job name |
| `namespace` | string | CR namespace | Target namespace |
| `image` | string | — | Container image. Required. |
| `command` | `[]string` | — | Container entrypoint |
| `args` | `[]string` | — | Container arguments |
| `backoffLimit` | int | `3` | Retry count before Job is marked Failed |
| `gracePeriodSeconds` | int | `0` | Time before force-delete |
| `labels` | `[]{ key, value }` | `[]` | Applied to Job metadata |
| `when` | `[]Condition` | `[]` | Creation conditions |

!!! tip "Jobs under `onDelete`"
    Jobs declared in `onDelete` block CR deletion until they complete, via
    owner reference `blockOwnerDeletion: true`. This is how Orkestra ensures
    cleanup runs before the CR is removed. Set `backoffLimit` to a value
    appropriate for the cleanup task.

### CronJobTemplate

| Field | Type | Default | Description |
|---|---|---|---|
| `name` | string | `<cr-name>-cronjob` | CronJob name |
| `namespace` | string | CR namespace | Target namespace |
| `schedule` | string | — | Cron expression: `"0 * * * *"`. Supports templates. |
| `image` | string | — | Container image. Required. |
| `command` | `[]string` | — | Container entrypoint |
| `args` | `[]string` | — | Container arguments |
| `reconcile` | bool | `false` | Correct schedule and image drift |
| `labels` | `[]{ key, value }` | `[]` | Applied to CronJob metadata |
| `when` | `[]Condition` | `[]` | Creation conditions |

---

## Error reference

### Dependency errors

```
error: circular dependency detected: application → namespace → application
```
`dependsOn` contains a cycle. Remove the circular reference.

```
warning: CRD "project" declared in dependsOn for "application" not found in Katalog
  — will retry in background
```
A dependency is missing from the cluster. Not fatal — retried automatically.

### API type errors

```
error: CRD "my-crd": apiTypes is partially specified — either declare kind
only (for built-ins) or declare all fields (kind, group, version, plural)
```
You declared some but not all of `group`, `version`, `plural`. Either omit
all three (for built-ins) or declare all three (for custom CRDs).

```
error: CRD "my-crd": kind "Foobar" is not a known Kubernetes built-in and
apiTypes is incomplete (missing group, version, plural)
```
`kind` was declared alone but does not match any built-in. Declare the full
`apiTypes` block for custom CRDs.

### Conversion errors

```
no conversion path declared for v1alpha1 → v1beta1 in kind "Website"
```
A `ConversionReview` arrived for a version pair not covered by `paths`. Add
the missing `from`/`to` path to the `conversion` block.

```
conversion server error: TLS_CERT is required
```
`ENABLE_CONVERSION=true` is set but `TLS_CERT` environment variable is not.

### Template errors

```
error: deployments[0]: resolving template "{{ .spec.imageName }}": field
"spec.imageName" not found in object
```
A template expression references a field that does not exist in the CR.
Check the field name against the CRD's OpenAPI schema.

```
warning: conditions: gt requires numeric values — condition evaluates false
  field: spec.replicas value: "many"
```
A `gt` or `lt` condition has a non-numeric value. Both the field value and
the condition value must be numeric strings.

### Constructor errors

```
error: constructor declared but reconciler.default is true — set
reconciler.default: false when using a custom constructor
```
When providing a custom constructor, the GenericReconciler must be disabled.
