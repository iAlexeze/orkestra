# Type Notes

Runtime type inspection and conversion.

## Type inspection

| Note | Signature | Returns |
|------|-----------|---------|
| `typeOf` | `v` | `"string"` `"map"` `"slice"` `"number"` `"bool"` `"null"` |
| `typeMap` | `v` | `bool` |
| `typeList` | `v` | `bool` |
| `typeString` | `v` | `bool` |
| `typeNumber` | `v` | `bool` |
| `typeBool` | `v` | `bool` |
| `typeNull` | `v` | `bool` |
| `len` | `v` | `int` — element count for maps/slices, char count for strings |
| `isEmpty` | `v` | `bool` |
| `isScalar` | `v` | `bool` (string, number, bool, null) |
| `isComposite` | `v` | `bool` (map, slice) |

## Type conversion

| Note | Signature | Returns |
|------|-----------|---------|
| `toInt` | `v` | `int64` — truncates floats, parses strings, 1/0 for bools |
| `toFloat` | `v` | `float64` |
| `toBool` | `v` | `bool` — `"true"/"yes"/"1"` → true, `"false"/"no"/"0"` → false |
| `toString` | `v` | `string` |

## Examples

```yaml
# Expose input format in status
- path: scheduleFormat
  value: "{{ typeOf .spec.schedule }}"

# Branch on type in onReconcile (Approach B — no normalize)
when:
  - field: spec.schedule
    operator: typeOf
    value: map

# Validate a field is a map
- field: spec.config
  operator: custom
  value: "{{ typeMap .spec.config }}"
  message: "spec.config must be a structured object"
  action: deny

# Arithmetic on a field that may arrive as a string
replicas: "{{ toInt .spec.replicas }}"
```

> For field-shape coercion (parsing JSON/YAML strings back to maps or lists), see [type coercion notes](coercion.md).
