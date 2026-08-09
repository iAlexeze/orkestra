# ork validate

Validate an Orkestra pattern. The kind is detected automatically from the `kind` field: Katalog, Komposer, Motif, E2E, or Simulate.

```bash
ork validate --file <path>
```

## Flags

| Flag | Description |
|------|-------------|
| `--file` / `-f` | Path or URL to an Orkestra pattern (repeatable) |
| `--notes` | Print the merged user-defined note registry after validation |
| `--profiles` | Print the merged user-defined profile registry after validation |
| `--full` | Show per-CRD permissions, dependency graph, and system-level RBAC |
| `--play` | For Simulate specs: skip `spec.cr` requirement (`ork serve play` will supply the CR) |

## Examples

```bash
ork validate
# Reads katalog.yaml from the current directory.

ork validate --file ./infra.yaml --file ./apps.yaml
ork validate --file https://raw.github.com/.../katalog.yaml

# Validate a simulate spec that will be used with ork serve play
ork validate -f simulate.yaml --play
```

## Simulate specs and --play

When validating a `kind: Simulate` spec, `ork validate` normally requires `spec.cr` — a path to the CR file that the reconciler will run against. When the spec is intended to be used with `ork serve play --simulate=simulate.yaml`, the CR is built by play at runtime and does not need to be present on disk. Pass `--play` to skip the `spec.cr` requirement:

```bash
ork validate -f simulate.yaml --play
```

All other fields (`spec.katalog`, `spec.expect`, `spec.cycles`) are still validated normally. If `spec.cr` is set, its path is still checked.

Validation errors are specific and actionable.
