# ork generate crd

Generate a Kubernetes CustomResourceDefinition (CRD) from a Katalog.

```bash
ork generate crd --file <file> [flags]
```

The Katalog is the single source of truth.  
CRDs are derived from the CRD entries declared inside the Katalog.

---

## Flags

| Flag | Description |
|------|-------------|
| `-k, --file <file>` | Path(s) to Katalog YAML file(s) (required) |
| `-o, --output <path>` | Output file or directory (default: stdout) |
| `--crd <name>` | Generate CRD for a specific CRD name |
| `--all` | Generate CRDs for all CRDs in the Katalog |

---

## Usage

Generate a CRD:

```bash
ork generate crd --file katalog.yaml -o crd.yaml
```

Generate CRD for a specific CRD in a multi‑CRD Katalog:

```bash
ork generate crd --file katalog.yaml --crd pipeline -o pipeline-crd.yaml
```

Generate all CRDs into a directory:

```bash
ork generate crd --file katalog.yaml --all -o ./crds/
```

---

## Behavior

- Merges one or more Katalog files.
- Extracts CRD entries from the merged Katalog.
- Selects:
  - the first CRD (default)
  - a specific CRD (`--crd`)
  - all CRDs (`--all`)
- Generates a CRD with:
  - OpenAPI schema derived from validation rules and template expressions  
  - required fields from deny/exists rules  
  - optional fields inferred from mutation defaults  
  - status subresource schema from `status.fields`  
  - AdditionalPrinterColumns (status fields, phase first)  
  - conversion webhook config when conversion paths are declared  
- Writes to:
  - stdout (default)
  - a file (`-o file.yaml`)
  - a directory (`-o dir/` when `--all` is used)

---

## Notes

- Multiple `--file` values may be provided, including comma‑separated lists.
- Output is prefixed with `---` when writing multiple CRDs.
- The generated CRD is ready for `kubectl apply` or inclusion in operator bundles.
