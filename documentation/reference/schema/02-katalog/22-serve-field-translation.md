# serve — Field Translation

Field translation lets the serve layer transform what the caller submits before it reaches the CRD. The caller speaks a human vocabulary; the CRD speaks whatever schema it defines. The Katalog bridges them.

Two fields on `serve.fields.<name>` control translation:

| Field | Description |
|-------|-------------|
| `value` | Single template expression — transforms the submitted value before writing to one spec path. |
| `values` | Fanout map — one intent field fans out to multiple CRD spec paths. |

Both are mutually exclusive with each other. `ork validate` rejects a field that declares both.

---

## `value` — single transform

Use `value` when one intent field maps to one CRD spec path, but needs to be transformed before writing:

```yaml
serve:
  fields:
    schedule:
      path: job.cronExpression
      value: '{{ normalise .value }}'   # transform before writing
```

`.value` is the raw submitted value. The result of the expression is written to `spec.<path>` (or `spec.<name>` when `path` is absent).

```json
// caller submits
{ "target": "job", "name": "cleanup", "schedule": "every day at 2am" }

// spec written to CR
{ "job": { "cronExpression": "0 2 * * *" } }
```

---

## `values` — fanout

Use `values` when one intent field needs to populate multiple CRD spec paths. Keys are dot-notation spec paths; values are template expressions.

```yaml
serve:
  fields:
    schedule:
      label: "Schedule (cron)"
      hint: 'Standard cron expression, e.g. "*/5 * * * *"'
      required: true
      values:
        schedule.minute:     '{{ cronMinute .value }}'
        schedule.hour:       '{{ cronHour   .value }}'
        schedule.dayOfMonth: '{{ cronDom    .value }}'
        schedule.month:      '{{ cronMonth  .value }}'
        schedule.dayOfWeek:  '{{ cronDow    .value }}'
```

The caller submits a flat cron string. The serve layer fans it out to the five structured fields before the CR reaches the API server:

```json
// caller submits
{ "target": "cronjob-tutorial", "name": "daily-backup", "schedule": "0 2 * * 1-5" }

// spec written to CR
{
  "schedule": {
    "minute": "0",
    "hour": "2",
    "dayOfMonth": "*",
    "month": "*",
    "dayOfWeek": "1-5"
  }
}
```

All keys in `values` must be dot-notation paths. A flat key (no dot) is a validation error — use `value` instead.

---

## `.value` — the submitted field value

Both `value` and `values` expressions receive `.value` — the raw value the caller submitted for that field. This is distinct from `.spec.<name>`, which refers to what's already on the CR.

```yaml
values:
  schedule.minute: '{{ cronMinute .value }}'   # .value = "0 2 * * 1-5"
```

---

## `.request.<field>` — cross-field reads

All translation expressions also receive `.request` — the full raw intent payload. Use it to read other submitted fields:

```yaml
values:
  schedule.minute: '{{ cronMinute .request.schedule }}'
  job.image:       '{{ .request.image }}'
```

`.request` is also available in `validation.rules` for intent gating — checking the original submitted value before translation:

```yaml
validation:
  rules:
    - field: "{{ cronValid .request.schedule }}"
      equals: true
      link: schedule
      fires:
        reconcile: false
      message: 'schedule must be a valid cron expression (e.g. "*/5 * * * *")'
      action: deny

    - field: spec.schedule
      operator: exists
      message: "schedule is required"
      action: deny
```

`fires.reconcile: false` limits the `cronValid` rule to the admission path. `.request` is only present at the serve-layer boundary; the reconciler sees the translated CR with no raw intent. The second rule fires at both admission and reconcile to ensure the structured field is present regardless of how the CR arrived.

---

## Gate-only fields

A field with no `path`, `value`, or `values` is written to `spec.<name>` unchanged (existing behavior). It's still accessible in validation rules via `.request.<name>` — useful when you want to gate on a submitted value without writing it to the CRD at all.

---

## Combining with `path`

`value` and `path` can be combined — `path` sets the destination, `value` transforms the content:

```yaml
fields:
  schedule:
    path: job.cronExpression
    value: '{{ normalise .value }}'
  # caller submits schedule → spec.job.cronExpression = normalise(schedule)
```

`values` ignores `path` — each key in the map is its own destination.

---

## Validation at `ork validate` time

`ork validate` checks:

- `value` and `values` are mutually exclusive — both on the same field is an error
- All `values` keys must be dot-notation paths (no flat keys)
- Template expressions in `value` and all `values` entries compile without errors

---

## Testing locally

`ork serve play` runs the full translation pipeline without a cluster:

```bash
ork serve play -f katalog.yaml --token dev -i intent.yaml
```

Stage 3 shows the built CR with all fanout expressions evaluated. Stage 5 shows whether validation rules (including `request.*` gates) pass.

→ [serve](20-serve.md) — full `serve` field reference  
→ [21-serve-nested-spec.md](21-serve-nested-spec.md) — `path` for nested spec mapping

## Try it

```bash
ork init --pack use-cases/crd-conversion
```

The `with-serve-translation` variant demonstrates `values` fanout of a cron string to a structured schedule object, with a `fires.reconcile: false` `cronValid` rule at admission and a plain `spec.schedule` exists-check at reconcile.
