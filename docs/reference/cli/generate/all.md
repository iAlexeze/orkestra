# ork generate all

Run all supported generators in a single command.

```bash
ork generate all --katalog <file> [flags]
```

This command executes, in order:

1. **registry** – generate the runtime registry  
2. **docs** – generate Markdown documentation  
3. **dashboards** – generate Grafana dashboards  
4. **rbac** – generate minimal RBAC rules  
5. *(examples/tests removed in the new standard)*

It is the fastest way to regenerate all derived artifacts after modifying your Katalog.

---

## Flags

| Flag | Description |
|------|-------------|
| `-k, --katalog <file>` | One or more Katalog files (comma‑separated or repeated) |
| `--dry-run` | Print output without writing files |
| `-o, --output <file>` | Write output to file (not used by all generators) |
| `-n, --namespace <name>` | Namespace for generated resources (default: `orkestra-system`) |

---

## Usage

Run all generators:

```bash
ork generate all --katalog katalog.yaml
```

Multiple Katalogs:

```bash
ork generate all --katalog a.yaml --katalog b.yaml
```

Dry‑run:

```bash
ork generate all --katalog katalog.yaml --dry-run
```

---

## Behavior

- Merges one or more Katalog files.
- Validates the merged Katalog.
- Sequentially runs:
  - `generate registry`
  - `generate docs`
  - `generate dashboards`
  - `generate rbac`
- Stops immediately if any generator fails.
- Produces the same output as running each command individually.

---

## Notes

- This is ideal for CI pipelines and release automation.
- Output locations depend on each generator:
  - registry → `pkg/runtime/zz_generated_runtime_registry.go`
  - docs → `./dash/`
  - dashboards → `./dash/`
  - rbac → stdout or `--output` if provided