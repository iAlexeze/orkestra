---
title: "ork validate"
weight: 35
---

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
