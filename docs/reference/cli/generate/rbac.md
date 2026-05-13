# ork generate rbac

Generate a minimal ClusterRole for all CRDs declared in a Katalog.

```bash
ork generate rbac --file <file> [flags]
```

The generated RBAC contains only the permissions required by the CRDs in the merged Katalog, including conditional webhook permissions when validation, mutation, or conversion rules are present.

---

## Flags

| Flag | Description |
|------|-------------|
| `-k, --file <file>` | One or more Katalog files (comma‑separated or repeated) |
| `-o, --output <file>` | Write output to file (default: stdout) |
| `-n, --namespace <name>` | Namespace for the ServiceAccount (default: `orkestra-system`) |
| `--dry-run` | Print output without writing files |

---

## Usage

Generate RBAC from a single Katalog:

```bash
ork generate rbac --file katalog.yaml
```

Multiple Katalogs:

```bash
ork generate rbac --file a.yaml --file b.yaml
```

Comma‑separated:

```bash
ork generate rbac --file a.yaml,b.yaml
```

Write to file:

```bash
ork generate rbac --file katalog.yaml -o rbac.yaml
```

---

## Behavior

- Merges one or more Katalog files.
- Validates the merged Katalog.
- Computes the minimal RBAC rules required for:
  - CRUD operations on declared CRDs  
  - status subresource  
  - finalizers  
  - webhook operations when validation, mutation, or conversion rules exist  
- Generates a ClusterRole manifest.
- Writes to:
  - stdout (default)
  - a file when `--output` is provided

---

## Notes

- The generated RBAC is deterministic and suitable for GitOps.
- This command is used internally by `ork generate bundle`.