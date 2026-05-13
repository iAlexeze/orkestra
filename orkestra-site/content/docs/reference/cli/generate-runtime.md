---
title: "Generate Runtime Registry"
weight: 81
---

# ork generate registry

Generate runtime registry for typed CRDs, Go hooks, and custom constructors.

```bash
ork generate registry --file <path>
```

## Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Print output without writing files |

## When Required

| Situation | Needed |
|-----------|--------|
| Dynamic templates only | No |
| Hooks declared | Yes |
| Typed CRDs | Yes |
| Custom constructor | Yes |

---

## Related Documentation

- [Typed CRDs](../../runtime-manual/concepts/typed-crds.md)
- [Go Hooks](../../runtime-manual/concepts/hooks.md)
- [ork run](./run.md)
