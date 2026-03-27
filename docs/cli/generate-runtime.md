# ork generate runtime

Generate runtime wiring for typed CRDs, Go hooks, and custom constructors.

```bash
ork generate runtime --katalog <path>
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

- [Typed CRDs](../concepts/typed-crds.md)
- [Go Hooks](../concepts/hooks.md)
- [ork run](./run.md)
