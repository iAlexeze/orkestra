# ork template

Render the merged, validated Katalog exactly as Orkestra will see it at runtime.

```bash
ork template --file <path>
```

## Flags

| Flag | Description |
|------|-------------|
| `--json` | Output CRDs as JSON |
| `--yaml` | Output CRDs as YAML |
| `--graph` | ASCII dependency graph |
| `--verbose` | Show all fields |

---

## Examples

```bash
ork template --file ./katalog.yaml
ork template --file ./katalog.yaml --json
ork template --file ./komposer.yaml --graph
```

---

## Related Documentation

- [ork validate](./validate.md)
- [ork run](./run.md)
- [Katalog Schema](../katalog-schema.md)
