# Introducing Orkestra Notes

**Pure transformation functions for the declarative operator runtime.**

---

## The problem with template expressions

From the beginning, Orkestra's template expressions covered the common case:
`{{ .spec.image }}`, `{{ .metadata.name }}-deployment`, `{{ .spec.replicas }}`.
These are field references — reading a value from the CR and placing it somewhere.

Field references are not enough. Operators frequently need to *transform* values:

- Convert a cron string `"*/5 * * * *"` into named fields `{minute: "*/5", hour: "*"...}`
- Apply a default when a field is absent: replicas should be 2 if not declared
- Choose between two values based on a boolean: "Suspended" or "Active"
- Cap a numeric field: replicas should not exceed 10
- Reconstruct a string from parts: `minute + " " + hour + " * * *"`

Before notes, the answer was Go hooks. Writing a hook to apply a default, or to
extract one field from a cron string, required the full hook machinery: a typed
struct, `ork generate registry`, a registered function, a binary build.

The ceremony was disproportionate to the transformation.

---

## What a note is

A note is a pure named transformation function available in every Orkestra
template expression. Declared once in `pkg/note/`, registered in `note.Map()`,
available everywhere templates are evaluated — conversion paths, status fields,
mutation rules, when conditions, onCreate templates.

The name is deliberate. In Orkestra's musical identity, notes are the atomic
units from which everything is composed — small, precise, individually simple,
combinable into complex behavior. Operator behavior in Orkestra is composed
from template expressions the same way music is composed from notes.

A note is:

- **Pure** — same input always produces same output
- **Safe** — handles empty input, never panics
- **Stateless** — no I/O, no external calls, no shared state
- **Typed** — returns the correct Go type, not always a string

Notes are not hooks. Hooks exist for external API calls, database writes, and
operations with side effects. Notes exist for data transformation — turning
one value into another. The boundary is precise and permanent.

---

## The cron problem

The Kubebuilder CronJob tutorial changes `schedule` from a cron string to a
structured object between v1 and v2. The tutorial implements this with
handwritten conversion functions in Go. Here is the ConvertTo function:

```go
func (src *CronJob) ConvertTo(dstRaw conversion.Hub) error {
    dst := dstRaw.(*v2.CronJob)
    
    schedParts := strings.Split(src.Spec.Schedule, " ")
    if len(schedParts) != 5 {
        return fmt.Errorf("invalid schedule format: %s", src.Spec.Schedule)
    }
    
    dst.Spec.Schedule = v2.CronSchedule{
        Minute:     schedParts[0],
        Hour:       schedParts[1],
        DayOfMonth: schedParts[2],
        Month:      schedParts[3],
        DayOfWeek:  schedParts[4],
    }
    dst.Spec.Image = src.Spec.Image
    dst.Spec.ConcurrencyPolicy = v2.ConcurrencyPolicy(src.Spec.ConcurrencyPolicy)
    dst.Spec.StartingDeadlineSeconds = src.Spec.StartingDeadlineSeconds
    dst.Spec.SuccessfulJobsHistoryLimit = src.Spec.SuccessfulJobsHistoryLimit
    dst.Spec.FailedJobsHistoryLimit = src.Spec.FailedJobsHistoryLimit
    return nil
}
```

With notes, the same conversion is declared in the Katalog:

```yaml
- from: v1
  to: v2
  spec:
    schedule:
      minute:     "{{ cronMinute .spec.schedule }}"
      hour:       "{{ cronHour   .spec.schedule }}"
      dayOfMonth: "{{ cronDom    .spec.schedule }}"
      month:      "{{ cronMonth  .spec.schedule }}"
      dayOfWeek:  "{{ cronDow    .spec.schedule }}"
```

The cron notes handle @-macros, empty input, and malformed expressions
gracefully — returning `"*"` when a field cannot be extracted rather than
panicking or returning an error that breaks a conversion.

**Verified in production:** 11,279 conversions, 0 failures, 0.69ms p95 latency.

---

## The default problem

Before notes, applying a default value to an absent field required either:
- A mutation rule with `default: "2"` — which always writes a string
- A Go hook that reads the field and writes it if missing

With the `default` note:

```yaml
mutation:
  rules:
    - field: spec.replicas
      default: "{{ default .spec.replicas 2 }}"
```

Or in a status field:

```yaml
status:
  fields:
    - path: phase
      value: "{{ default .status.phase \"Pending\" }}"
```

`default` takes two arguments: the value to check, and the fallback. If the
value is empty or nil, the fallback is returned. One note, zero Go.

---

## The conditional problem

Go templates provide `{{ if }}...{{ else }}...{{ end }}` for conditional values.
This is verbose for inline expressions. The `ternary` note makes it a single expression:

```yaml
# Before notes
status:
  fields:
    - path: phase
      value: |
        {{ if .spec.suspend }}Suspended{{ else }}Active{{ end }}

# With ternary note
status:
  fields:
    - path: phase
      value: "{{ ternary .spec.suspend \"Suspended\" \"Active\" }}"
```

Both produce the same result. The note version fits on one line and reads
as a direct statement of intent.

---

## The numeric problem

When a field in the CRD schema is typed `integer`, writing a string value
to it causes the API server to reject the patch with:

```
spec.replicas: Invalid value: "string": spec.replicas in body must be of type integer
```

Go templates always produce strings. The mutation system previously always
wrote strings. The fix requires detecting that the YAML `default: 2` value
is an `int64`, not the string `"2"`, and preserving that type through the
patch pipeline.

Notes do this naturally for numeric operations. The `add`, `max`, `min`,
`clamp` notes return `int64` when the result is a whole number, `float64`
otherwise. The patch pipeline receives a native Go type. JSON marshal
produces `2`, not `"2"`. The API server accepts it.

```yaml
# Cap replicas at 10, floor at 2
status:
  fields:
    - path: observedReplicas
      value: "{{ clamp .spec.replicas 2 10 }}"
```

---

## Integration

Notes are integrated into the resolver in one line:

```go
tmpl, err := template.New("f").
    Option("missingkey=zero").
    Funcs(note.Map()).   // ← all notes available everywhere
    Parse(value)
```

`note.Map()` is built once at package init — no allocation overhead per call.
Every position where Orkestra evaluates a template expression gains the full
note library without any additional configuration.

---

## The current note catalogue

**Cron:** `cronMinute`, `cronHour`, `cronDom`, `cronMonth`, `cronDow`,
`cronField`, `cronExpr`, `cronValid`

**String:** `toLower`, `toUpper`, `trimSpace`, `trimPrefix`, `trimSuffix`,
`replace`, `contains`, `hasPrefix`, `hasSuffix`, `split`, `join`,
`camelToKebab`, `truncate`

**Math:** `add`, `sub`, `mul`, `div`, `mod`, `min`, `max`, `clamp`, `abs`

**Type conversion:** `toInt`, `toFloat`, `toBool`, `toString`

**Conditional:** `ternary`, `coalesce`, `default`, `empty`, `notEmpty`

---

## The boundary

Notes handle pure transformation. They receive values and return values.
They cannot call an HTTP endpoint, write to a database, or emit a Kubernetes
event. This boundary is not a limitation — it is the constraint that makes
notes safe to use in conversion webhooks, status patching, and validation,
all of which run inside the reconcile loop where side effects are dangerous.

For operations with side effects: Go hooks remain the right answer.

For everything else: a note.

---

## Writing a new note

A note is a Go function registered in the appropriate domain file and in
`note.Map()`. The contract:

1. Accept plain Go types: `string`, `int64`, `float64`, `bool`, `interface{}`
2. Return `(value, error)` for functions that can fail, or just `value` for infallible ones
3. Handle empty input — return a safe zero value, not a panic
4. Be pure — same input always produces same output

```go
// Example: slugify a string for Kubernetes resource naming
// camelToKebab already exists; this is an extension
func slugify(s string) string {
    s = strings.ToLower(s)
    s = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(s, "-")
    s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
    return strings.Trim(s, "-")
}

// Register in stringNotes():
"slugify": slugify,
```

Add it to the catalogue in `pkg/note/README.md`. The note is then available
in every Katalog expression across the entire ecosystem, including patterns
published to the OrkestraRegistry.

---

## Impact on hooks

Before notes, the decision tree for any data transformation was:
1. Can I express this as a template expression? Often not.
2. Write a Go hook.

After notes, the decision tree is:
1. Can I express this as a note composition? Usually yes.
2. Is there an existing note that does this? Check the catalogue.
3. Does the note exist? Use it.
4. Does it not exist? Write a new note (10-20 lines), register it, done.
5. Does the transformation require I/O? Write a hook.

The hook surface shrinks to its irreducible minimum: external API calls,
database writes, and operations with side effects. Everything else — string
manipulation, numeric operations, type conversion, cron parsing, conditional
values — is a note.

This is the correct boundary. Notes are the vocabulary of declarative operators.
Hooks are the escape hatch for what cannot be expressed declaratively.
