# ork generate registry

Generate the runtime registry file used by typed operators.

```bash
ork generate registry --katalog <file> [flags]
```

This command produces:

- `pkg/runtime/zz_generated_runtime_registry.go`
- containing `RegisterRuntimeObjects()` and `RegisterScheme()`
- for all **enabled CRDs** with `reconciler.default: false`

It is required whenever you use **typed CRDs**, **Go hooks**, or **custom constructors**.

---

## Flags

| Flag | Description |
|------|-------------|
| `-k, --katalog <file>` | One or more Katalog files (comma‑separated or repeated) |
| `--dry-run` | Print generated output without writing files |
| `-o, --output <file>` | (Ignored — registry always writes to the runtime package) |
| `-n, --namespace <name>` | Namespace (not used by registry generation) |

---

## Usage

Generate the registry from a single Katalog:

```bash
ork generate registry --katalog katalog.yaml
```

Multiple Katalogs:

```bash
ork generate registry --katalog a.yaml --katalog b.yaml
```

Comma‑separated:

```bash
ork generate registry --katalog a.yaml,b.yaml
```

Dry‑run:

```bash
ork generate registry --katalog katalog.yaml --dry-run
```

---

## When Required

| Situation | Required |
|----------|----------|
| Dynamic templates only | No |
| Hooks declared | Yes |
| Typed CRDs | Yes |
| Custom constructors | Yes |
| Using Orkestra as a pure dynamic operator | No |

---

## Behavior

- Merges one or more Katalog files.
- Validates the merged Katalog.
- Selects all CRDs with `reconciler.default: false`.
- Generates:
  - type registrations  
  - scheme registrations  
  - runtime object constructors  
- Writes to `pkg/runtime/zz_generated_runtime_registry.go` (idempotent).

---

## Related Documentation

- [Typed CRDs](../../runtime-manual/concepts/typed-crds.md)
- [Go Hooks](../../runtime-manual/concepts/hooks.md)
- [ork run](./run.md)
