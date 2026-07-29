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
      notEquals: staging
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
| `equals` | `equals` | Field equals value |
| `notEquals` | `notEquals` | Field does not equal value |
| `contains` | `contains` | Field contains the substring |
| `notContains` | `notContains` | Field does not contain the substring |
| `prefix` | `prefix` | Field starts with the value |
| `suffix` | `suffix` | Field ends with the value |
| `regex` | `regex` | Field matches the value as an RE2 regular expression (Go's `regexp` syntax) |
| `exists` | `exists: true` | Field is present and non-empty |
| `notExists` | `notExists: true` | Field is absent or empty |
| `gt` | `greaterThan` | Field is numerically greater than value — **strict** |
| `lt` | `lessThan` | Field is numerically less than value — **strict** |
| `gte` | `min`, `greaterThanOrEqual` | Field is numerically greater than or equal to value |
| `lte` | `max`, `lessThanOrEqual` | Field is numerically less than or equal to value |
| `between` | `between` | Field is numerically within an inclusive range. Value is `"min,max"` |
| `notBetween` | `notBetween` | Field is numerically outside an inclusive range. Value is `"min,max"` |
| `in` | `in` | Field is one of a comma-separated list |
| `notIn` | `notIn` | Field is none of a comma-separated list |
| `unique` | — | Field value must be unique across all existing instances of this CRD. Works in both `validation.rules` and `when:`/`anyOf:`, but only at reconcile time — always passes at admission time (no live checker there) |
| `typeOf` / `typeMap` / `typeList` / `typeString` / `typeNumber` / `typeBool` / `typeNull` | — | Check the field's YAML type rather than its value. No shorthand — use `operator:` explicitly. |

`gt`/`lt` are strict (exclusive); use `gte`/`lte` (or the `min`/`max` shorthand) for an inclusive bound. `min`/`max` and `greaterThanOrEqual`/`lessThanOrEqual` resolve to the same `gte`/`lte` operators — `min`/`max` read better for a bound on a quantity (`min: "1"`), `greaterThanOrEqual`/`lessThanOrEqual` for a direct comparison. Same operators and shorthand as [validation.rules](07-validation.md#operators) — the `Condition` type is shared by both.

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

#### `between`, `in`, and `regex`

```yaml
when:
  - field: spec.replicas
    between: "1,10"          # inclusive range — 1 and 10 both pass

  - field: spec.tier
    in: "standard,premium"   # comma-separated list

  - field: spec.name
    regex: "^app-[a-z0-9-]+$"
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

## `negate`

Inverts the result of any condition. Applies to field conditions, `time:`, `dayOfWeek:`, and `cron:`.

```yaml
when:
  - dayOfWeek:
      weekday: true
    negate: true        # passes on weekends

  - time:
      after: "09:00"
      before: "18:00"
    negate: true        # passes outside the window
```

`negate` is a top-level field on the condition — it inverts whatever the condition evaluates to, regardless of kind.
