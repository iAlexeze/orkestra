# ork generate docs

!!! important "in development"
    This generator is still evolving.  
    Today it produces static Markdown documentation for your CRDs.  
    In the future, it will power the Control Center with **real‑time internal documentation** derived directly from your Katalog and runtime.

Generate Markdown documentation for all CRDs declared in a Katalog.

```bash
ork generate docs --katalog <file> [flags]
```

The generated docs describe each CRD’s fields, validation rules, defaults, status fields, and relationships.

---

## Flags

| Flag | Description |
|------|-------------|
| `-k, --katalog <file>` | One or more Katalog files (comma‑separated or repeated) |
| `-o, --output <file>` | Write output to file (default: stdout) |
| `-n, --namespace <name>` | Namespace for generated resources (default: `orkestra-system`) |
| `--dry-run` | Print output without writing files |

---

## Usage

Generate documentation:

```bash
ork generate docs --katalog katalog.yaml
```

Write to a file:

```bash
ork generate docs --katalog katalog.yaml -o docs.md
```

Multiple Katalogs:

```bash
ork generate docs --katalog a.yaml --katalog b.yaml
```

---

## Behavior

- Merges one or more Katalog files.
- Extracts all enabled CRDs.
- Generates Markdown documentation for each CRD, including:
  - spec fields  
  - required vs optional fields  
  - defaults from mutation rules  
  - validation rules  
  - status fields  
  - printer columns  
- Writes to:
  - stdout (default)
  - a file when `--output` is provided

---

## Notes

- Output is suitable for internal documentation, GitOps repos, and onboarding materials.
- Future versions will integrate with the Control Center to provide live, runtime‑aware documentation.