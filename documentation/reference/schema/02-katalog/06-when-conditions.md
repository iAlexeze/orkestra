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

---

## Condition kinds

### Field conditions

Compare a dot-notation path into the CR against a value.

| Field | Required | Description |
|-------|----------|-------------|
| `field` | yes | Dot-notation path into the CR (e.g. `spec.replicas`, `spec.config.mode`) |
| `operator` | yes* | Comparison operator (see below) |
| `value` | yes* | Value to compare against |
| `valueType` | no | `string` (default), `int`, `float`, `bool` |

*Use either the `operator`+`value` form or a shorthand form — not both.

#### Operators

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

#### Shorthand form

```yaml
when:
  - field: spec.engine
    equals: postgres         # shorthand for operator: eq, value: postgres

  - field: spec.replicas
    greaterThan: "0"         # shorthand for operator: gt
    valueType: int
```

#### Explicit form

```yaml
when:
  - field: spec.replicas
    operator: gt
    value: "3"
    valueType: int
```

---

### Time conditions

Evaluate against the current wall clock (UTC). No `field:` key — mutually exclusive with field conditions.

#### `time:`

Active when the current clock time falls within the declared window.

```yaml
when:
  - time:
      after: "09:00"
      before: "18:00"
```

| Field | Required | Description |
|-------|----------|-------------|
| `after` | no | Active at or after this time. Format: `HH:MM` (24h, UTC). |
| `before` | no | Active at or before this time. Format: `HH:MM` (24h, UTC). |

Both `after` and `before` may be set together. Either may be omitted.

#### `dayOfWeek:`

Active on the declared set of days.

```yaml
when:
  - dayOfWeek:
      weekday: true          # Mon–Fri shorthand
```

```yaml
when:
  - dayOfWeek:
      weekend: true          # Sat–Sun shorthand
```

```yaml
when:
  - dayOfWeek:
      in: [Monday, Wednesday, Friday]
```

```yaml
when:
  - dayOfWeek:
      notIn: [Saturday, Sunday]
```

| Field | Mutually exclusive with | Description |
|-------|------------------------|-------------|
| `weekday` | `weekend`, `in`, `notIn` | `true` → active Mon–Fri; `false` → active Sat–Sun |
| `weekend` | `weekday`, `in`, `notIn` | `true` → active Sat–Sun; `false` → active Mon–Fri |
| `in` | `weekday`, `weekend`, `notIn` | Active on these days (full English names) |
| `notIn` | `weekday`, `weekend`, `in` | Active on all days except these |

Exactly one field must be set. Day names are case-insensitive (`monday`, `Monday`, and `MONDAY` are equivalent).

**Combined example** — business hours only (weekday AND time window):

```yaml
when:
  - time:
      after: "09:00"
      before: "18:00"
  - dayOfWeek:
      weekday: true
```

#### `cron:`

Active when a cron-defined window is open. The window opens at each cron fire and stays open for `duration`.

```yaml
when:
  - cron: "0 2 * * 0"
    duration: 2h
```

| Field | Required | Description |
|-------|----------|-------------|
| `cron` | yes | Standard cron expression (5-field: min hour dom month dow) |
| `duration` | no | How long the window stays open after each fire. Default: `60s`. |

---
