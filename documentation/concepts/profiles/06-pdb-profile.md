# PDB Behavior Profile

A PDB behavior profile is a named preset that expands into a concrete `minAvailable` or `maxUnavailable` value on a PodDisruptionBudget at Katalog load time.

You write a name. Orkestra writes the disruption limit.

---

## Profiles

| Profile | Field | Value | Use for |
|---------|-------|-------|---------|
| `zero-downtime` | minAvailable | 100% | Databases, stateful services — no pod may be voluntarily disrupted |
| `rolling` | maxUnavailable | 1 | Standard production services — exactly one pod at a time |
| `relaxed` | maxUnavailable | 25% | Stateless services where brief capacity reduction is acceptable |

`zero-downtime` uses `minAvailable` semantics (must stay up). `rolling` and `relaxed` use `maxUnavailable` semantics (may go down). The two fields are mutually exclusive — a PDB uses one or the other, never both.

---

## Usage

Set `behavior.profile` on any PDB resource:

```yaml
onCreate:
  pdb:
    - name: "{{ .metadata.name }}-pdb"
      selector:
        app: "{{ .metadata.name }}"
      behavior:
        profile: rolling
```

---

## Rules

**Profile or explicit — not both.**

`behavior.profile` cannot coexist with `behavior.minAvailable` or `behavior.maxUnavailable`. Orkestra rejects the Katalog at load time if both are present.

```yaml
# Valid
behavior:
  profile: rolling

# Valid
behavior:
  maxUnavailable: "1"

# Invalid — rejected at load time
behavior:
  profile: rolling
  maxUnavailable: "2"
```

**Unknown profiles fail fast.** A typo (`roling`, `Zero-Downtime`) without correct casing is caught at load time. Profile names are case-insensitive — `ROLLING` and `rolling` are both valid.

---

## Choosing a profile

| Situation | Profile |
|-----------|---------|
| Database, message broker, or any stateful service | `zero-downtime` |
| Standard web service or API | `rolling` |
| Stateless worker, batch runner | `relaxed` |
