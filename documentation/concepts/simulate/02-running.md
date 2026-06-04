# Running simulate

`ork simulate` has three invocation forms: direct flags, an e2e.yaml file, and discovery mode.

---

## Direct flags

Point at a katalog and a CR explicitly:

```bash
# katalog.yaml and cr.yaml in current directory (default filenames)
ork simulate

# Non-standard paths
ork simulate -f my-operator.yaml --cr my-cr.yaml

# Simulate one CRD from a multi-CRD Katalog
ork simulate --crd database

# Run exactly 5 cycles instead of the default 10
ork simulate --cycles 5
```

---

## From an e2e.yaml

If you already have an `e2e.yaml`, point simulate at it directly — it reads `spec.katalog` and `spec.cr` automatically:

```bash
ork simulate -f e2e.yaml
```

`spec.customOperator: true` files are skipped with a note. Aggregator files (imports but no `spec.cr`) expand their imports automatically.

**Multi-document CR files** are supported — separate multiple CRs with `---` and simulate matches each CRD to the CR whose `kind` matches. CRDs with no matching CR in the file are skipped with a note:

```text
  note: no CR found for DatabaseBackedApp — skipped

Simulating managed-database/my-app-db
  ✓ Steady state at cycle 3 in 199ms
```

When `--skip-external` is passed, `external:` blocks produce a note instead of hitting the real network:

```text
Simulating gated-app/my-gated-app
  note: external: calls stubbed — result fields will be empty

  Cycle 1:
    ~ status/my-gated-app
  ✓ Steady state at cycle 2 in 188ms
```

---

## Discovery mode

Run simulate over every e2e.yaml under the current directory:

```bash
ork simulate ./...

# Skip specific files or directories
ork simulate ./... --skip cr-e2e.yaml
ork simulate ./... --skip vendor,testdata
```

Output:

```text
Simulating 6 e2e file(s) under .

  01-hello-website/e2e.yaml .............. ✓ steady at cycle 2 (180ms)
  02-external-gate/e2e.yaml .............. ✓ steady at cycle 1 (120ms)  [external: inactive]
  03-cross-crd/e2e.yaml .................. ✓ steady at cycle 3 (210ms)
  04-secret-copy/e2e.yaml ................ ✓ steady at cycle 2 (19ms)  [cycle errors]
  05-custom-operator/e2e.yaml ............ ○ skipped (customOperator)
  06-custom-operator-2/e2e.yaml .......... ○ skipped (customOperator)

  6 file(s) — 4 simulated, 2 skipped
  Slowest: 03-cross-crd (cycle 3, 210ms)

  Files with cycle errors — run directly for full output:
    ork simulate -f 04-secret-copy/e2e.yaml
```

`customOperator: true` files show as skipped. Pure aggregators (no `spec.cr`) are skipped silently — they contain no simulation target themselves. Files tagged `[cycle errors]` reached steady state but had reconcile errors on every cycle — run them directly to see the full output.

---

## Flag reference

| Flag | Default | Description |
|------|---------|-------------|
| `-f` | `katalog.yaml` | Path to katalog.yaml or e2e.yaml |
| `--cr` | `cr.yaml` | Path to the CR YAML to simulate |
| `--crd` | all | CRD name to simulate (default: all CRDs in the Katalog) |
| `--cycles` | `10` | Maximum number of reconcile cycles |
| `--skip` | — | Comma-separated path patterns to exclude during `./...` discovery |
| `--skip-external` | `false` | Stub `external:` HTTP calls with empty 200 responses instead of hitting the real network |

---

→ Next: [Hooks and constructors](03-hooks-and-constructors.md)
