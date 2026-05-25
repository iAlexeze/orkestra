---
title: "ork validate"
date: 2026-05-20
weight: 59
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
ork validate
# Orkestra reads katalog.yaml from the current directory.
ork validate --file ./infra.yaml --file ./apps.yaml
ork validate --file https://raw.github.com/.../katalog.yaml
```

Validation errors are specific and actionable.
