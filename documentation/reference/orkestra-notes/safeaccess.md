# Safe Access Notes

Type-safe field access with defaults. Use when navigating paths where intermediate keys may be absent or the value may arrive in an unexpected type.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `getOr` | `val, def interface{}` | `val` if non-empty, else `def` |
| `getStringOr` | `val interface{}, def string` | `string` or `def` |
| `getIntOr` | `val interface{}, def int` | `int` or `def` |
| `getBoolOr` | `val interface{}, def bool` | `bool` or `def` |

## Examples

```yaml
# Replicas with default — handles absent key and non-int types
replicas: "{{ getIntOr (get . \"spec\" \"replicas\") 1 }}"

# Image with fallback
image: "{{ getStringOr (get . \"spec\" \"image\") \"nginx:latest\" }}"

# Feature flag with safe default
enabled: "{{ getBoolOr (get . \"spec\" \"monitoring\") false }}"

# Any value with fallback
value: "{{ getOr .spec.config \"{}\" }}"
```

## Difference from `default`

`default val def` returns `def` when `val` is empty (nil, `""`, 0, false). It does not do type coercion.

`getStringOr`, `getIntOr`, `getBoolOr` additionally coerce the value to the target type — useful when a field may arrive as a string `"3"` but you need an `int`, or when navigating deep paths with `get` that may return `nil`.
