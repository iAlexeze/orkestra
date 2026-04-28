# Data Notes

Encoding, hashing, and naming utilities.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `toBase64` | `string` | `string` |
| `fromBase64` | `string` | `string` — `""` on error |
| `toJSON` | `interface{}` | `string` — `"{}"` on error |
| `sha256sum` | `string` | `string` (8 hex chars) |
| `slugify` | `string` | `string` — lowercase, spaces/symbols → dashes, DNS-safe |
| `truncateName` | `s string, n int` | `string` — hard cut, no suffix (for resource names) |

## Examples

```yaml
# Secret data: encode value to base64 for Secret.data
data:
  password: "{{ toBase64 .spec.password }}"

# Read a base64-encoded value from a child Secret
- path: dbPassword
  value: "{{ fromBase64 .children.secret.data.password }}"

# Content-addressed resource name — changes when config changes
name: "config-{{ sha256sum .spec.config }}"

# DNS-safe name from user input ("My App / Service" → "my-app-service")
name: "{{ slugify .spec.teamName }}-operator"

# Hard-truncate for k8s resource names (no "..." suffix)
name: "{{ truncateName .spec.projectName 50 }}-deployment"

# Embed structured spec in an annotation
annotations:
  orkestra.io/config: "{{ toJSON .spec.config }}"
```

## `truncateName` vs `truncate`

- `truncate` (from [string notes](string.md)) appends `...` — for human-readable display.
- `truncateName` hard-cuts — for Kubernetes resource names where `...` is not a valid character.
