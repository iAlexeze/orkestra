# Math Notes

Arithmetic on spec fields. All notes accept `string`, `int64`, or `float64`. Return `int64` when the result is a whole number, `float64` otherwise.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `add` | `a, b` | number |
| `sub` | `a, b` | number |
| `mul` | `a, b` | number |
| `div` | `a, b` | number |
| `mod` | `a, b` | `int64` |
| `min` | `a, b` | number |
| `max` | `a, b` | number |
| `clamp` | `val, lo, hi` | number |
| `abs` | `a` | number |

## Examples

```yaml
# Port arithmetic
port: "{{ add .spec.basePort 1000 }}"

# Enforce replica bounds
replicas: "{{ clamp .spec.replicas 2 20 }}"

# Default floor
replicas: "{{ max .spec.replicas 1 }}"

# Tenant resource quota: 1/4 of the declared CPU limit
cpu: "{{ formatQuantity (div (parseQuantity .spec.cpuLimit) 4) }}"
```

Compose with [resource quantity notes](quantities.md) for Kubernetes CPU/memory arithmetic.
