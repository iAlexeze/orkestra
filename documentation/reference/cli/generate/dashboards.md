# ork generate dashboards

!!! important "in development"
    This command is under active development.  
    Output format, structure, and generator behavior may change in upcoming releases.

Generate Grafana dashboards for all CRDs declared in a Katalog.

```bash
ork generate dashboards --file <file> [flags]
```

The generated dashboards follow Orkestra’s standard layout conventions and include panels for status, phases, conditions, and resource‑specific metrics.

---

## Flags

| Flag | Description |
|------|-------------|
| `-k, --file <file>` | One or more Katalog files (comma‑separated or repeated) |
| `-o, --output <file>` | Write output to file (default: stdout) |
| `-n, --namespace <name>` | Namespace for generated resources (default: `orkestra-system`) |
| `--dry-run` | Print output without writing files |

---

## Usage

Generate dashboards:

```bash
ork generate dashboards --file katalog.yaml
```

Write to a file:

```bash
ork generate dashboards --file katalog.yaml -o dashboards.json
```

Multiple Katalogs:

```bash
ork generate dashboards --file a.yaml --file b.yaml
```

---

## Behavior

- Merges one or more Katalog files.
- Extracts all enabled CRDs.
- Generates a Grafana dashboard per CRD, including:
  - summary panels  
  - phase/condition visualizations  
  - status field panels  
  - resource‑specific metrics (where applicable)
- Writes to:
  - stdout (default)
  - a file when `--output` is provided

---

## Notes

- Dashboard generation is evolving and may expand with additional panels and metrics.
- Output is suitable for import into Grafana or packaging into Helm charts.