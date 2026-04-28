# Resource Quantity Notes

Kubernetes CPU and memory arithmetic using standard SI (`m`, `K`, `M`, `G`) and binary (`Ki`, `Mi`, `Gi`) suffixes.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `parseQuantity` | `string` | `float64` — error on invalid |
| `formatQuantity` | `float64` | `string` — error on invalid |
| `sumQuantity` | `a, b string` | `string` — error on invalid |

`parseQuantity` returns CPU in cores (not millicores) — `"500m"` → `0.5`, `"2"` → `2.0`.  
`parseQuantity` returns memory in bytes — `"1Gi"` → `1073741824.0`.

`formatQuantity` expresses sub-unit fractions as milli-units: `0.1` → `"100m"`, `0.5` → `"500m"`.

## Examples

```yaml
# Give each tenant 1/4 of the declared CPU limit
resources:
  requests:
    cpu: "{{ formatQuantity (div (parseQuantity .spec.cpuLimit) 4) }}"

# Total memory across two pools
- path: totalMemory
  value: "{{ sumQuantity .spec.poolA.memory .spec.poolB.memory }}"

# Cap memory request at node size × headroom
memory: "{{ formatQuantity (min (parseQuantity .spec.memory) (mul (parseQuantity .spec.nodeMemory) 0.8)) }}"
```

Compose with [math notes](math.md) for arithmetic between parsed quantities.
