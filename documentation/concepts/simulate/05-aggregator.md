# Suites and imports

A simulate suite is a Simulate file whose only job is to run other Simulate files. It has no spec of its own — it composes.

---

## Why suites

Each Katalog in a Komposer deserves its own focused simulate: small, fast, easy to debug when it fails. But verifying a whole pattern — or enforcing simulate coverage across an entire pack — should be one command.

The `imports` field bridges the two levels. One suite file at the root imports all the sub-simulations. `ork simulate -f simulate.yaml` runs the suite. Sub-simulations still run individually with `ork simulate -f sub/simulate.yaml`.

---

## Writing a suite file

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Simulate
metadata:
  name: multi-tenancy-sim
  description: >
    Runs simulate for all three multi-tenancy sub-examples.

imports:
  - ./01-basic-namespacing/simulate.yaml
  - ./02-cross-access-control/simulate.yaml
  - ./03-shared-platform/simulate.yaml
```

No `spec:` needed. This is a pure aggregator. `ork validate` confirms each import exists:

```text
✓ multi-tenancy-sim
    imports : 3 file(s)
      ✓ ./01-basic-namespacing/simulate.yaml
      ✓ ./02-cross-access-control/simulate.yaml
      ✓ ./03-shared-platform/simulate.yaml

3 import(s) valid
```

**Try it:**
```bash
ork simulate init      # generate simulate.yaml in each sub-directory first
ork simulate -f simulate.yaml
```

**Scaffold it:**

`ork simulate init --suite` discovers all `simulate.yaml` leaf files under the given directory and writes the aggregator for you:

```bash
ork simulate init --suite               # discover under .
ork simulate init --suite ./examples/   # scoped to a subdirectory
```

See [Scaffolding Tests](03-init.md#scaffolding-a-suite) for full details.

---

## Discovery: `ork simulate ./...`

Discovers all `simulate.yaml` leaf files (files with a `spec:`, not pure aggregators) recursively under the current directory and runs each in assert mode:

```bash
cd my-operator
ork simulate ./...
```

```text
Simulating 4 file(s) under .

  01-hello-website/simulate.yaml ......... ✓ passed (180ms)  [assert]
  02-with-serviceaccount/simulate.yaml ... ✓ passed (195ms)  [assert]
  03-secret-copy/simulate.yaml ........... ✓ passed (210ms)  [assert]
  03b-bonus-configmap-copy/simulate.yaml . ✓ passed (175ms)  [assert]

  4 file(s) — 4 simulated, 0 skipped
  Slowest: 03-secret-copy/simulate.yaml (210ms)
```

---

## Skip rules

| Condition | Behaviour |
|-----------|-----------|
| Pure aggregator (imports, no spec) | Skipped silently — nothing to simulate directly |

Use `--skip` to exclude directories or filename patterns:

```bash
ork simulate ./... --skip vendor,testdata
```
