# Field Translation

The serve layer can transform what a caller submits before it reaches the CRD. The caller speaks a human vocabulary — flat strings, friendly names. The CRD speaks its own schema. `serve.fields.value` and `serve.fields.values` bridge the two without exposing either side to the other.

---

## The problem

A CRD's internal schema often doesn't match what a developer wants to type. A cron schedule might be a structured object (`schedule.minute`, `schedule.hour`, …) inside the CRD, but callers think in cron strings (`"0 2 * * 1-5"`). A registry might be a compound path (`spec.image.registry`, `spec.image.tag`) but callers supply a single image reference.

Without translation, the platform team either:
- Forces callers to submit the CRD's internal shape (leaks internals)
- Adds a conversion webhook or normalize step in the reconciler (extra moving parts)

Field translation does it at the serve layer, in the Katalog, with no extra infrastructure.

---

## How it works

### Single transform (`value`)

One intent field → one spec field, transformed:

```yaml
serve:
  fields:
    image:
      value: '{{ trimPrefix "docker.io/" .value }}'
      # caller submits: "docker.io/myorg/app:latest"
      # spec receives:  "myorg/app:latest"
```

`.value` is the raw submitted value. The result replaces it in the spec.

### Fanout (`values`)

One intent field → multiple spec fields:

```yaml
serve:
  fields:
    schedule:
      label: "Schedule (cron)"
      hint: 'e.g. "*/5 * * * *"'
      required: true
      values:
        schedule.minute:     '{{ cronMinute .value }}'
        schedule.hour:       '{{ cronHour   .value }}'
        schedule.dayOfMonth: '{{ cronDom    .value }}'
        schedule.month:      '{{ cronMonth  .value }}'
        schedule.dayOfWeek:  '{{ cronDow    .value }}'
```

The caller submits `"0 2 * * 1-5"`. The gateway fans it out to five spec fields. The CRD never sees the string. The caller never sees the struct.

---

## Intent gating

Translation happens in stage 3 (CR construction). Validation rules in stage 5 can still read the original submitted value via `.request.<field>` — before translation happened.

This lets you validate the caller's vocabulary, not the CRD's:

```yaml
validation:
  rules:
    - field: "{{ cronValid .request.schedule }}"
      equals: true
      link: schedule
      fires:
        reconcile: false
      message: 'schedule must be a valid cron expression'
      action: deny

    - field: spec.schedule
      operator: exists
      message: "schedule is required"
      action: deny
```

`fires.reconcile: false` limits the `cronValid` check to the admission path — it reads `.request.schedule`, the raw intent string, which only exists during a serve-layer submit. The second rule covers reconcile: it checks the structured field the fanout produced, without any dependency on `.request`.

---

## What the caller sees vs. what the CRD sees

| Caller submits | CRD receives |
|---|---|
| `schedule: "0 2 * * 1-5"` | `schedule.minute: "0"`, `schedule.hour: "2"`, `schedule.dayOfMonth: "*"`, `schedule.month: "*"`, `schedule.dayOfWeek: "1-5"` |
| `image: "docker.io/myorg/app:latest"` | `image: "myorg/app:latest"` |

Neither side knows about the other's vocabulary. The Katalog is the contract between them.

---

## Testing locally

```bash
ork serve validate --full      # confirms target and field config
ork serve play -f katalog.yaml --token dev -i intent.yaml
```

Stage 3 of `ork serve play` shows the built CR with all expressions evaluated, before any cluster is involved.

---

## Try it

```bash
ork init --pack use-cases/crd-conversion
```

The `with-serve-translation` variant demonstrates `values` fanout of a cron string to a structured schedule object.

## See also

- **Reference:** → [serve — field translation](../../reference/schema/02-katalog/22-serve-field-translation.md) — `value`, `values`, `.value`, `.request` reference
- **Target mode:** → [02-target-mode.md](02-target-mode.md) — how the serve layer works end-to-end
