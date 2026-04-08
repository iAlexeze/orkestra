---
title: "Validate"
weight: 90
---

# ork validate

Validate a Katalog or Komposer. Resolves sources, merges, and runs the full validation pipeline.

```bash
ork validate --katalog <path>
```

## Flags

| Flag | Description |
|------|-------------|
| `--katalog` | Path or URL to a Katalog or Komposer (repeatable) |

## Examples

```bash
ork validate --katalog ./katalog.yaml
ork validate --katalog ./infra.yaml --katalog ./apps.yaml
ork validate --katalog https://raw.github.com/.../katalog.yaml
```

Validation errors are specific and actionable.

---

## Related Documentation

- [ork template](./template.md)
- [Komposer Schema](../komposer-schema.md)
- [Katalog Schema](../katalog-schema.md)
