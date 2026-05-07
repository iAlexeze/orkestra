---
title: "Validate"
weight: 90
---

# ork validate

Validate a Katalog or Komposer. Resolves sources, merges, and runs the full validation pipeline.

```bash
ork validate --file <path>
```

## Flags

| Flag | Description |
|------|-------------|
| `--file` | Path or URL to a Katalog or Komposer (repeatable) |

## Examples

```bash
ork validate --file ./katalog.yaml
ork validate --file ./infra.yaml --file ./apps.yaml
ork validate --file https://raw.github.com/.../katalog.yaml
```

Validation errors are specific and actionable.

---

## Related Documentation

- [ork template](./template.md)
- [Komposer Schema](../komposer-schema.md)
- [Katalog Schema](../katalog-schema.md)
