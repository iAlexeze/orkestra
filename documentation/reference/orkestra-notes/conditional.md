# Conditional Notes

Value selection and truthiness helpers. Replace `{{ if }}...{{ else }}...{{ end }}` blocks with inline expressions.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `ternary` | `cond, trueVal, falseVal interface{}` | `trueVal` if cond is truthy, else `falseVal` |
| `boolTernary` | `cond bool, trueVal, falseVal interface{}` | `trueVal` if cond is true |
| `boolDefault` | `v interface{}, def bool` | `bool` — use when field may be absent |
| `coalesce` | `vals ...interface{}` | first non-empty value |
| `default` | `val, def interface{}` | `val` if non-empty, else `def` |
| `empty` | `v interface{}` | `bool` — true for nil, `""`, 0, false, empty slice/map |
| `notEmpty` | `v interface{}` | `bool` — inverse of `empty` |

## `ternary` vs `boolTernary`

`ternary` uses Orkestra's truthiness rules — a non-empty string, non-zero number, or non-nil map is truthy. Use it for general conditions.

`boolTernary` requires a strict Go `bool`. Use it when the field is a typed boolean (e.g. `spec.suspend`, `spec.enabled`) to avoid truthiness surprises with string representations of `"false"`.

```yaml
# General condition
value: "{{ ternary .spec.debug \"debug\" \"info\" }}"

# Typed bool field — spec.suspend is a bool, not a string
value: "{{ boolTernary .spec.suspend \"Suspended\" \"Active\" }}"

# Boolean field that may be absent
value: "{{ boolTernary (boolDefault .spec.suspend false) \"Suspended\" \"Active\" }}"
```

## Examples

```yaml
# First non-empty image source
image: "{{ coalesce .spec.image .spec.defaultImage \"nginx:latest\" }}"

# Default replicas when not specified
replicas: "{{ default .spec.replicas 2 }}"

# Conditional resource creation
when:
  - field: spec.monitoring
    operator: notExists
```
