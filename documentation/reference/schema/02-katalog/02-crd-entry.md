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

→ [enrich.md](15-enrich.md)

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

→ Full Motif import schema: [motif.md](../01-motif/index.md)

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

### `idp` top-level fields

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `false` | `true` — this CRD gets a **[+ Create]** button in the Control Center and its schema is served at `GET /api/v1/schema/{kind}`. |
| `category` | — | Category label shown in the schema catalog (`GET /api/v1/schema/`). Groups related CRDs in the service catalog. |
| `description` | — | Short description shown in the catalog. Overrides the CRD-level description. |
| `ignoreFields` | — | Dot-notation field paths excluded from the IDP form. Use for internal or platform-managed fields that should not be visible to users. |
| `fields` | — | Optional form hints. Each key matches a field name in the CRD spec. Missing keys are rendered from the OpenAPI schema alone. |
| `include` | — | Path (relative to the katalog file) to a YAML file containing a `fields:` key. Inline `fields:` take precedence over included values. Expanded at load time. |
| `forceConflict` | `false` | When `true`, every Apply API request for this CRD uses `Force: true` on server-side apply — the gateway takes ownership of any conflicting fields rather than surfacing a conflict error. Equivalent to `helm --force-conflict`. Callers can still override per-request with `?overwrite=true`. |

`idp/platformresource.yaml` (the include file):

```yaml
fields:
  environment:
    label: "Environment"
    placeholder: "staging"
    category: "Basics"
    order: 1
  team:
    label: "Owning Team"
    category: "Basics"
    order: 2
```

### `idp.fields.<name>`

| Field | Description |
|-------|-------------|
| `label` | Display label. Overrides the OpenAPI schema `description`. |
| `hint` | Helper text shown below the field. |
| `placeholder` | Input placeholder. |
| `category` | Section heading in the form. Fields with the same category are grouped together. Empty defaults to `"Spec"`. |
| `order` | Sort order within the form. Fields without `order` follow fields that have it. |
| `when` | All conditions must pass for this field to be shown (AND). Uses the same `Condition` type as resource templates — see [06-when-conditions.md](06-when-conditions.md). Evaluated client-side in the Control Center; gateway/admission is the backstop. |
| `anyOf` | At least one condition must pass for this field to be shown (OR). When both `when` and `anyOf` are declared, both blocks must pass. |
| `required` | When `true`, marks the field as mandatory in the IDP form. The browser enforces this natively — the label shows an asterisk and the form cannot be submitted while the field is empty. Has no effect on fields currently hidden by a `when:` or `anyOf:` condition. For server-side enforcement use `validation.rules` with `action: deny`. |
| `disabled` | Non-empty string — field is rendered greyed-out with this message. Useful for platform-managed fields that should be visible but not editable. |

Without any `idp:` block on the CRD entry, the CRD is not exposed via the Apply API regardless of what the Katalog-level `gateway.applyAPI` config says.

→ [17-katalog-applyapi.md](17-katalog-applyapi.md) — Katalog-level Apply API config
→ [concepts/idp](../../../concepts/idp/) — conceptual overview

---

## Sub-schemas

| Field | Reference |
|-------|-----------|
| `apiTypes` | [apitypes.md](03-apitypes.md) |
| `enrich` | [enrich.md](15-enrich.md) |
| `operatorBox` | [operatorbox.md](04-operatorbox.md) |
| `conversion` | [conversion.md](09-conversion.md) |
| `validation` | [validation.md](07-validation.md) |
| `mutation` | [mutation.md](08-mutation.md) |
