# ork init

Initialize a new Orkestra operator project using a versioned example pack.

```
ork init <project-name> [flags]
```

`ork init` downloads a version‑matched example pack, extracts it into a new project directory, and prepares a ready‑to‑run scaffold.  
Packs are cached locally for fast repeated use and offline initialization.

---

## Flags

| Flag | Description |
|------|-------------|
| `-p, --pack <name>` | Example pack to initialize (default: `beginner`) |
| `-l, --list-packs` | List available example packs |
| `--clear-cache` | Clear all cached example packs |
| `--refresh-cache` | Force re-download of the selected pack |

Global flags (`--kubeconfig`, `--file`, `--debug`, `--verbose`) are intentionally hidden for this command.

---

## Example Packs

List available packs:

```
ork init --list-packs
```

Available packs:

- `beginner` — Simple CRDs, Deployments, Services  
- `intermediate` — Multi-resource patterns, when/anyOf, Komposer basics  
- `advanced` — Hooks, constructors, validation/mutation, registries  
- `use-cases` — Full-stack, cross-CRD flows, external gates, once-secrets  

Use a specific pack:

```
ork init my-operator --pack advanced
```

---

## Cache

Packs are cached under:

```
~/.orkestra/packs/
```

Clear cache:

```
ork init --clear-cache
```

Force re-download:

```
ork init my-operator --refresh-cache
```

---

## Project Structure

Running:

```
ork init my-operator
```

Produces:

```
my-operator/
  examples/
    <pack>/
      01-hello-website/
        crd.yaml
        katalog.yaml
        cr.yaml
  examples_<pack>_<version>.tar.gz
```

---

## Next Steps

Inside the project:

```
cd my-operator
kubectl apply -f examples/<pack>/01-hello-website/crd.yaml
ork run --file examples/<pack>/01-hello-website/katalog.yaml
kubectl apply -f examples/<pack>/01-hello-website/cr.yaml
```

Open Control Center:

```
ork control start
```

---

## Notes

- `--list-packs`, `--clear-cache`, and `--refresh-cache` do **not** require a project name.
- Packs are version‑matched to the installed Orkestra CLI.
- Initialization works offline if the pack is already cached.

---
