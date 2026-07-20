# validation

Declarative validation rules evaluated at admission time (via webhook) and at reconcile time.
Declared on a CRDEntry or inside a Motif's `admission` block.

```yaml
validation:
  rules:
    - field: spec.replicas
      operator: lte
      value: "10"
      valueType: int
      message: "replicas must not exceed 10"

    - field: spec.engine
      equals: postgres
      message: "only postgres engine is supported"
      action: deny

    - field: spec.image
      prefix: "myregistry.example.com/"
      message: "image must come from the internal registry"
      action: warn
```

## `validation.include`

Pull rules from an external file to keep the Katalog compact. The file contains a `rules:` key with the same list structure as inline rules.

```yaml
validation:
  include: ./admission/apprequest.yaml   # relative to the katalog file
  rules:
    - field: spec.tier                   # appended after included rules
      operator: in
      value: "free,pro,enterprise"
      action: deny
```

`admission/apprequest.yaml`:

```yaml
rules:
  - field: spec.team
    operator: exists
    message: "spec.team is required"
    action: deny
  - field: spec.environment
    operator: in
    value: "staging,production"
    message: "spec.environment must be staging or production"
    action: deny
```

Included rules come first. Inline `rules:` append after. The `include:` path is resolved relative to the katalog file's directory — the katalog can be run from any working directory. The field is cleared from the runtime bundle after expansion.

## `validation.rules`

Each rule describes one check. Rules are evaluated in order.

| Field | Required | Description |
|-------|----------|-------------|
| `field` | yes | Dot-notation path in the CR (e.g. `spec.replicas`, `spec.config.engine`) |
| `message` | yes | Error or warning message — shown in events, webhook response, and logs |
| `action` | no | `deny` (default) — reject; `warn` — allow but log a warning |
| `operator` + `value` | yes* | Explicit comparison (see operators) |
| `valueType` | no | `string` (default), `int`, `float`, `bool` |

*Use either an operator+value pair or a shorthand field.

## Operators

| Shorthand | Operator | Description |
|-----------|----------|-------------|
| `equals` | `eq` | Field equals value |
| `notEquals` | `neq` | Field does not equal value |
| `prefix` | `prefix` | Field starts with value |
| `suffix` | `suffix` | Field ends with value |
| `contains` | `contains` | Field contains substring |
| `min` | `gte` | Field is greater than or equal (numeric) |
| `max` | `lte` | Field is less than or equal (numeric) |
| `greaterThan` | `gt` | Field is greater than (numeric) |
| `lessThan` | `lt` | Field is less than (numeric) |

## `action`

| Value | Effect |
|-------|--------|
| `deny` (default) | Webhook returns a rejection; reconcile fails with an error. |
| `warn` | Webhook allows the operation; a warning is logged. |

## When validation runs

- **At admission**: if `security.webhooks.admission.enabled: true` and the CRD's `webhooks.validation: true`.
- **At reconcile**: always — even without a webhook, rules are checked during each cycle.

---
