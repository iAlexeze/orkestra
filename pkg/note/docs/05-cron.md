# 05 — Cron Notes

Cron notes work with cron schedule expressions. They cover three problems: extracting individual fields from a string expression, converting structured map inputs to a string, and producing human-readable descriptions.

## The two schedule shapes

Users may declare schedules as a plain string or as a structured object:

```yaml
# String
spec:
  schedule: "*/5 * * * *"

# Structured map
spec:
  schedule:
    minute: "*/5"
    hour: "*"
    dayOfMonth: "*"
    month: "*"
    dayOfWeek: "*"
```

Cron notes handle both, and `normalize:` is the recommended place to collapse them into a single canonical string before any downstream phase uses the field. See the [normalize documentation](../../reconciler/docs/06-normalize.md).

## Cron expression format

Standard five-field cron: `minute hour dayOfMonth month dayOfWeek`

```
*/5 * * * *
 │  │ │   │ └── day of week  (0-7, 0=Sun, 7=Sun, or names)
 │  │ │   └──── month        (1-12 or names)
 │  │ └──────── day of month (1-31)
 │  └────────── hour         (0-23)
 └───────────── minute       (0-59)
```

## Reference

### `cronExpr`

Build a five-field cron expression from five named parts. Empty parts default to `*`.

```yaml
# value: "{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}"
# minute="*/1", hour="*", dom="*", month="*", dow="*" → "*/1 * * * *"
```

The canonical way to reconstruct a cron string from a structured map's individual fields when you access them separately.

---

### `cronFromMap`

Convert a structured `map[string]interface{}` to a five-field cron string in one step. Reads keys `minute`, `hour`, `dayOfMonth`, `month`, `dayOfWeek`; defaults absent keys to `*`.

```yaml
normalize:
  spec:
    schedule: >
      {{ if typeMap .spec.schedule }}
        {{ cronFromMap .spec.schedule }}
      {{ else }}
        {{ cronNormalize .spec.schedule }}
      {{ end }}
```

```
{minute: "*/5", hour: "*", dayOfMonth: "*", month: "*", dayOfWeek: "*"}
→ "*/5 * * * *"
```

Use `cronFromMap` when the whole map is available. Use `cronExpr` when you need to access individual fields.

---

### `cronNormalize`

Normalize a cron string: expand `@`-macros, trim whitespace, ensure exactly five fields. Returns `"* * * * *"` for empty or malformed input.

```yaml
# value: "{{ cronNormalize .spec.schedule }}"
# "@daily"     → "0 0 * * *"
# "@hourly"    → "0 * * * *"
# "*/5 * * * *" → "*/5 * * * *"  (unchanged, already valid)
# ""            → "* * * * *"
```

Use `cronNormalize` in the `else` branch of a `typeMap` check — it handles string schedules that may carry macros or extra whitespace.

---

### `cronDescribe`

Return a human-readable description of a cron expression. Useful in `status.fields` for the Control Center UI.

```yaml
- path: scheduleDescription
  value: "{{ cronDescribe .spec.schedule }}"
```

| Expression | Description |
|------------|-------------|
| `*/5 * * * *` | Every 5 minutes |
| `0 * * * *` | Every hour |
| `0 2 * * *` | At 2:0 every day |
| `0 2 * * 1` | At 02:00 on Mondays |
| `@daily` | At 0:0 every day (after normalization) |

---

### `cronValid`

Return `true` when the expression is structurally valid (five fields present after macro expansion). Does not validate field ranges.

```yaml
validation:
  rules:
    - field: spec.schedule
      operator: custom
      value: "{{ cronValid .spec.schedule }}"
      message: "spec.schedule must be a valid cron expression"
      action: deny
```

---

### `cronMinute` / `cronHour` / `cronDom` / `cronMonth` / `cronDow`

Extract a single field by position from a cron string. Returns `*` for empty input. Returns an error for strings that don't expand to five fields.

```yaml
# value: "{{ cronMinute \"*/5 2 * * 1\" }}"   → "*/5"
# value: "{{ cronHour   \"*/5 2 * * 1\" }}"   → "2"
# value: "{{ cronDom    \"0 0 15 * *\" }}"    → "15"
# value: "{{ cronMonth  \"0 0 1 6 *\" }}"     → "6"
# value: "{{ cronDow    \"0 0 * * 1\" }}"     → "1"
```

These are useful in conversion paths when you need to split a string schedule into its parts for storage in a structured format.

---

### `cronField`

Extract a field by index (0–4). The general form of the five named extractors above.

```yaml
# value: "{{ cronField .spec.schedule 0 }}"   → minute field
# value: "{{ cronField .spec.schedule 3 }}"   → month field
```

---

## Supported `@`-macros

| Macro | Expands to | Meaning |
|-------|-----------|---------|
| `@yearly` / `@annually` | `0 0 1 1 *` | Once a year, Jan 1 at midnight |
| `@monthly` | `0 0 1 * *` | Once a month, 1st at midnight |
| `@weekly` | `0 0 * * 0` | Once a week, Sunday at midnight |
| `@daily` / `@midnight` | `0 0 * * *` | Once a day at midnight |
| `@hourly` | `0 * * * *` | Once an hour at minute 0 |

---

## Quick reference

| Note | Signature | Returns |
|------|-----------|---------|
| `cronExpr` | `(minute, hour, dom, month, dow string)` | `string` |
| `cronFromMap` | `(m map[string]interface{})` | `string` |
| `cronNormalize` | `(expr string)` | `string` |
| `cronDescribe` | `(expr string)` | `string` |
| `cronValid` | `(expr string)` | `bool` |
| `cronMinute` | `(expr string)` | `string` |
| `cronHour` | `(expr string)` | `string` |
| `cronDom` | `(expr string)` | `string` |
| `cronMonth` | `(expr string)` | `string` |
| `cronDow` | `(expr string)` | `string` |
| `cronField` | `(expr string, pos int)` | `string` |

---

**Next →** [06 — Random Notes](06-random.md)
