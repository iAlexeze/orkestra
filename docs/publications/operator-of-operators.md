# Operator of Operators: Compositional Platform Engineering with Orkestra

*Orkestra Project — May 2026*

---

## Abstract

Orkestra operators have always been able to create Kubernetes primitives —
Deployments, Services, StatefulSets, Secrets. The `custom:` block in
HookTemplates extends this to Custom Resources, enabling one Orkestra operator
to instantiate and manage resources that are themselves watched by other
operators. Combined with Orkestra's `when:` conditions, `forEach:` expansion,
and `anyOf:` logic, this produces a compositional platform engineering model
where a single CRD can represent any combination of infrastructure concerns,
with the operator deciding at runtime which sub-operators to activate.

This paper describes the architecture, the proof-of-concept in the
`AppEnvironment` operator, and what the capability unlocks.

---

## 1. The gap this fills

Before this capability, Orkestra operators could declare what Kubernetes
primitives to create. They could not declare what other operators to
instantiate. This meant platform composition — "deploy a CacheCluster
when caching is enabled, a SearchIndex when search is enabled" — required
either:

- Multiple separate CRDs with manual application of each CR
- External orchestration (Helm, Argo, Flux) managing the CR lifecycle
- Platform engineers writing Go code to create child CRs dynamically

All three approaches break the declarative model. The platform engineer
either writes imperative code, or delegates to a tool outside Orkestra's
reconcile loop. The result is state that Orkestra cannot observe, drift
that Orkestra cannot correct, and deletion that Orkestra cannot order.

The `custom:` block brings child CR creation inside the reconcile loop.
Everything Orkestra knows how to do — conditional creation, drift correction,
ordered deletion, owner references, event emission — now applies to child
CRs exactly as it applies to Deployments.

---

## 2. What custom: is

`custom:` is a field in `HookTemplates` alongside `deployments:`,
`statefulsets:`, `services:`, and every other resource type Orkestra manages.
It accepts the same primitives:

```yaml
custom:
  - apiVersion: platform.example.io/v1alpha1
    kind: CacheCluster
    when:
      - field: spec.cache.enabled
        equals: "true"
    metadata:
      name: "{{ .metadata.name }}-cache"
      namespace: "{{ .metadata.namespace }}"
    spec:
      size: "{{ .spec.cache.size }}"
      ttl: "{{ .spec.cache.ttl }}"
    hasStatus: false
    reconcile: true
```

This entry behaves identically to a Deployment entry in every respect:

- `when:` gates creation on a condition
- `reconcile: true` enables drift correction on subsequent reconciles
- `{{ .metadata.name }}` uses the parent CR's name for consistent naming
- Owner reference is set — Kubernetes garbage-collects the child CR when
  the parent is deleted
- The `hasStatus: false` hint prevents Orkestra from attempting status
  writes on CRDs that do not expose the status subresource

The implementation uses the dynamic client and RESTMapper to resolve any
`apiVersion` + `kind` combination to a GroupVersionResource at runtime.
No code generation required. No CRD-specific client. Any installed CRD
works.

---

## 3. The AppEnvironment proof

The `appenvironment-operator` demonstrates the full capability:

```yaml
spec:
  crds:
    appenvironment:
      apiTypes:
        kind: AppEnvironment
      operatorBox:
        onCreate:
          custom:
            - apiVersion: platform.example.io/v1alpha1
              kind: CacheCluster
              when:
                - field: spec.cache.enabled
                  equals: "true"
              metadata:
                name: "{{ .metadata.name }}-cache"
              spec:
                size: "{{ .spec.cache.size }}"
              hasStatus: false

            - apiVersion: platform.example.io/v1alpha1
              kind: SearchIndex
              when:
                - field: spec.search.enabled
                  equals: "true"
              metadata:
                name: "{{ .metadata.name }}-search"
              spec:
                indexName: "{{ .spec.search.indexName }}"
              hasStatus: false

    cachecluster:
      apiTypes:
        kind: CacheCluster
      operatorBox:
        onCreate:
          deployments:
            - image: "redis:7-alpine"
              port: 6379

    searchindex:
      apiTypes:
        kind: SearchIndex
      operatorBox:
        onCreate:
          deployments:
            - image: "opensearchproject/opensearch:2.11.0"
              port: 9200
```

A user applies one AppEnvironment CR:

```yaml
apiVersion: platform.example.io/v1alpha1
kind: AppEnvironment
metadata:
  name: prod-env
spec:
  tier: production
  cache:
    enabled: "true"
    size: "2Gi"
    ttl: "3600"
  search:
    enabled: "true"
    indexName: "prod-index"
    replicas: "3"
```

Orkestra reconciles the AppEnvironment CR. The `when:` conditions pass.
Orkestra creates a `CacheCluster` CR named `prod-env-cache` and a
`SearchIndex` CR named `prod-env-search`. The `cachecluster` and
`searchindex` reconcilers in the same Katalog then pick up their respective
CRs and create Redis and OpenSearch deployments.

The user applied one CR. Three separate reconcile loops ran. Six
Kubernetes resources were created. One YAML file described the intent.

Change `cache.enabled` to `"false"` and update the CR. On the next
reconcile, the `when:` condition fails. Orkestra deletes the `CacheCluster`
CR. The `cachecluster` reconciler sees the deletion and removes the Redis
deployment. The platform contracts to match the updated intent.

---

## 4. The architectural properties

### Conditionality is first-class

Every child CR is gated by `when:`. The parent operator decides at runtime
whether to instantiate a sub-operator based on the parent CR's spec. This
is not a static composition — it is a dynamic one. A single CRD can
represent a lean development environment (no cache, no search) and a full
production environment (both enabled) with the same operator handling both.

### Drift correction applies to child CRs

`reconcile: true` on a custom resource entry means Orkestra compares the
desired spec (from the parent CR's template) against the existing child CR's
spec on every reconcile. If they differ, Orkestra updates the child CR.
The child operator then reconciles to match the updated spec. Drift at
any level in the composition is corrected automatically.

### Deletion is ordered

The `custom:` entries participate in Orkestra's ordered deletion model.
When the parent CR is deleted, Orkestra can be configured to delete child
CRs in a specific order, wait for each to be fully deleted before proceeding,
and emit events throughout. Infrastructure teardown is as structured as
infrastructure creation.

### Owner references cascade Kubernetes garbage collection

Every child CR created by `custom:` has the parent CR as its owner reference.
When the parent is deleted without going through Orkestra's ordered deletion
(for example, a direct `kubectl delete`), Kubernetes garbage collection
removes the child CRs. The child operators then clean up their resources.
Nothing is orphaned.

### The same primitives, all the way down

`custom:` supports `when:`, `anyOf:`, `forEach:`, `reconcile:`, `sleep:`.
These are the same primitives that every other resource type in HookTemplates
supports. There is no special API for operator composition. The Katalog
author writes the same patterns they already know.

---

## 5. What this unlocks

### Platform CRDs

A platform team can now write a single CRD — `AppEnvironment`,
`MicroservicePlatform`, `DataPipeline` — that encapsulates any combination
of infrastructure. The CRD's spec defines what the platform offers. The
operator's `custom:` blocks define which sub-operators implement each
offered capability. The platform user sees one CRD with a clean API.
The platform team manages one Katalog with composable operators.

### Operator hierarchies

Orkestra can now express operator hierarchies that mirror organizational
boundaries. A cluster-level `Platform` operator creates namespace-level
`Tenant` CRs. Each `Tenant` operator creates application-level `Service`
CRs. Each `Service` operator creates Deployments, Services, and HPAs.
Three levels of operator, one Katalog, one reconcile loop per level,
fully declarative.

### Conditional infrastructure

The `when:` condition on `custom:` entries enables infrastructure that
responds to application state. When a `Service` CR's `spec.tier` changes
from `development` to `production`, the operator creates a `CacheCluster`
CR. The cache sub-operator activates. The service gets a sidecar connection.
All triggered by a spec field change on the parent CR.

### Motifs for operator composition

The Motif system — Orkestra's reusable primitive layer — can now include
Motifs that compose operators, not just Kubernetes resources. A
`postgres-platform` Motif might import a `postgres` Motif (for the
StatefulSet and Service) and a `backup-operator` Motif (for a CronJob
that creates `Backup` CRs managed by a backup operator). The composition
is declarative. The inputs are typed. The validation is static.

### The Orkestra registry becomes an operator marketplace

Operators in the registry are no longer just reconcilers for specific CRDs.
They are building blocks that other operators can instantiate. A
`monitoring` operator from the registry can be referenced in a `custom:`
block — when a service opts into monitoring, the parent operator creates
the monitoring CR, and the monitoring operator activates. The registry
becomes a marketplace of composable operator behaviors.

---

## 6. The mental model shift

Before `custom:`, Orkestra operators created infrastructure. After `custom:`,
Orkestra operators create infrastructure and orchestrate other operators.

The distinction matters because it changes the granularity at which Orkestra
operates. An operator that creates a Deployment is managing compute. An
operator that creates a `CacheCluster` CR is managing a concern — caching —
and delegating the implementation of that concern to the `CacheCluster`
operator.

This delegation is the correct model for platform engineering at scale.
Platform teams do not manage Redis clusters. They manage the concern of
caching. The concern has an API (`CacheCluster`). The implementation is
hidden behind the API. The operator hierarchy enforces this separation.

Orkestra's `custom:` block makes this separation declarative.