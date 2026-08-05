# idp

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

## `idp` top-level fields

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `false` | `true` — this CRD gets a **[+ Create]** button in the Control Center and its schema is served at `GET /api/v1/schema/{kind}`. |
| `category` | — | Category label shown in the schema catalog (`GET /api/v1/schema/`). Groups related CRDs in the service catalog. |
| `description` | — | Short description shown in the catalog. Overrides the CRD-level description. |
| `ignoreFields` | — | Dot-notation field paths excluded from the IDP form. Use for internal or platform-managed fields that should not be visible to users. |
| `fields` | — | Optional form hints. Each key matches a field name in the CRD spec. Missing keys are rendered from the OpenAPI schema alone. |
| `include` | — | Path (relative to the katalog file) to a YAML file containing a `fields:` key. Inline `fields:` take precedence over included values. Expanded at load time. |
| `forceConflict` | `false` | When `true`, every Apply API request for this CRD uses `Force: true` on server-side apply — the gateway takes ownership of any conflicting fields rather than surfacing a conflict error. Equivalent to `helm --force-conflict`. Callers can still override per-request with `?overwrite=true`. |
| `name` | — | Template expression resolving the CR's `metadata.name`. Optional, unlike `namespace` — when unset (the common case), the caller must supply a name. See [`idp.name`](#idpname) below. |
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

## `idp.fields.<name>`

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
| `path` | — | Dot-notation path mapping the field to a nested location in the CRD `spec`. When set, the field value is written to `spec.<path>` instead of `spec.<name>`. See [`path` — nested spec paths](#idpfieldspath) below. |

---
`order` isn't just cosmetic form layout. When multiple `required`/`type: enum` fields fail validation at once, only the first violation is reported as the headline denial reason — and synthesized rules are evaluated in the same order `order` puts the fields in, so the field a developer sees *first* on the form is also the one whose error they see first if several are wrong simultaneously. Two fields on the same CRD sharing a non-zero `order` value is a load-time error (`ork validate`) for exactly this reason — `0`/unset is the only value any number of fields may share, since it means "no preference," not a real position.

## `path` — nested spec paths

By default, `idp.fields` maps field names directly to top-level `spec` paths:

```yaml
fields:
  repository:
    label: "Repository"
  # → spec.repository
```

Use `path` to map a field to a nested location:

```yaml
fields:
  repository:
    path: app.repository
    label: "Repository"
  # → spec.app.repository

  cpu:
    path: app.resources.cpu
    label: "CPU Request"
  # → spec.app.resources.cpu
```

Callers submit flat field names — they don't need to know the nesting structure. The gateway maps the field to the correct location in the CRD.

```json
POST /api/v1/apply
{
  "target": "app",
  "repository": "myorg/app",
  "cpu": "500m"
}
```

→ [Full `path` reference](21-idp-nested-spec.md)


## `idp.name`

`metadata.name` exists on every CR regardless of scope, so `idp.name` doesn't care whether the CRD is namespaced or cluster-scoped — it applies uniformly either way. It's optional, not required, though: most CRDs still want the caller to choose a name, since multiple concurrent instances of the same underlying app are normal (PR previews, ephemeral environments), and a name is the only thing distinguishing them. Set `idp.name` only when instances are 1:1 with some other identity the caller already supplies, and a redeploy is meant to update that same CR in place rather than create a new one — a stable environment where only the image tag changes between deploys:

```yaml
idp:
  enabled: true
  name: '{{ repoSlug .spec.repository }}'
```

`idp.name` is a template expression the gateway resolves server-side, against exactly what the caller submitted (labels, annotations, spec — the same data `validation.rules` sees), and always wins over whatever (if anything) the caller sent. Once set, the Control Center form never renders a Name field.

**When it's not set:** the Apply API requires the caller to supply `metadata.name`, and rejects the request immediately with a structured violation (`metadata.name is required`) if it's empty — the same clean-rejection treatment every other Apply API failure gets, rather than letting the request fall through to a raw Kubernetes "name is required" error from the SSA patch.

**`requireIdpName`** — `GET /katalog` (and therefore the Control Center) exposes this as a computed field, `true` unless `idp.name` is declared. It's not something you set directly; it's derived from whether `idp.name` is present.

If a templated `idp.name` resolves to something surprising, it's immediately visible in the Apply API's own response (`name`, `pollUrl`) rather than silently breaking anything.

## `idp.namespace`

For a namespace-scoped CRD, something has to decide which namespace a self-service `POST /api/v1/apply` creates the CR in — and a form or a CI `curl` shouldn't be the one deciding it, any more than it should be the one choosing which Deployment image tag is "correct." `idp.namespace` works the same way as `idp.name` above — a template expression the gateway resolves server-side against exactly what the caller submitted, always winning over whatever (if anything) the caller sent — but only applies to namespaced CRDs, and (unlike `idp.name`) is required rather than optional:

```yaml
idp:
  enabled: true
  namespace: '{{ teamName }}'    # or a note, or a literal: "platform-workloads"
```

Once this is set, the Control Center form never renders a namespace field, and no Apply API caller — curl, CI, the Control Center — needs to know or supply one. `idp.name` plus `idp.namespace` (when both are set) leaves the declared fields as the whole contract, no Kubernetes identity questions left for the developer or CI to answer at all.

Three things `ork validate` enforces about it:

- **Required on a namespaced CRD with `idp.enabled: true`.** CRDs are namespaced by default (`namespaced: false` opts out) — without `idp.namespace`, self-service creation has no way to know where the CR belongs.
- **Rejected on a cluster-scoped CRD** (`namespaced: false`) — there's no namespace to resolve into, so declaring one is always a mistake, not a preference. (A cluster-scoped CRD sidesteps the whole namespace question a different way — see the note below.)
- **Rejected when templated *and* the CRD's informer is pinned to one fixed namespace** (`allowedNamespaces` with exactly one entry, or the legacy `namespace:` field). A templated expression can resolve differently per submission; a CR placed outside the one namespace the informer watches would exist in the cluster but never be reconciled, silently.

`idp.namespace` **routes into** a namespace — it doesn't create one. Whatever it can resolve to must already exist; the platform team provisions it ahead of time (`kubectl apply -f namespace.yaml` in a real rollout, `setup.apply` in an e2e fixture), the same way a namespaced CRD already requires today.

This only affects the Apply API. A raw `kubectl apply` is unaffected either way — `kubectl` always resolves *some* namespace client-side before a request ever reaches the API server (typically `default`), so there's never a genuinely empty namespace for anything server-side to notice and fill in the way an omitted JSON field lets the Apply API detect intent. `idp.namespace` is deliberately not a mutating-webhook default for that reason — it would only ever see the wrong namespace to silently override, never a blank one to fill in.

**The cluster-scoped alternative.** A CRD can sidestep this entirely by being cluster-scoped (`namespaced: false`) and having `onCreate` provision a namespace as a *child resource* of the CR instead — the CR itself has no namespace, so there's nothing for `idp.namespace` to resolve. Two different answers to the same "a developer shouldn't have to pick a namespace" problem, matched to two different scope choices: cluster-scoped + `onCreate`-provisions-a-child-namespace, or namespaced + `idp.namespace`-routes-into-a-platform-provisioned-one.

## `idp.additionalFields`

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

Here are the documentation entries for the new IDP features:

---

## `idp.target`

A caller-facing identifier for the CRD, decoupled from the Kubernetes kind. Callers use this in target mode instead of constructing a full CR.

```yaml
idp:
  enabled: true
  target: smartapp   # callers use this, not "AppRequest"
```

When omitted, defaults to the lowercased kind (e.g., `AppRequest` → `apprequest`). Validated for uniqueness at `ork validate` time.

**Caller usage:**

```bash
# Target mode — use the target
curl -X POST /api/v1/apply \
  -d '{"target": "smartapp", "repository": "...", "image": "..."}'

# Full CR mode — still works
curl -X POST /api/v1/apply \
  -d '{"apiVersion": "platform.myorg.io/v1", "kind": "AppRequest", ...}'
```

## `idp.config.response`

Configures how the Apply API response and resource GET responses are shaped.

### `idp.config.response.default`

Controls whether the full CR is included in the response before `payload` and `exclude` are applied.

| Value | Behavior |
|-------|----------|
| `true` (default) | Start with the full CR; `payload` fields are merged in, `exclude` paths are stripped |
| `false` | Start with an empty map; only `payload` fields appear |

### `idp.config.response.payload`

A map of named template expressions added to the response. Keys become top-level fields in the returned JSON.

```yaml
idp:
  config:
    response:
      payload:
        phase: '{{ .status.phase }}'
        serviceURL: '{{ serviceURL }}'
        queueDepth: '{{ .external.queueDepth.value | default 0 }}'
        nextSteps: '{{ nextSteps }}'
```

**Evaluation timing:**

| Endpoint | What's available |
|----------|------------------|
| `POST /api/v1/apply` | `.spec`, `.metadata`, labels, annotations — `.status` is absent |
| `GET /api/v1/resources/...` | Full CR including `.status` (runtime has written it) |

**Payload is always a flat map of only the declared fields — not the full CR.**

### `idp.config.response.poll`

Configures how the `pollUrl` in the Apply API response is generated.

```yaml
idp:
  config:
    response:
      poll:
        url: "{{ devServerURL }}"      # replaces the default URL
        field: status.phase            # appends ?field=value
```

| Field | Behavior |
|-------|----------|
| `url` | Replaces the default `/api/v1/resources/{kind}/{namespace}/{name}` with a custom template |
| `field` | Appends `?field=<value>` to whatever URL is resolved (default or custom) |

**When both are set:** `field` is appended to the custom URL.

**Default `pollUrl`:** `/api/v1/resources/{kind}/{namespace}/{name}`

**Examples:**

```yaml
# Only field — appends to default
poll:
  field: status.phase
# → /api/v1/resources/AppRequest/ns/name?field=status.phase

# Custom URL + field
poll:
  url: "{{ devServerURL }}"
  field: status.phase
# → http://dev-server:9999?field=status.phase

# Custom URL only
poll:
  url: 'https://monitor.myorg.io/status/{{ .metadata.name }}'
# → https://monitor.myorg.io/status/payments-api
```

### `idp.config.response.exclude`

A list of dot-notation field paths to strip from the response after `payload` is applied.

```yaml
idp:
  config:
    response:
      exclude:
        - metadata.managedFields
        - status.observedGeneration
        - '{{ toList (getAnnotation . "exclude-fields") }}'  # dynamic
```

Each entry can be:
- A plain string path — `metadata.managedFields`
- A template expression that resolves to a comma-separated string — `{{ toList (getAnnotation . "exclude-fields") }}`

**Exclusions apply to the GET response, not the payload itself.**

## `idp.allowedTokens`

Restricts which Apply API tokens can access this CRD, with per-token permissions and namespace scoping.

```yaml
idp:
  allowedTokens:
    include: allowed_tokens.yaml   # or inline:
    control-center:
      namespaces: ["default"]
      permissions:
        global: ["*"]              # full access
    ci-pipeline:
      namespaces: ["team-payments-staging"]
      permissions:
        resources: ["create", "update", "get", "list"]
        schema: ["get", "list"]
    security-audit:
      namespaces: ["default", "team-payments-staging"]
      permissions:
        resources: ["get", "list"]
```

### `allowedTokens` structure

| Field | Description |
|-------|-------------|
| `namespaces` | List of allowed namespaces. Empty means all namespaces. Ignored for cluster-scoped CRDs. |
| `permissions` | `global`, `schema`, and `resources` lists (see below). |

### `permissions` scopes

| Scope | Endpoints | Valid operations |
|-------|-----------|------------------|
| `global` | All endpoints | `get`, `list`, `create`, `update`, `delete`, `*` |
| `schema` | `GET /api/v1/schema`, `GET /api/v1/raw-schema` | `get`, `list` (only) |
| `resources` | `GET/POST/DELETE /api/v1/resources`, `POST /api/v1/apply` | `get`, `list`, `create`, `update`, `delete`, `*` |

### Resolution priority (per endpoint class)

1. Class-specific list (`schema:` or `resources:`) when non-empty
2. `global:` list when class-specific is empty
3. No access when both are empty

### Validation

`ork validate` checks:
- Every token name in `allowedTokens` exists in `gateway.applyAPI.auth.tokens`
- Schema permissions only contain `get` or `list`
- Namespaces are within CRD's `allowedNamespaces` and outside `restrictedNamespaces`
- When `global` is non-empty, class lists must be subsets

## `include:` support for tokens and allowedTokens

`include:` is supported in two new places, following the same pattern as `status.include`, `validation.include`, and `idp.include`:

### Gateway auth tokens

```yaml
gateway:
  applyAPI:
    auth:
      include: ./shared/tokens.yaml
      tokens:
        - name: control-center   # overrides if in included file
          secretRef:
            name: ork-apply-token
            key: token
```

### IDP allowedTokens

```yaml
idp:
  allowedTokens:
    include: ./shared/allowed-tokens.yaml
    control-center:   # overrides if in included file
      permissions:
        global: ["*"]
```

Both follow established merge semantics: included entries are loaded first, then inline entries override by name.

---

## Complete Example

```yaml
# katalog.yaml
idp:
  enabled: true
  target: smartapp
  name: '{{ repoSlug .spec.repository }}'
  namespace: '{{ teamName }}-{{ environmentName }}'
  include: ./fields.yaml

  allowedTokens:
    include: allowed_tokens.yaml

  config:
    response:
      default: true
      poll:
        url: "{{ devServerURL }}"
        field: status.phase
      payload:
        phase: '{{ .status.phase }}'
        serviceURL: '{{ serviceURL }}'
        queueDepth: '{{ .external.queueDepth.value | default 0 }}'
        nextSteps: '{{ nextSteps }}'
      exclude:
        - metadata.managedFields
        - '{{ toList (getAnnotation . "platform.myorg.io/exclude-fields") }}'
```

```yaml
# fields.yaml
fields:
  repository:
    label: "Repository"
    order: 1
  image:
    label: "Container Image"
    order: 2
  replicas:
    label: "Replicas"
    type: integer
    order: 4

additionalFields:
  labels:
    team:
      label: "Team"
      order: 0
    environment:
      label: "Environment"
      order: 3
```

```yaml
# allowed_tokens.yaml
control-center:
  namespaces: ["default"]
  permissions:
    global: ["*"]

ci-pipeline:
  namespaces: ["team-payments-staging"]
  permissions:
    resources: ["create", "update", "get", "list"]
    schema: ["get", "list"]
```

```yaml
# tokens.yaml
- name: control-center
  secretRef:
    name: ork-apply-token
    key: token
- name: ci-pipeline
  secretRef:
    name: ci-pipeline-token
    key: token
```

## See also
**Conceptual overview:** → [idp](../../../concepts/idp/)

**Apply API:** → [katalog-applyapi](17-katalog-applyapi.md)
