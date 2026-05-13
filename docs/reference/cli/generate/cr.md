# ork generate cr

Generate an example CustomResource (CR) from a Katalog.

```bash
ork generate cr --file <file> [flags]
```

The generated CR is derived entirely from the CRD entry declared in the Katalog.  
It includes required fields, optional fields with defaults, and typed placeholder values.

---

## Flags

| Flag | Description |
|------|-------------|
| `-k, --file <file>` | Path(s) to Katalog YAML file(s) (required) |
| `-o, --output <file>` | Write output to file (default: stdout) |
| `--crd <name>` | Generate CR for a specific CRD (default: first CRD) |

---

## Usage

Generate an example CR:

```bash
ork generate cr --file katalog.yaml -o cr.yaml
```

Generate CR for a specific CRD:

```bash
ork generate cr --file katalog.yaml --crd pipeline -o pipeline-cr.yaml
```

---

## Behavior

- Merges one or more Katalog files.
- Selects:
  - the first CRD (default)
  - a specific CRD (`--crd`)
- Generates an example CR with:
  - required fields populated with typed placeholders  
  - optional fields showing their default values  
  - comments explaining each field  
  - metadata derived from the CRD  
- Writes:
  - a single CR to stdout (default)
  - or to a file when `--output` is provided

---

## Notes

- Multiple `--file` values may be provided, including comma‑separated lists.
- Output includes a header with:
  - CRD kind  
  - source katalog path  
  - usage hint (`kubectl apply -f`)  
- The generated CR is ready for immediate use with `kubectl apply`.

