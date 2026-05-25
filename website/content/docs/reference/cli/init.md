---
title: "ork init"
date: 2026-05-25
weight: 85
---

Initialize a new Orkestra operator project.

```bash
ork init [project-name] [flags]
```

With no arguments, `ork init` creates a `katalog.yaml` in the current directory — a ready-to-run hello-website operator scaffold. Pass a name to create a new subdirectory. Use `--pack` to initialize from an example pack instead.

Packs are cached locally for fast repeated use and offline initialization.

---

## Flags

| Flag | Description |
|------|-------------|
| `-p, --pack <name>` | Example pack to initialize (default: `beginner`) |
| `-l, --list` | List available example packs |
| `--clear-cache` | Clear all cached example packs |
| `--refresh-cache` | Force re-download of the selected pack |

Global flags (`--kubeconfig`, `--file`, `--debug`, `--verbose`) are intentionally hidden for this command.

---

## Example Packs

List available packs:

```bash
ork init --list
```

Available packs:

- `beginner` — Simple CRDs, Deployments, Services  
- `intermediate` — Multi-resource patterns, when/anyOf, Komposer basics  
- `advanced` — Hooks, constructors, validation/mutation, registries  
- `use-cases` — Full-stack, cross-CRD flows, external gates, once-secrets  

Use a specific pack:

```bash
ork init --pack advanced
```

Or init into a named subdirectory:

```bash
ork init my-operator --pack advanced
```

---

## Cache

Packs are cached under:

```text
~/.orkestra/packs/
```

Clear cache:

```bash
ork init --clear-cache
```

Force re-download:

```bash
ork init --refresh-cache
```

---

## Project Structure

Running:

```bash
ork init --pack beginner
```

Produces (in current directory):

```text
beginner/
  01-hello-website/
    crd.yaml
    katalog.yaml
    cr.yaml
    README.md
  02-configmap-operator/
    ...
```

---

## Next Steps

Inside the example:

```bash
cd beginner/01-hello-website
ork run
```

Open Control Center:

```bash
ork control
```

---

## Notes

- `--list`, `--clear-cache`, and `--refresh-cache` do **not** require a project name.
- Packs are version‑matched to the installed Orkestra CLI.
- Initialization works offline if the pack is already cached.

---
