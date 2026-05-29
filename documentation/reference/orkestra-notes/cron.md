# Cron Notes

Parse, convert, and reconstruct cron schedule expressions. Handles five-field expressions and `@`-macros (`@hourly`, `@daily`, `@weekly`, `@monthly`, `@yearly`).

## The three conversion notes

| Note | Input | Output | Use in |
|------|-------|--------|--------|
| `cronToMap` | `string` | `map` (via sentinel) | conversion path spec — produces nested object |
| `cronFromMap` | `map` only | `string` — errors on string | `onReconcile` behind `typeOf: map` gate |
| `cronFromAny` | `map` or `string` | `string` | normalize, status, conversion — unknown shape |

```yaml
# v1 (cron string) → v2 (structured map)
- from: v1
  to: v2
  spec:
    schedule: "{{ cronToMap .spec.schedule }}"

# v2 → v1: input may be map or legacy flat string
- from: v2
  to: v1
  spec:
    schedule: "{{ cronFromAny .spec.schedule }}"

# normalize block: collapse either shape to canonical string
normalize:
  spec:
    schedule: "{{ cronFromAny .spec.schedule }}"

# onReconcile Path B — input guaranteed a map by the when: gate
- name: "{{ .metadata.name }}"
  schedule: "{{ cronFromMap .spec.schedule }}"
  when:
    - field: spec.schedule
      operator: typeOf
      value: map
```

## Validation and normalization

| Note | Signature | Returns |
|------|-----------|---------|
| `cronValid` | `string` | `bool` — structurally valid (five fields present) |
| `cronNormalize` | `string` | `string` — expanded macros, exactly five fields |
| `cronDescribe` | `string` | `string` — human-readable ("Every 5 minutes") |

```yaml
- field: spec.schedule
  value: "{{ cronValid .spec.schedule }}"
  message: "spec.schedule must be a valid cron expression"
  action: deny

- path: scheduleDescription
  value: "{{ cronDescribe .spec.schedule }}"
```

## Field extraction

| Note | Signature | Returns |
|------|-----------|---------|
| `cronMinute` | `string` | minute field |
| `cronHour` | `string` | hour field |
| `cronDom` | `string` | day-of-month field |
| `cronMonth` | `string` | month field |
| `cronDow` | `string` | day-of-week field |
| `cronField` | `string, int` | field at position 0–4 |
| `cronExpr` | `min hr dom mon dow string` | canonical five-field string |

```yaml
# Extract a single field
- path: scheduleMinute
  value: "{{ cronMinute .spec.schedule }}"

# Build from five explicit parts
schedule: "{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}"
```

## Supported @-macros

| Macro | Expands to | Meaning |
|-------|-----------|---------|
| `@yearly` / `@annually` | `0 0 1 1 *` | Once a year, Jan 1 at midnight |
| `@monthly` | `0 0 1 * *` | Once a month, 1st at midnight |
| `@weekly` | `0 0 * * 0` | Once a week, Sunday at midnight |
| `@daily` / `@midnight` | `0 0 * * *` | Once a day at midnight |
| `@hourly` | `0 * * * *` | Once an hour at minute 0 |
