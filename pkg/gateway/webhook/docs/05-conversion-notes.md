# 05 — Conversion Notes and the CronMap Sentinel

## The problem

Go templates produce strings. `text/template` writes every expression into a string buffer — there is no way to make `{{ someNote .spec.field }}` return a `map[string]interface{}` directly.

Conversion path specs, however, can legitimately need to produce structured objects. The v1 → v2 CronJob conversion is the canonical example: `spec.schedule` must become a nested map of named fields, not a plain string.

```yaml
# v1 (string)           →    v2 (map)
spec:                         spec:
  schedule: "*/5 0 * * 1"       schedule:
                                   minute: "*/5"
                                   hour: "0"
                                   dayOfMonth: "*"
                                   month: "*"
                                   dayOfWeek: "1"
```

## The sentinel pattern

`cronToMap` bridges this gap using a sentinel-prefixed JSON string:

1. **`cronToMapTemplate`** (registered as `cronToMap` in the FuncMap) serialises the result as:
   ```
   \x00CMAP\x00{"minute":"*/5","hour":"0","dayOfMonth":"*","month":"*","dayOfWeek":"1"}
   ```
   The prefix uses null bytes — they cannot appear in any valid YAML field value.

2. **`resolveValue`** in `conversion_logic.go` checks every resolved string for the sentinel:
   ```go
   if strings.HasPrefix(resolved, note.CronMapSentinel) {
       payload := resolved[len(note.CronMapSentinel):]
       var m map[string]interface{}
       json.Unmarshal([]byte(payload), &m)
       return m, nil
   }
   ```
   When detected, it unmarshals the JSON and returns `map[string]interface{}` instead of a string. The converted spec field lands as a nested object.

The constant is exported from `pkg/note` so both sides share it without creating a circular import:

```go
// pkg/note/cron.go
const CronMapSentinel = "\x00CMAP\x00"
```

## When to use this pattern

Only when a **conversion path spec field** must produce a nested object rather than a scalar string. Today that is only `cronToMap`.

Status fields, mutation rules, `onCreate`, and `onReconcile` fields all write strings into JSON patches — none of those paths need the sentinel.

If a future note needs to produce an array or a deeply nested map in a conversion spec, follow the same pattern:

1. Serialise the result as `<sentinel> + JSON` in the template function.
2. Define a unique sentinel constant in `pkg/note` (use null bytes to guarantee no collision with YAML values).
3. Detect and decode it in `resolveValue` in `conversion_logic.go`.

## Why not extend the conversion spec DSL instead?

An alternative would be a dedicated syntax for declaring that a conversion field is a structured object rather than a string — for example, writing the value as a YAML map instead of a template expression. That would require a schema change to the Katalog format and a separate resolution path in the conversion engine.

The sentinel approach keeps the conversion spec uniform: every field is a string template expression. The complexity is contained to `resolveValue` and the note implementation, invisible to the Katalog author.
