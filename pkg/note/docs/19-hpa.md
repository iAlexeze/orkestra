# 19 — HPA Notes

HPA notes surface replica counts, scaling state, and scale-target details from HorizontalPodAutoscaler objects.

---

## Reference

### `hpaCurrentReplicas`

Returns `status.currentReplicas` as `int64`.

```yaml
- path: currentReplicas
  value: "{{ hpaCurrentReplicas .children.hpa }}"
# → 3
```

---

### `hpaDesiredReplicas`

Returns `status.desiredReplicas` as `int64`.

```yaml
- path: desiredReplicas
  value: "{{ hpaDesiredReplicas .children.hpa }}"
# → 5
```

---

### `hpaMinReplicas`

Returns `spec.minReplicas` as `int64`, defaulting to `1` when not set (matching the Kubernetes default).

```yaml
- path: minReplicas
  value: "{{ hpaMinReplicas .children.hpa }}"
# → 2
```

---

### `hpaMaxReplicas`

Returns `spec.maxReplicas` as `int64`.

```yaml
- path: maxReplicas
  value: "{{ hpaMaxReplicas .children.hpa }}"
# → 10
```

---

### `hpaScaling`

Returns `true` when `currentReplicas != desiredReplicas` — the HPA is actively adjusting.

```yaml
when:
  - field: "{{ hpaScaling .children.hpa }}"
    equals: "false"
```

---

### `hpaAtMax`

Returns `true` when `currentReplicas >= maxReplicas` — the HPA has hit its ceiling.

```yaml
- path: atCapacity
  value: "{{ hpaAtMax .children.hpa }}"
```

---

### `hpaScaleTargetName`

Returns `_scaleTarget.name` — the name of the resource this HPA scales. Requires `enrich: [hpa]`.

```yaml
- path: scaleTargetName
  value: "{{ hpaScaleTargetName .children.hpa }}"
# → "my-app"
```

---

### `hpaScaleTargetKind`

Returns `_scaleTarget.kind` — the kind of the resource this HPA scales. Requires `enrich: [hpa]`.

```yaml
- path: scaleTargetKind
  value: "{{ hpaScaleTargetKind .children.hpa }}"
# → "Deployment"
```

---

### `hpaMetricTypes`

Returns a comma-separated list of metric source types from `_currentMetrics`. Requires `enrich: [hpa]`.

```yaml
- path: metricTypes
  value: "{{ hpaMetricTypes .children.hpa }}"
# → "Resource, External"
```

---

## Quick reference

| Note | Signature | Returns | Enrichment |
|------|-----------|---------|------------|
| `hpaCurrentReplicas` | `(obj any)` | `int64` | none |
| `hpaDesiredReplicas` | `(obj any)` | `int64` | none |
| `hpaMinReplicas` | `(obj any)` | `int64` | none |
| `hpaMaxReplicas` | `(obj any)` | `int64` | none |
| `hpaScaling` | `(obj any)` | `bool` | none |
| `hpaAtMax` | `(obj any)` | `bool` | none |
| `hpaScaleTargetName` | `(obj any)` | `string` | `enrich: [hpa]` |
| `hpaScaleTargetKind` | `(obj any)` | `string` | `enrich: [hpa]` |
| `hpaMetricTypes` | `(obj any)` | `string` | `enrich: [hpa]` |

---

**Next →** [20 — Ingress Notes](20-ingress.md)
