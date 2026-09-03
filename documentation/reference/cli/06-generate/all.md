# ork generate all

Run all supported generators in a single command.

```bash
ork generate all --file <file> [flags]
```

This command executes, in order:

1. **registry** – generate the runtime registry  
2. **dashboards** – generate Grafana dashboards

It is the fastest way to regenerate all derived artifacts after modifying your Katalog.

---

## Flags

| Flag | Description |
|------|-------------|
| `-k, --file <file>` | One or more Katalog files (comma‑separated or repeated) |
| `--dry-run` | Print output without writing files |
| `-o, --output <file>` | Write output to file (not used by all generators) |
| `-n, --namespace <name>` | Namespace for generated resources (default: `orkestra-system`) |

---

## Usage

Run all generators:

```bash
ork generate all --file katalog.yaml
```

Multiple Katalogs:

```bash
ork generate all --file a.yaml --file b.yaml
```

Dry‑run:

```bash
ork generate all --file katalog.yaml --dry-run
```

---

## Behavior

- Merges one or more Katalog files.
- Validates the merged Katalog.
- Sequentially runs `generate registry` then `generate dashboards`.
- Stops immediately if any generator fails.
- Produces the same output as running each command individually.

---

## Notes

- This is ideal for CI pipelines and release automation.
- Output locations:
  - registry → `pkg/typeregistry/zz_generated_typeregistry.go`
  - dashboards → `_generated/dashboards/`
