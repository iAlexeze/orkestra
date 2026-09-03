# Running simulate

`ork simulate` has three invocation forms: direct flags, a simulate.yaml file, and discovery mode.

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

# Start the dev server at localhost:9999 before running
ork simulate --dev-server
```

---

## From a simulate.yaml

Point simulate at a `simulate.yaml` directly:

```bash
ork simulate -f simulate.yaml
```

A `simulate.yaml` carries its own `spec.katalog` and `spec.cr` — you do not need to pass them separately. Aggregator files (`imports:` but no `spec:`) expand their imports automatically. See [Aggregator mode](05-aggregator.md).

**Multi-document CR files** are supported — separate multiple CRs with `---`. Simulate matches each CRD to the CR whose `kind` matches and seeds the remaining CRs as peers in the fake cluster, enabling `cross:` observation between them. CRDs with no matching CR in the file are skipped with a note:

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

Run simulate over every `simulate.yaml` under the current directory:

```bash
ork simulate ./...

# Skip specific files or directories
ork simulate ./... --skip vendor,testdata
```

Output:

```text
Simulating 6 simulate file(s) under .

  01-hello-website/simulate.yaml ......... ✓ steady at cycle 2 (180ms)
  02-external-gate/simulate.yaml ......... ✓ steady at cycle 1 (120ms)  [external: inactive]
  03-cross-crd/simulate.yaml ............. ✓ steady at cycle 3 (210ms)
  04-secret-copy/simulate.yaml ........... ✓ steady at cycle 2 (19ms)  [cycle errors]
  05-hooks/simulate.yaml ................. ✓ steady at cycle 2 (85ms)
  06-constructor/simulate.yaml ........... ✓ steady at cycle 1 (42ms)

  6 file(s) — 6 simulated
  Slowest: 03-cross-crd (cycle 3, 210ms)

  Files with cycle errors — run directly for full output:
    ork simulate -f 04-secret-copy/simulate.yaml
```

Pure aggregators (no `spec:`) are expanded — their imports run in place. Files tagged `[cycle errors]` reached steady state but had reconcile errors on some cycle — run them directly to see the full output.

---

## Flag reference

| Flag | Default | Description |
|------|---------|-------------|
| `-f` | `simulate.yaml` | Path to a `simulate.yaml` or `katalog.yaml` |
| `--cr` | `cr.yaml` | Path to the CR YAML to simulate |
| `--crd` | all | CRD name to simulate (default: all CRDs in the Katalog) |
| `--cycles` | `10` | Maximum number of reconcile cycles |
| `--dev-server` | `false` | Start the Orkestra dev server at `localhost:9999` before running |
| `--skip` | — | Comma-separated path patterns to exclude during `./...` discovery |
| `--skip-external` | `false` | Stub `external:` HTTP calls with empty 200 responses |
