# String Notes

Standard string transformations. All notes are infallible — they return a safe zero value for empty input and never error.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `toLower` | `string` | `string` |
| `toUpper` | `string` | `string` |
| `trimSpace` | `string` | `string` |
| `trim` | `s, cutset string` | `string` |
| `trimPrefix` | `s, prefix string` | `string` |
| `trimSuffix` | `s, suffix string` | `string` |
| `hasPrefix` | `s, prefix string` | `bool` |
| `hasSuffix` | `s, suffix string` | `bool` |
| `contains` | `s, substr string` | `bool` |
| `replace` | `s, old, new string` | `string` (all occurrences) |
| `split` | `s, sep string` | `[]string` — empty slice for empty input |
| `join` | `[]string, sep string` | `string` |
| `repeat` | `s string, n int` | `string` |
| `camelToKebab` | `string` | `string` |
| `truncate` | `s string, n int` | `string` — appends `...` if truncated |

## Examples

```yaml
# Derive names
name: "{{ trimPrefix .spec.image \"myorg/\" }}"    # "nginx" from "myorg/nginx"
name: "{{ camelToKebab .spec.type }}"              # "web-server" from "WebServer"

# Safe display truncation (appends "...")
label: "{{ truncate .metadata.name 63 }}"

# Extract first element of a comma-separated list
image: "{{ index (split .spec.images \",\") 0 }}"

# Conditional suffix
host: "{{ toLower .spec.name }}.{{ .spec.domain }}"
```

> For Kubernetes resource names (DNS labels), use `truncateName` from [data notes](data.md) — it hard-cuts without appending `...`, which is not a valid character in resource names.
