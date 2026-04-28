# List / Map Notes

Inspect and transform slices and maps. All notes accept `interface{}` and return safe zero values for nil or wrong-type input.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `listHas` | `list, val interface{}` | `bool` |
| `listGet` | `list interface{}, index int` | `interface{}` — nil when out of bounds |
| `listLen` | `list interface{}` | `int` |
| `mapGet` | `map interface{}, key string` | `interface{}` |
| `mapKeys` | `map interface{}` | `[]string` |
| `mapValues` | `map interface{}` | `[]interface{}` |
| `mapPick` | `map interface{}, keys ...string` | `map[string]interface{}` — only named keys |
| `mapOmit` | `map interface{}, keys ...string` | `map[string]interface{}` — keys removed |

## Examples

```yaml
# Check region membership
- path: hasEastRegion
  value: "{{ listHas .spec.regions \"us-east-1\" }}"

# Use primary region
primaryRegion: "{{ listGet .spec.regions 0 }}"

# Region count
- path: regionCount
  value: "{{ listLen .spec.regions }}"

# Forward only public labels to a child resource
labels: "{{ mapOmit .metadata.labels \"internal-tag\" \"cost-center\" }}"

# Pass a focused subset of spec to a child
config: "{{ mapPick .spec \"image\" \"replicas\" \"port\" }}"

# Read a label value
- path: appLabel
  value: "{{ mapGet .metadata.labels \"app\" }}"
```

> For parsing YAML/JSON strings into lists or maps, see [type coercion notes](coercion.md).
