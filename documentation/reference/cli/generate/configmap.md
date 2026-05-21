# ork generate configmap

Generate a Kubernetes ConfigMap that embeds a Katalog or Komposer file.

```bash
ork generate configmap --file <file> [flags]
```

This command is used for **production deployments**, where the Orkestra runtime running inside the cluster must receive its Katalog or Komposer via a ConfigMap.

The generated ConfigMap places the file contents under `data:<filename>` and is ready to apply with `kubectl`.

---

## Flags

| Flag | Description |
|------|-------------|
| `-k, --file <file>` | Path to a Katalog or Komposer file (required) |
| `-o, --output <file>` | Write output to file (default: stdout) |
| `-n, --namespace <name>` | Namespace for the ConfigMap (default: `orkestra-system`) |
| `--dry-run` | Print output without writing files |

---

## Usage

Generate a ConfigMap from a Katalog:

```bash
ork generate configmap --file katalog.yaml
```

Write to a file:

```bash
ork generate configmap --file katalog.yaml -o katalog-configmap.yaml
```

Specify a namespace:

```bash
ork generate configmap --file katalog.yaml -n production
```

Komposer input is also supported:

```bash
ork generate configmap --file komposer.yaml
```

---

## Behavior

- Reads the provided file (Katalog or Komposer).
- Embeds the file contents under `data:<filename>`.
- Produces a valid Kubernetes ConfigMap manifest.
- Writes to:
  - stdout (default)
  - a file when `--output` is provided.

---

## Notes

- This command is required when deploying Orkestra in‑cluster without external storage.
- The generated ConfigMap is consumed by the Orkestra runtime during startup.