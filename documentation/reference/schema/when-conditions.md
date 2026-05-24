# when / anyOf conditions

Conditions control whether a resource template field is written during reconciliation.
Used inside `operatorBox` and `autoscale`.

```yaml
operatorBox:
  when:
    - field: spec.replicas
      operator: gt
      value: "1"
      valueType: int

  anyOf:
    - field: spec.mode
      equals: production
    - field: spec.mode
      equals: staging
```

## Semantics

| Block | Behaviour |
|-------|-----------|
| `when` | AND — all conditions must be true |
| `anyOf` | OR — at least one condition must be true |

Both can be combined. The overall result is: `when` AND `anyOf`.

## Condition fields

| Field | Required | Description |
|-------|----------|-------------|
| `field` | yes | Dot-notation path into the CR (e.g. `spec.replicas`, `spec.config.mode`) |
| `operator` | yes* | Comparison operator (see below) |
| `value` | yes* | Value to compare against |
| `valueType` | no | `string` (default), `int`, `float`, `bool` |

*Use either the `operator`+`value` form or a shorthand form — not both.

## Operators

| Operator | Shorthand | Description |
|----------|-----------|-------------|
| `eq` | `equals` | Field equals value |
| `neq` | `notEquals` | Field does not equal value |
| `gt` | `greaterThan` | Field is greater than value (numeric) |
| `lt` | `lessThan` | Field is less than value (numeric) |
| `gte` | `min` | Field is greater than or equal to value (numeric) |
| `lte` | `max` | Field is less than or equal to value (numeric) |
| `contains` | `contains` | Field contains the substring |
| `prefix` | `prefix` | Field starts with the value |
| `suffix` | `suffix` | Field ends with the value |

## Shorthand form

```yaml
when:
  - field: spec.engine
    equals: postgres         # shorthand for operator: eq, value: postgres

  - field: spec.replicas
    greaterThan: "0"         # shorthand for operator: gt
    valueType: int
```

## Explicit form

```yaml
when:
  - field: spec.replicas
    operator: gt
    value: "3"
    valueType: int
```

---
