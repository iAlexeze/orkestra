# 18 — Warning Event Notes

Warning event notes navigate the `_warnings` enrichment embedded by the enrichment layer into Deployment, StatefulSet, Job, Service, and any other resource type. They let templates surface Kubernetes Warning events without hooks or custom Go code. All require `enrich: [events]` (or `enrichAll: true`) on the CRD.

---

## Reference

### `hasWarnings`

Returns `true` when `_warnings` contains at least one event.

```yaml
when:
  - field: "{{ hasWarnings .children.deployment }}"
    equals: "false"
```

---

### `warningCount`

Returns the number of warning events as `int`.

```yaml
- path: warningCount
  value: "{{ warningCount .children.deployment }}"
# → 3
```

---

### `firstWarning`

Returns the message of the first warning event, or `""` when there are no warnings.

```yaml
- path: lastWarning
  value: "{{ firstWarning .children.deployment }}"
# → "Back-off restarting failed container"
```

---

### `warningMessages`

Returns a comma-separated list of all warning messages.

```yaml
- path: warningMessages
  value: "{{ warningMessages .children.deployment }}"
# → "OOMKilled, CrashLoopBackOff"
```

---

### `warningReasons`

Returns a comma-separated list of all warning reasons (de-duplicated).

```yaml
- path: warningReasons
  value: "{{ warningReasons .children.deployment }}"
# → "OOMKilled, BackOff"
```

---

## Quick reference

| Note | Signature | Returns | Enrichment |
|------|-----------|---------|------------|
| `hasWarnings` | `(obj any)` | `bool` | `enrich: [events]` |
| `warningCount` | `(obj any)` | `int` | `enrich: [events]` |
| `firstWarning` | `(obj any)` | `string` | `enrich: [events]` |
| `warningMessages` | `(obj any)` | `string` | `enrich: [events]` |
| `warningReasons` | `(obj any)` | `string` | `enrich: [events]` |

---

**Next →** [19 — HPA Notes](19-hpa.md)
