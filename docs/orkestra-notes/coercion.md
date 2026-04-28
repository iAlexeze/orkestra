# Type Coercion Notes

Parse YAML or JSON strings back into structured Go values, or convert any value to string. Useful when data is stored as a serialized string (e.g. in annotations or ConfigMap values) and needs to be treated as a structured object.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `asList` | `interface{}` | `[]interface{}` — tries YAML then JSON; empty slice on failure |
| `asMap` | `interface{}` | `map[string]interface{}` — tries YAML then JSON; empty map on failure |
| `asString` | `interface{}` | `string` — JSON-encodes non-string inputs |

## Examples

```yaml
# Annotation stored as a JSON array → use with listHas
regions: "{{ listHas (asList (index .metadata.annotations \"orkestra.io/regions\")) \"us-east-1\" }}"

# ConfigMap value stored as YAML/JSON → treat as a map
config: "{{ asMap (get .children.configmap \"data\" \"settings\") }}"

# Convert any value to a string for annotation storage
annotations:
  orkestra.io/spec: "{{ asString .spec }}"
```

## When to use

Use `asList` / `asMap` when:
- A field comes from a ConfigMap `data` value (always strings)
- A value is stored in an annotation as serialized JSON or YAML
- A field type changed between CRD versions and may arrive in either string or structured form

If the input is already the right type, these notes return it as-is — they are idempotent.
