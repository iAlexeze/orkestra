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
| `namespace` | — | Template expression resolving the namespace a new CR is created in. Required on a namespaced CRD with `idp.enabled: true`; rejected on a cluster-scoped one. See [`idp.namespace`](#idpnamespace) below. |

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
| `order` | Sort order within the form *and* validation-rule priority — see note below. `0`/unset follows every field that declares one. |
| `when` | All conditions must pass for this field to be shown (AND). Uses the same `Condition` type as resource templates — see [06-when-conditions.md](06-when-conditions.md). Evaluated client-side in the Control Center; gateway/admission is the backstop. |
| `anyOf` | At least one condition must pass for this field to be shown (OR). When both `when` and `anyOf` are declared, both blocks must pass. |
| `required` | When `true`, marks the field as mandatory — enforced both client-side (the browser shows an asterisk and blocks submission while empty) and server-side: an implicit `exists` rule with `action: deny` is synthesized automatically at load time, so every caller of the Apply API is covered, not just the Control Center form. No matching `validation.rules` entry needs to be hand-written. Has no effect on fields currently hidden by a `when:` or `anyOf:` condition. |
| `disabled` | Non-empty string — field is rendered greyed-out with this message. Useful for platform-managed fields that should be visible but not editable. |

`order` isn't just cosmetic form layout. When multiple `required`/`type: enum` fields fail validation at once, only the first violation is reported as the headline denial reason — and synthesized rules are evaluated in the same order `order` puts the fields in, so the field a developer sees *first* on the form is also the one whose error they see first if several are wrong simultaneously. Two fields on the same CRD sharing a non-zero `order` value is a load-time error (`ork validate`) for exactly this reason — `0`/unset is the only value any number of fields may share, since it means "no preference," not a real position.

### `idp.namespace`

For a namespace-scoped CRD, something has to decide which namespace a self-service `POST /api/v1/apply` creates the CR in — and a form or a CI `curl` shouldn't be the one deciding it, any more than it should be the one choosing which Deployment image tag is "correct." `idp.namespace` is a template expression the gateway resolves server-side, against exactly what the caller submitted (labels, annotations, spec — the same data `validation.rules` sees), and always wins over whatever (if anything) the caller sent:

```yaml
idp:
  enabled: true
  namespace: '{{ teamName }}'    # or a note, or a literal: "platform-workloads"
```

Once this is set, the Control Center form never renders a namespace field, and no Apply API caller — curl, CI, the Control Center — needs to know or supply one. `name` plus the declared fields is the whole contract.

Three things `ork validate` enforces about it:

- **Required on a namespaced CRD with `idp.enabled: true`.** CRDs are namespaced by default (`namespaced: false` opts out) — without `idp.namespace`, self-service creation has no way to know where the CR belongs.
- **Rejected on a cluster-scoped CRD** (`namespaced: false`) — there's no namespace to resolve into, so declaring one is always a mistake, not a preference. (A cluster-scoped CRD sidesteps the whole namespace question a different way — see the note below.)
- **Rejected when templated *and* the CRD's informer is pinned to one fixed namespace** (`allowedNamespaces` with exactly one entry, or the legacy `namespace:` field). A templated expression can resolve differently per submission; a CR placed outside the one namespace the informer watches would exist in the cluster but never be reconciled, silently.

`idp.namespace` **routes into** a namespace — it doesn't create one. Whatever it can resolve to must already exist; the platform team provisions it ahead of time (`kubectl apply -f namespace.yaml` in a real rollout, `setup.apply` in an e2e fixture), the same way a namespaced CRD already requires today.

This only affects the Apply API. A raw `kubectl apply` is unaffected either way — `kubectl` always resolves *some* namespace client-side before a request ever reaches the API server (typically `default`), so there's never a genuinely empty namespace for anything server-side to notice and fill in the way an omitted JSON field lets the Apply API detect intent. `idp.namespace` is deliberately not a mutating-webhook default for that reason — it would only ever see the wrong namespace to silently override, never a blank one to fill in.

**The cluster-scoped alternative.** A CRD can sidestep this entirely by being cluster-scoped (`namespaced: false`) and having `onCreate` provision a namespace as a *child resource* of the CR instead — the CR itself has no namespace, so there's nothing for `idp.namespace` to resolve. Two different answers to the same "a developer shouldn't have to pick a namespace" problem, matched to two different scope choices: cluster-scoped + `onCreate`-provisions-a-child-namespace, or namespaced + `idp.namespace`-routes-into-a-platform-provisioned-one.

### `idp.additionalFields`

Exposes label and annotation keys as self-service form fields, written to `metadata.labels`/`metadata.annotations` on apply instead of `spec`. Flat — `labels` and `annotations` are the only two buckets; there is no third kind of Kubernetes object-metadata field to add later.

```yaml
idp:
  enabled: true
  additionalFields:
    labels:
      team:
        label: "Team"
        placeholder: "team-payments"
        required: true
    annotations:
      canary.myorg.io:
        label: "Enable canary rollout"
        type: boolean
      cost-center.myorg.io:
        label: "Cost Center"
        type: enum
        enum: ["finance", "engineering", "sales"]
```

Each entry is an `IDPFieldConfig` — the same shape as `idp.fields.<name>` above, plus two fields that only matter here (spec fields always infer these from the CRD's OpenAPI schema instead):

| Field | Description |
|-------|-------------|
| `type` | Required for additionalFields — labels/annotations have no CRD schema to infer type from. One of `string` (default), `integer`, `number`, `boolean`, `enum`. |
| `enum` | Valid values when `type: enum`. Required in that case — `ork validate` rejects `type: enum` with no `enum:` list. |

Validated at `ork validate` time: every key must be a syntactically valid Kubernetes label/annotation key (`[prefix/]name`), and no key may collide with `idp.fields` or the other `additionalFields` bucket.

→ [concepts/idp — Additional Fields](../../../concepts/idp/01-additional-fields.md) — why this exists, and the boolean-checkbox gotcha with `hasAnnotation`

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
