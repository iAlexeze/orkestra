# note

**Pure transformation functions available in every Orkestra template expression.**

The name is intentional. In music, notes are the atomic units from which
everything is composed — precise, combinable, universally understood. In
Orkestra, notes serve the same role: small transformation functions that
compose into complete operator behavior, available wherever a template
expression is evaluated.

---

## Where notes work

Every position where Orkestra evaluates a `{{ }}` expression:

```yaml
# Version conversion paths
conversion:
  paths:
    - from: v1
      to: v2
      spec:
        schedule:
          minute: "{{ cronMinute .spec.schedule }}"
          hour:   "{{ cronHour   .spec.schedule }}"

# Status field declarations
status:
  fields:
    - path: environment
      value: "{{ toLower .spec.environment }}"
    - path: scheduleDisplay
      value: "{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}"

# Mutation defaults and overrides
mutation:
  rules:
    - field: spec.replicas
      default: "{{ max .spec.minReplicas 2 }}"
    - field: spec.environment
      default: "{{ default .spec.env "development" }}"

# Conditional when expressions
when:
  - field: spec.tier
    equals: "enterprise"

# onCreate and onReconcile templates
onCreate:
  cronJobs:
    - schedule: "{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}"
      image: "{{ trimPrefix .spec.image "docker.io/" }}"
```

---

## What a note is

A note is a Go function registered in `note.Map()`. It must be:

- **Pure** — same input always produces same output
- **Safe** — handles empty input, never panics
- **Stateless** — no I/O, no external calls, no shared state
- **Typed** — accepts plain Go types: `string`, `int64`, `float64`, `bool`, `interface{}`

Notes are not hooks. Hooks are for external API calls, database writes, and
side effects. Notes are for data transformation — turning one value into another.

---

## Catalogue

### Cron

For working with cron schedule expressions. Handles standard five-field
expressions and @-macros (`@hourly`, `@daily`, `@weekly`, `@monthly`,
`@yearly`, `@annually`, `@midnight`).

| Note | Signature | Description |
|---|---|---|
| `cronMinute` | `(expr string) (string, error)` | Extract minute field |
| `cronHour` | `(expr string) (string, error)` | Extract hour field |
| `cronDom` | `(expr string) (string, error)` | Extract day-of-month field |
| `cronMonth` | `(expr string) (string, error)` | Extract month field |
| `cronDow` | `(expr string) (string, error)` | Extract day-of-week field |
| `cronField` | `(expr string, pos int) (string, error)` | Extract field at position 0-4 |
| `cronExpr` | `(min, hr, dom, mon, dow string) string` | Reconstruct cron string |
| `cronValid` | `(expr string) bool` | Validate cron structure |

**The version conversion pattern:**

```yaml
# v1 (string) → v2 (structured)
- from: v1
  to: v2
  spec:
    schedule:
      minute: "{{ cronMinute .spec.schedule }}"
      hour:   "{{ cronHour   .spec.schedule }}"
      dayOfMonth: "{{ cronDom  .spec.schedule }}"
      month:  "{{ cronMonth  .spec.schedule }}"
      dayOfWeek:  "{{ cronDow  .spec.schedule }}"

# v2 (structured) → v1 (string)
- from: v2
  to: v1
  spec:
    schedule: "{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}"
```

Round-trip: `"0 2 * * 1"` → `{minute:"0", hour:"2", dom:"*", month:"*", dow:"1"}` → `"0 2 * * 1"` ✓

### String

| Note | Signature | Example |
|---|---|---|
| `toLower` | `(s string) string` | `{{ toLower .spec.environment }}` |
| `toUpper` | `(s string) string` | `{{ toUpper .spec.region }}` |
| `trimSpace` | `(s string) string` | `{{ trimSpace .spec.image }}` |
| `trimPrefix` | `(s, prefix string) string` | `{{ trimPrefix .spec.image "docker.io/" }}` |
| `trimSuffix` | `(s, suffix string) string` | `{{ trimSuffix .spec.image ":latest" }}` |
| `replace` | `(s, old, new string) string` | `{{ replace .metadata.name "_" "-" }}` |
| `contains` | `(s, substr string) bool` | `{{ contains .spec.image "myorg/" }}` |
| `hasPrefix` | `(s, prefix string) bool` | `{{ hasPrefix .spec.image "myorg/" }}` |
| `hasSuffix` | `(s, suffix string) bool` | `{{ hasSuffix .spec.image ":latest" }}` |
| `split` | `(s, sep string) []string` | `{{ index (split .spec.tags ",") 0 }}` |
| `join` | `(slice []string, sep string) string` | `{{ join .spec.hosts "," }}` |
| `camelToKebab` | `(s string) string` | `{{ camelToKebab "WebsiteOp" }}` → `"website-op"` |
| `truncate` | `(s string, n int) string` | `{{ truncate .metadata.name 63 }}` |

### Math

| Note | Signature | Example |
|---|---|---|
| `add` | `(a, b interface{}) (interface{}, error)` | `{{ add .spec.basePort 1000 }}` |
| `sub` | `(a, b interface{}) (interface{}, error)` | `{{ sub .spec.port 80 }}` |
| `mul` | `(a, b interface{}) (interface{}, error)` | `{{ mul .spec.replicas 2 }}` |
| `div` | `(a, b interface{}) (interface{}, error)` | `{{ div .spec.total 3 }}` |
| `mod` | `(a, b interface{}) (interface{}, error)` | `{{ mod .spec.index 3 }}` |
| `min` | `(a, b interface{}) (interface{}, error)` | `{{ min .spec.replicas 10 }}` |
| `max` | `(a, b interface{}) (interface{}, error)` | `{{ max .spec.replicas 2 }}` |
| `clamp` | `(val, lo, hi interface{}) (interface{}, error)` | `{{ clamp .spec.replicas 1 20 }}` |
| `abs` | `(a interface{}) (interface{}, error)` | `{{ abs .spec.offset }}` |

### Type conversion

| Note | Signature | Example |
|---|---|---|
| `toInt` | `(v interface{}) (int64, error)` | `{{ toInt .spec.port }}` |
| `toFloat` | `(v interface{}) (float64, error)` | `{{ toFloat .spec.ratio }}` |
| `toBool` | `(v interface{}) (bool, error)` | `{{ toBool .spec.enabled }}` |
| `toString` | `(v interface{}) string` | `{{ toString .spec.replicas }}` |

### Conditional

These replace verbose `{{ if }}...{{ else }}...{{ end }}` constructs.

| Note | Signature | Example |
|---|---|---|
| `ternary` | `(cond, trueVal, falseVal interface{}) interface{}` | `{{ ternary .spec.debug "debug" "info" }}` |
| `coalesce` | `(vals ...interface{}) interface{}` | `{{ coalesce .spec.image .spec.default "nginx" }}` |
| `default` | `(val, def interface{}) interface{}` | `{{ default .spec.replicas 2 }}` |
| `empty` | `(v interface{}) bool` | `{{ empty .spec.image }}` |
| `notEmpty` | `(v interface{}) bool` | `{{ notEmpty .spec.image }}` |

---

## Integration

One line in `resolver.go`:

```go
tmpl, err := template.New("f").
    Option("missingkey=zero").
    Funcs(note.Map()).   // ← all notes available everywhere
    Parse(value)
```

`note.Map()` is a package-level variable — built once at init, no allocation
on each call.

---

## Writing a new note

1. Identify the domain: `cron`, `strings`, `math`, `types`, `conditional`
2. Add the function to the appropriate file
3. Register it in that file's `xxxNotes()` function
4. Document it in this README
5. Write a test

**Contract:**
- Handle empty/nil input — return a safe zero value, not a panic
- Return `(value, error)` for functions that can meaningfully fail
- Return just `value` for infallible functions
- Preserve numeric types — return `int64` not `string` for integer results

```go
// Example: slugify a string for use as a DNS label
func slugify(s string) string {
    s = strings.ToLower(s)
    s = strings.ReplaceAll(s, " ", "-")
    s = strings.ReplaceAll(s, "_", "-")
    return s
}

// Register in stringNotes():
"slugify": slugify,
```

---

## Notes vs hooks — the boundary

| What you need | Use |
|---|---|
| Transform a field value | Note |
| Parse a known format (cron, semver, duration) | Note |
| Compute a value from spec fields | Note |
| Convert between API versions | Notes in conversion paths |
| Apply math or string manipulation | Note |
| Express a conditional value | Note |
| Call an external HTTP API | Hook |
| Write to a database | Hook |
| Send a notification or event | Hook |
| Read from another cluster | Hook |
| Complex stateful orchestration | Hook |

If your hook contains only pure data transformation, extract it to a note.
Keep hooks for what genuinely requires I/O.
