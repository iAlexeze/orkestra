# Notes

Notes are pure transformation functions available in every Orkestra template
expression. They are the vocabulary of declarative operator behavior.

---

## The idea

A Katalog template expression like `{{ .spec.image }}` reads a field.
A note like `{{ cronMinute .spec.schedule }}` transforms one. The distinction
is simple but the consequence is significant: with notes, the gap between
"what I can express declaratively" and "what requires Go hooks" becomes very
small.

Notes are named after musical notes — the atomic units from which music is
composed. In Orkestra, notes are the atomic units from which operator behavior
is composed. Small. Precise. Individually simple. Combinable into complex logic.

---

## Where notes work

Every position where Orkestra evaluates `{{ }}` expressions:

```yaml
# Version conversion paths
conversion:
  paths:
    - from: v1
      to: v2
      spec:
        schedule:
          minute: "{{ cronMinute .spec.schedule }}"

# Status field values
status:
  fields:
    - path: environment
      value: "{{ toLower .spec.environment }}"
    - path: phase
      value: "{{ ternary .spec.suspend \"Suspended\" \"Active\" }}"

# Mutation defaults
mutation:
  rules:
    - field: spec.replicas
      default: "{{ default .spec.replicas 2 }}"

# onCreate templates
onCreate:
  cronJobs:
    - schedule: "{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}"
```

---

## The catalogue

### Cron

Parse and reconstruct standard cron expressions. Handles five-field expressions
and @-macros (`@hourly`, `@daily`, `@weekly`, `@monthly`, `@yearly`).

```
cronMinute  expr          → minute field (0-59, */n, ranges)
cronHour    expr          → hour field (0-23)
cronDom     expr          → day-of-month field (1-31)
cronMonth   expr          → month field (1-12 or JAN-DEC)
cronDow     expr          → day-of-week field (0-6 or SUN-SAT)
cronField   expr pos      → field at position 0-4
cronExpr    min hr d m w  → reconstruct "min hr d m w" string
cronValid   expr          → true if structurally valid
```

**Example — v1 (string) to v2 (structured) conversion:**

```yaml
- from: v1
  to: v2
  spec:
    schedule:
      minute: "{{ cronMinute .spec.schedule }}"
      hour:   "{{ cronHour   .spec.schedule }}"
      dayOfMonth: "{{ cronDom .spec.schedule }}"
      month:  "{{ cronMonth  .spec.schedule }}"
      dayOfWeek:  "{{ cronDow .spec.schedule }}"

- from: v2
  to: v1
  spec:
    schedule: "{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}"
```

### String

```
toLower      s             → lowercase
toUpper      s             → uppercase
trimSpace    s             → strip leading/trailing whitespace
trimPrefix   s prefix      → remove prefix if present
trimSuffix   s suffix      → remove suffix if present
replace      s old new     → replace all occurrences
contains     s substr      → true if contains
hasPrefix    s prefix      → true if starts with
hasSuffix    s suffix      → true if ends with
split        s sep         → []string (use with index)
join         slice sep     → joined string
camelToKebab s             → CamelCase → kebab-case
truncate     s n           → truncate to n chars, append "..."
```

**Examples:**

```yaml
# Derive resource name from CR name
name: "{{ trimPrefix .spec.image \"myorg/\" }}"   # "nginx" from "myorg/nginx"
name: "{{ camelToKebab .spec.type }}"             # "web-server" from "WebServer"
name: "{{ truncate .metadata.name 63 }}"          # respect label length limit

# Access first element of a list field
image: "{{ index (split .spec.images \",\") 0 }}"
```

### Math

All math notes accept string, int64, or float64. Return int64 when the result
is a whole number, float64 otherwise.

```
add   a b       → a + b
sub   a b       → a - b
mul   a b       → a × b
div   a b       → a ÷ b
mod   a b       → a mod b (integer)
min   a b       → smaller of a and b
max   a b       → larger of a and b
clamp val lo hi → val constrained to [lo, hi]
abs   a         → absolute value
```

**Examples:**

```yaml
# Port arithmetic
port: "{{ add .spec.basePort 1000 }}"

# Enforce replica bounds
replicas: "{{ clamp .spec.replicas 2 20 }}"

# Default floor
replicas: "{{ max .spec.replicas 1 }}"
```

### Type conversion

```
toInt    v  → int64 (truncates floats, parses strings, 1/0 for bool)
toFloat  v  → float64
toBool   v  → bool ("true"/"yes"/"1" → true, "false"/"no"/"0" → false)
toString v  → string (fmt.Sprintf("%v", v))
```

### Conditional

```
ternary  cond trueVal falseVal  → trueVal if cond is truthy, else falseVal
coalesce vals...               → first non-empty value
default  val def               → val if non-empty, else def
empty    v                     → true if nil, "", 0, false, empty slice/map
notEmpty v                     → inverse of empty
```

**Examples:**

```yaml
# Phase based on suspension
value: "{{ ternary .spec.suspend \"Suspended\" \"Active\" }}"

# First non-empty image
image: "{{ coalesce .spec.image .spec.defaultImage \"nginx:latest\" }}"

# Default replicas
replicas: "{{ default .spec.replicas 2 }}"

# Conditional resource via status field
- path: message
  value: "No issues"
  when:
    - field: spec.replicas
      operator: notExists
```

---

## Notes vs hooks

| Need | Use |
|---|---|
| Parse a cron string | `cronMinute`, `cronHour`, ... |
| Apply a default value | `default` |
| Choose between values | `ternary`, `coalesce` |
| String manipulation | `toLower`, `trimPrefix`, `replace`, ... |
| Numeric operations | `add`, `max`, `clamp`, ... |
| Type conversion | `toInt`, `toBool`, ... |
| Call an external HTTP API | Hook |
| Write to a database | Hook |
| Send a notification | Hook |
| Complex auth flows | Hook |
| Multi-resource orchestration | Hook (if it involves external I/O) |

If the transformation is pure — same input always produces same output, no
side effects — it is a note. If it requires I/O, it is a hook.

---

## Writing a new note

1. Identify the domain: `cron`, `strings`, `math`, `types`, `conditional`
2. Open `pkg/note/<domain>.go`
3. Write the function — pure, safe, handles empty input
4. Register it in the domain's `xxxNotes()` function
5. Add it to this page and `pkg/note/README.md`

**Contract:**

```go
// Pure — same input, same output
// Safe — never panic, handle nil/empty
// Typed — return native type, not always string

// Return (value, error) for functions that can fail:
func cronMinute(expr string) (string, error) { ... }

// Return just value for infallible functions:
func toLower(s string) string { return strings.ToLower(s) }
```

---

## The `in` operator

One note worth explaining on its own: `in` is used in `when:` conditions
and status field conditions to check membership in a comma-separated list.

```yaml
when:
  - field: status.phase
    operator: in
    value: "Pending,"    # matches "Pending" or empty string
```

The empty string in the list matches a field that has not yet been written.
This is the first-reconcile detection pattern: `"Pending,"` means "when
phase is Pending OR when phase hasn't been written yet."

This is not a note — it is a condition operator. But it works with the
same template evaluation infrastructure and is worth knowing alongside notes.

---

## Integration point

In `pkg/orkestra-registry/template/resolver.go`, the `Resolve` method:

```go
func (r *Resolver) Resolve(value string) (string, error) {
    if !strings.Contains(value, "{{") {
        return value, nil // fast path — no template
    }

    tmpl, err := template.New("f").
        Option("missingkey=zero").
        Funcs(note.Map()).   // ← all notes registered here
        Parse(value)
    // ...
}
```

`note.Map()` is a package-level variable built once at init time. The FuncMap
is safe for concurrent use by multiple goroutines — the resolver is called
from worker goroutines in the reconcile pool.
