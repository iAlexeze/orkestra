# CRDEntry

Each entry in `spec.crds` is a CRDEntry. The map key becomes the CRD name at runtime — it is never written in the YAML body.

```yaml
spec:
  crds:
    database:                  # ← this is the name
      enabled: true
      description: string
      crdFile: ./my-crd.yaml
      crFiles: [./my-cr.yaml]
      setup: [./my-setup.yaml]

      apiTypes:                # → apitypes.md
        ...

      namespaced: true
      namespace: default

      dependsOn:
        other-crd:
          condition: healthy

      enrich:                      # optional → enrich.md
        - pods
        - events

      labelSelector:
        app: my-operator
      fieldSelector:
        metadata.namespace: production

      operatorBox:             # → operatorbox.md
        reconciler:
          workers: 3
          resync: 30s
          queue:
            shared: false
            maxDepth: 100
            failureThreshold: 5
        ...

      conversion:              # → conversion.md
        ...

      validation:              # → validation.md
        ...

      mutation:                # → mutation.md
        ...

      webhooks:
        validation: true
        mutation: true
        operations: [CREATE, UPDATE]

      restrictedNamespaces:
        - kube-system
      allowedNamespaces:
        - dev

      endpoints:
        enabled: true
        health: true
        info: true

      imports:
        - motif: ./motifs/postgres/motif.yaml   # → motif.md
          with:
            image: postgres:14
```

## Identity and mode

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | `false` skips this CRD entirely — it is not started or watched. |
| `description` | string | — | Shown in the `/katalog` endpoint. |
| `mode` | string | auto | `typed` or `dynamic`. Auto-detected from `apiTypes.location`. |
| `crdFile` | string | — | Path or HTTPS URL to the CRD YAML. Used for CRD-driven API inference in dev mode. |
| ``crFiles`` | list(string) | — | Ordered list of CR YAML files to apply **after** the CRD is registered but **before** the runtime starts. Dev mode only. Supports relative paths and HTTPS URLs. |
| ``setup`` | list(string) | — | Ordered list of YAML files to apply **before** the operator starts. Use for external dependencies such as namespaces, Secrets, or additional CRDs. Applied with ``kubectl ``apply ``-f`` in declaration order. |

## Scope

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `namespaced` | bool | `true` | Whether the CRD is namespace-scoped or cluster-scoped. |
| `namespace` | string | — | Target namespace for namespaced CRDs. |

## Runtime behaviour

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `workers` | int | `3` | Concurrent reconcile goroutines. |
| `resync` | duration | `30s` | Full re-list interval (e.g. `30s`, `5m`). |
| `ignoreStatusPatch` | bool | `false` | Skip status patch operations. |
| `ignoreObservedGeneration` | bool | `false` | Skip the observed-generation idempotency check. |
| `removeFinalizers` | bool | `false` | Strip all finalizers on deletion. |

## `dependsOn`

CRDs that must reach a condition before this one starts reconciling.

```yaml
# All three forms are valid:

# map with condition
dependsOn:
  schema-migrator:
    condition: healthy

# scalar shorthand
dependsOn:
  schema-migrator: healthy

# list (bare names — condition defaults to "started")
dependsOn:
  - schema-migrator
```

| Condition | Description |
|-----------|-------------|
| `started` | Workers are running (default when no condition is given). |
| `healthy` | Workers running and consecutive failure count is zero. |

## `queue`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `shared` | bool | `false` | Use the shared default workqueue instead of a per-CRD queue. |
| `maxDepth` | int | `100` (`QUEUE_DEPTH` env) | Max items in the queue before new items are dropped. |
| `failureThreshold` | int | `5` (`FAILURE_THRESHOLD` env) | Consecutive reconcile failures before health transitions to degraded. |

## `enrich`

→ [enrich](15-enrich.md)

---

## `webhooks`

Per-CRD admission webhook override. Overrides `security.webhooks` for this CRD only.

| Field | Type | Description |
|-------|------|-------------|
| `validation` | bool | Include in `ValidatingWebhookConfiguration`. |
| `mutation` | bool | Include in `MutatingWebhookConfiguration`. |
| `operations` | []string | Which operations trigger the webhook. Default: `[CREATE, UPDATE]`. |

## `restrictedNamespaces` / `allowedNamespaces`

Per-CRD namespace guards. Override `security.namespaceProtection` lists for this CRD only.

```yaml
restrictedNamespaces:
  - kube-system
  - production
allowedNamespaces:
  - dev
  - staging
```

## `endpoints`

```yaml
endpoints:
  enabled: true
  health: true
  info: true
```

## `imports`

Motif imports — pull reusable resource templates into this CRD's reconciler.

```yaml
imports:
  - motif: ./motifs/postgres/motif.yaml
    with:
      image: postgres:14
  - motif: ghcr.io/orkspace/motifs/nginx:v1
    oci: true
    with:
      port: "8080"
```

→ Full Motif import schema: [motif](../01-motif/index.md)

## `idp`

Controls whether this CRD is exposed through the Gateway Apply API and the Control Center's IDP form. Requires `gateway.applyAPI.enabled: true` at the Katalog level.

```yaml
spec:
  crds:
    platformResource:
      idp:
        enabled: true
        category: "Compute"
        description: "Deploy and manage platform workloads"
        ignoreFields:
          - spec.internalRef
          - spec.managedBy
        include: ./idp/platformresource.yaml   # or inline fields:
        fields:
          environment:
            label: "Environment"
            placeholder: "staging"
            category: "Basics"
            order: 1
          workloadType:
            label: "Workload Type"
            category: "Basics"
            order: 2
          productionApproval:
            label: "Approval Ticket"
            hint: "Required for production deployments"
            category: "Governance"
            order: 10
            when:
              - field: environment
                equals: production
          certIssuer:
            label: "Certificate Issuer"
            category: "TLS"
            order: 20
            anyOf:
              - field: workloadType
                equals: cert
          maintenanceMode:
            label: "Maintenance Mode"
            disabled: "This field is managed by the platform team"
            order: 99
```

**Full reference:** → [idp](20-idp.md) — `idp.fields`, `idp.additionalFields`, `idp.name`, `idp.namespace`, `idp.config.response`, `idp.allowedTokens`.

## See also
**Conceptual overview:** → [concepts/idp](../../../concepts/idp/)
**Apply API:** → [katalog-applyapi](17-katalog-applyapi.md)

---

## Sub-schemas

| Field | Reference |
|-------|-----------|
| `apiTypes` | [apitypes](03-apitypes.md) |
| `enrich` | [enrich](15-enrich.md) |
| `operatorBox` | [operatorbox](04-operatorbox.md) |
| `conversion` | [conversion](09-conversion.md) |
| `validation` | [validation](07-validation.md) |
| `mutation` | [mutation](08-mutation.md) |
| `idp` | [idp](20-idp.md) |
