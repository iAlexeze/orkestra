---
title: "ork init"
weight: 50
description: "Scaffold a new Orkestra operator project."
---

Scaffold a new Orkestra operator project.

```bash
ork init <project-name> [flags]
```

## Flags

| Flag | Description |
|------|-------------|
| `--typed` | Scaffold a Go project for typed CRDs or custom reconcilers |
| `--module` | Go module path (used only with `--typed`) |

---

## Dynamic Project (default)

Creates a Katalog‑only operator — no Go code required.

Directory structure:

```
my-operator/
  examples/
  katalogs/
  .env.example
  README.md
```

Start immediately:

```bash
kubectl apply -f examples/website/website-crd.yaml
ork run --file examples/website/website-katalog.yaml
```

---

## Typed Project

For operators using typed CRDs, Go hooks, or custom reconcilers.

Adds:

```
cmd/orkestra/main.go
pkg/runtime/
pkg/hooks/
go.mod
```

Run:

```bash
go run ./cmd/orkestra/ run --file examples/website/website-katalog.yaml
```

---

## Related Documentation

- [Katalog Schema](../katalog-schema.md)
- [Komposer Schema](../komposer-schema.md)
- [ork validate](./validate.md)
