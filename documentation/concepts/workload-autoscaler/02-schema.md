# Schema Reference

---

## Scaling modes

Two modes, mutually exclusive per direction:

**`target`** — jump scaling. When conditions are true, set replicas to exactly this value. Appropriate for time-based scaling where the desired state is known. Converges in one reconcile cycle.

**`increment` / `decrement`** — step scaling. Add or remove N replicas per evaluation. Appropriate for metric-driven scaling where gradual ramp-up avoids overshooting. Each reconcile that finds conditions true adds one step.

---

## Field reference

| Field | Required | Description |
|-------|----------|-------------|
| `max` | yes | Ceiling — autoscaler never exceeds this value |
| `min` | no | Floor — defaults to the resolved `replicas` value |
| `cooldown` | no | Wait between scale events (e.g. `3m`, `30s`). Default `1m` |
| `scaleUp.conditions` | no | `when:` / `anyOf:` conditions that trigger scale-up |
| `scaleUp.target` | no | Jump to this exact replica count |
| `scaleUp.increment` | no | Add this many replicas per evaluation tick |
| `scaleDown.conditions` | no | Conditions that trigger scale-down |
| `scaleDown.target` | no | Jump to this exact replica count |
| `scaleDown.decrement` | no | Remove this many replicas per evaluation tick |

---

## Validation

`ork validate` enforces at load time:

- `max` is required — a workload autoscaler with no ceiling fails validation
- `min < max` — if both are declared and `min >= max`, validation fails
- `target` and `increment`/`decrement` are mutually exclusive per direction — if both are declared in the same `scaleUp` or `scaleDown` block, validation fails
