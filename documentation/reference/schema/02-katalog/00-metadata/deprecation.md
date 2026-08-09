# metadata.deprecation

The `deprecation` block marks a Katalog as deprecated and optionally schedules its end of life. The deprecation state is surfaced at every touchpoint:

- **`ork push`** — shown to the author immediately after the katalog is validated, before the artifact is uploaded. The author sees exactly what consumers will see.
- **`ork validate`** — shown during pre-flight checks, without blocking.
- **`ork inspect`** — shown before pulling, annotating each version in `--versions` and the full detail view.
- **`ork pull`** — shown after the artifact is cached or extracted.

The `accept` sub-block is the operator's explicit acknowledgement that a deprecated or EOL pattern is intentionally kept running — without it, `ork run` and `ork gate` refuse to start.

```yaml
metadata:
  name: database-operator
  deprecation:
    message: "Use postgres-operator:v2.0.0 — improved connection pooling and status reporting"
    migratedTo: postgres-operator:v2.0.0
    timeline:
      from: "2026-09-01"
      to:   "2027-03-01"
    accept:
      beforeEol: true   # required to run during the deprecation warning window
      eol: true         # required to run after the end-of-life date
```

---

## Fields

### `message` (required)


Human-readable explanation of the deprecation. Shown in every warning and in `ork inspect` output. Required when the `deprecation` block is present — `ork validate` fails if it is missing.

### `migratedTo`

The replacement pattern in `<name>:<version>` format. Shown alongside the deprecation message to give consumers a clear migration target.

### `timeline`

An optional schedule for the deprecation window. Both sub-fields are `YYYY-MM-DD` strings. Omitting `timeline` entirely declares the Katalog deprecated without a scheduled EOL.

#### `timeline.from`

The date the deprecation window opens. Before this date, the Katalog is declared deprecated but no timeline warning is shown. On and after this date, `ork inspect` and `ork pull` display the deprecation warning with the number of days until EOL.

#### `timeline.to`

The end-of-life date. On and after this date, the artifact is shown as **END OF LIFE** (red ✗). The artifact remains in the registry and remains pullable — this is a display state, not a deletion.

`from` must be strictly before `to`. `ork validate` rejects equal or reversed dates.

### `accept`

Explicit operator acknowledgement that a deprecated or EOL Katalog is intentionally kept running. Without the appropriate `accept` field set, `ork run` and `ork gate` refuse to start.

#### `accept.beforeEol`

Set to `true` to allow `ork run` and `ork gate` to start while the pattern is in the deprecation warning window (state `"warning"`). A deprecation warning is still printed on every start.

#### `accept.eol`

Set to `true` (alongside `beforeEol: true`) to allow `ork run` and `ork gate` to start after the pattern has passed its end-of-life date (state `"eol"`). `eol: true` alone is not sufficient — `beforeEol` must also be true.

Setting `accept.eol: true` is a strong signal in code review that a team has consciously decided to run a dead pattern. It should be accompanied by a tracked migration plan.

---

## Display states

The display state is computed by comparing today's date against the timeline:

| Condition | State | Display |
|-----------|-------|---------|
| No `timeline` declared | `warning` | ⚠ Deprecated — `message` |
| `today < from` | `warning` | ⚠ Deprecated — `message` |
| `from ≤ today < to` | `warning` | ⚠ Deprecated — `message` · N days until EOL (YYYY-MM-DD) |
| `today ≥ to` | `eol` | ✗ END OF LIFE — `message` |

`ork inspect --versions` uses the same logic to annotate each version row.

---

## Enforcement at runtime startup

`ork run` and `ork gate` check the deprecation policy after validation. The rules:

| State | `accept` required | Behaviour |
|-------|-------------------|-----------|
| `none` (before `from`, or no block) | — | No gate |
| `warning` (deprecated, before EOL) | `beforeEol: true` | Blocked without it; warning still printed when set |
| `eol` (past `to`) | `beforeEol: true` + `eol: true` | Blocked without both; `eol` alone is not sufficient |

The error message always shows the deprecation message, the migration target if set, and the exact YAML to add.

## Validation rules

`ork validate` enforces:

- `message` is non-empty when `deprecation` is declared
- `timeline.from` and `timeline.to`, when present, must be valid `YYYY-MM-DD` dates
- `timeline.from` must be strictly before `timeline.to`

`ork validate` does **not** enforce `accept` — it is a pre-flight tool and must not block. Enforcement is at runtime startup only.

---

## OCI annotations

`ork push` serialises the deprecation block to OCI manifest annotations. Consumers read these via `ork inspect` without pulling the artifact.

| Annotation | Value |
|-----------|-------|
| `io.orkestra.deprecated` | `"true"` |
| `io.orkestra.deprecated.message` | The deprecation message |
| `io.orkestra.katalog.deprecated.migrated_to` | `migratedTo` value |
| `io.orkestra.katalog.deprecated.timeline_from` | `timeline.from` value |
| `io.orkestra.katalog.deprecated.timeline_to` | `timeline.to` value |

---

## Example: immediate deprecation

```yaml
metadata:
  name: legacy-worker
  deprecation:
    message: "Replaced by task-runner which uses the Jobs API"
    migratedTo: task-runner:v1.0.0
```

## Example: scheduled EOL

```yaml
metadata:
  name: legacy-worker
  deprecation:
    message: "Replaced by task-runner which uses the Jobs API"
    migratedTo: task-runner:v1.0.0
    timeline:
      from: "2026-09-01"   # warning with countdown starts
      to:   "2027-03-01"   # shown as END OF LIFE after this date
```

## Example: accepted deprecation (running during warning window)

```yaml
metadata:
  name: legacy-worker
  deprecation:
    message: "Replaced by task-runner which uses the Jobs API"
    migratedTo: task-runner:v1.0.0
    timeline:
      from: "2026-09-01"
      to:   "2027-03-01"
    accept:
      beforeEol: true   # migration in progress — team is aware
```

## Example: accepted EOL (running past end-of-life)

```yaml
metadata:
  name: legacy-worker
  deprecation:
    message: "Replaced by task-runner which uses the Jobs API"
    migratedTo: task-runner:v1.0.0
    timeline:
      from: "2026-09-01"
      to:   "2027-03-01"
    accept:
      beforeEol: true   # required alongside eol
      eol: true         # explicit acknowledgement — migration tracking required
```

---

## See also

- [metadata](./index.md) — full metadata block overview
- [ork validate](../../cli/06-validate.md) — offline validation including deprecation checks
- [ork inspect](../../cli/11-inspect.md) — displays deprecation state before pulling
- [Lifecycle](../../../concepts/lifecycle/index.md) — where deprecation fits in the pattern lifecycle
