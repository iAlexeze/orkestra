# Validation Schema

Complete schema reference for `reconciler.validation` in a Katalog CRD entry.

---

## `reconciler.validation`

```yaml
operatorBox:
  validation:
    rules:
      - field: string           # required
        message: string         # required
        action: string          # optional, default: deny

        # Shorthands — use one
        equals: string
        notEquals: string
        prefix: string
        suffix: string
        contains: string
        min: string             # numeric, inclusive lower bound
        max: string             # numeric, inclusive upper bound

        # Explicit form — use when no shorthand applies
        operator: string        # see operator table below
        value: string
```

---

## `ValidationConfig`

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `rules` | `[]ValidationRule` | no | `[]` | Ordered list of validation rules. All are evaluated — not fail-fast. |

---

## `ValidationRule`

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `field` | string | yes | — | Dot-notation path to the CR field: `spec.image`, `metadata.labels.tier` |
| `message` | string | yes | — | Shown in the webhook rejection, Kubernetes event, and operator log. Make it actionable. |
| `action` | string | no | `deny` | `deny` or `warn`. See Actions below. |
| `equals` | string | no | — | Shorthand: field must exactly match this value |
| `notEquals` | string | no | — | Shorthand: field must not match this value |
| `prefix` | string | no | — | Shorthand: field must start with this value |
| `suffix` | string | no | — | Shorthand: field must end with this value |
| `contains` | string | no | — | Shorthand: field must contain this value as substring |
| `min` | string | no | — | Shorthand: field must be numerically >= this value |
| `max` | string | no | — | Shorthand: field must be numerically <= this value |
| `operator` | string | no | — | Explicit operator (see table). Used when no shorthand applies. |
| `value` | string | no | — | Comparison value for the explicit operator form. Not used for `exists`/`notExists`. |

!!! note "Shorthands and operator are mutually exclusive"
    When a shorthand is set (`prefix`, `max`, etc.), the `operator` and `value`
    fields are ignored. Use shorthands for common cases. Use `operator` + `value`
    for operators not covered by a shorthand.

---

## `action` values

| Value | At admission (ENABLE_ADMISSION_WEBHOOK) | At reconcile time |
|---|---|---|
| `deny` (default) | Synchronous rejection. `kubectl apply` returns error immediately. Object not stored. | Reconciliation halted. Warning event on CR. Workqueue retries with backoff. |
| `warn` | Warning header in response. `kubectl` prints warning to stderr. Object stored. | Active warning recorded on `/katalog/{crd}` health API. Reconciliation continues. |

---

## Operator values

| Shorthand | Operator string | Description | Value required |
|---|---|---|---|
| `equals:` | `equals` | Field exactly matches value | yes |
| `notEquals:` | `notEquals` | Field does not match value | yes |
| `prefix:` | `prefix` | Field starts with value | yes |
| `suffix:` | `suffix` | Field ends with value | yes |
| `contains:` | `contains` | Field contains value as substring | yes |
| `min:` | `gt` | Field is numerically >= value | yes (numeric string) |
| `max:` | `lt` | Field is numerically <= value | yes (numeric string) |
| — | `exists` | Field is present and non-empty | no |
| — | `notExists` | Field is absent or empty | no |

---

## Field path syntax

| Example path | Resolves to |
|---|---|
| `spec.image` | `obj.spec.image` |
| `metadata.labels.tier` | `obj.metadata.labels["tier"]` |
| `spec.database.engine` | `obj.spec.database.engine` |

Paths use dot-notation at any depth. Intermediate maps are traversed
automatically. A path that does not exist in the CR returns `found=false` —
`exists` rules fail, other rules depend on their semantics.

Field values are compared as strings after type coercion:
- `int64(5)` → `"5"`
- `float64(10.0)` → `"10"` (integer floats print without decimals)
- `bool(true)` → `"true"`

---

## `reconciler.webhooks` — per-CRD admission control

Controls whether this CRD participates in admission webhooks when
`ENABLE_ADMISSION_WEBHOOK=true`. By default, any CRD with validation rules is
included in the `ValidatingWebhookConfiguration`.

```yaml
- name: website
  webhooks:
    validation: true      # default: true when rules are declared
    operations:           # default: ["CREATE", "UPDATE"]
      - CREATE
      - UPDATE
```

| Field | Type | Default | Description |
|---|---|---|---|
| `validation` | bool | `true` | Include in `ValidatingWebhookConfiguration` |
| `operations` | `[]string` | `["CREATE", "UPDATE"]` | Operations that trigger the webhook. Valid values: `CREATE`, `UPDATE`, `DELETE`, `CONNECT` |

Set `validation: false` to use reconcile-time validation only for a specific
CRD, even when `ENABLE_ADMISSION_WEBHOOK=true` globally.

---

## Error reference

### Admission-time errors (kubectl output)

```
Error from server: admission webhook "validate.orkestra.konductor.io" denied the request:
[orkestra] validation failed: field "spec.image": image must be from the myorg registry (got: "nginx:1.25")
```

Multiple violations:

```
Error from server: admission webhook "validate.orkestra.konductor.io" denied the request:
[orkestra] validation failed: field "spec.image": image must be from the myorg registry (got: "nginx:1.25"); field "spec.replicas": replicas cannot exceed 10 (got: "15")
```

Warnings (action: warn):

```
Warning: orkestra: field "metadata.labels.team": all resources should declare a team owner
website.demo.orkestra.io/my-site created
```

### Reconcile-time events (kubectl describe)

```
Events:
  Type     Reason               Message
  ----     ------               -------
  Warning  WebsiteValidationDenied  validation failed: field "spec.image": image must be from the myorg registry (got: "nginx:1.25")
```

### Startup errors

```
error: ENABLE_ADMISSION_WEBHOOK requires ENABLE_CONVERSION — set ENABLE_CONVERSION=true
to start the HTTPS server that serves /validate and /mutate
```

```
error: webhook registration: reading CA bundle: TLS_CERT is required for webhook
registration — set TLS_CERT to the path of the serving certificate
```

```
error: webhook registration: validating: failed to create ValidatingWebhookConfiguration:
  clusterroles.rbac.authorization.k8s.io "orkestra" is forbidden: ...
  admissionregistration.k8s.io/validatingwebhookconfigurations requires get, create, update
```

---

## Complete example

```yaml
- name: website
  apiTypes:
    group: demo.orkestra.io
    version: v1alpha1
    kind: Website
    plural: websites

  webhooks:
    validation: true
    operations: ["CREATE", "UPDATE"]

  operatorBox:
    validation:
      # deny — blocks at admission and halts reconciliation
      - field: spec.image
        prefix: "registry.myorg.io/"
        message: "images must come from the internal registry"
        action: deny

      - field: spec.replicas
        max: "20"
        message: "replicas cannot exceed 20"
        action: deny

      # warn — surfaces advisory without blocking
      - field: metadata.labels.team
        operator: exists
        message: "all resources should declare a team owner label"
        action: warn

      - field: metadata.labels.cost-center
        operator: exists
        message: "cost-center label required for billing attribution"
        action: warn

      # numeric check — deny
      - field: spec.replicas
        min: "1"
        message: "replicas must be at least 1"
        # action omitted — defaults to deny
```
