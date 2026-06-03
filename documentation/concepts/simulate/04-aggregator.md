# Aggregator mode

`ork simulate` can run over an entire directory tree in one command, matching the same pattern as `ork e2e ./...`.

---

## Discovery: `ork simulate ./...`

Discovers all `e2e.yaml` files recursively under the current directory and simulates each:

```bash
cd my-operator/beginner
ork simulate ./...
```

```text
Simulating 4 e2e file(s) under .

  01-hello-website/e2e.yaml .............. ✓ steady at cycle 2 (180ms)
  02-website-with-service/e2e.yaml ....... ✓ steady at cycle 2 (195ms)
  03-secret-copy/e2e.yaml ................ ✓ steady at cycle 3 (210ms)
  03b-bonus-configmap-copy/e2e.yaml ...... ✓ steady at cycle 2 (175ms)

  4 file(s) — 4 simulated, 0 skipped
  Slowest: 03-secret-copy (cycle 3, 210ms)
```

---

## Aggregator e2e.yaml

An aggregator e2e.yaml has `imports` but no `spec.cr`. When simulate receives one as input, it expands the imports and simulates each:

```bash
ork simulate -f e2e.yaml          # e2e.yaml is an aggregator
```

This is the same as `./...` but restricted to the imports declared in that file.

---

## Skip rules

| Condition | Behaviour |
|-----------|-----------|
| `spec.customOperator: true` | Printed as `○ skipped (customOperator)` |
| Pure aggregator (no `spec.cr`) | Skipped silently — nothing to simulate directly |

Everything else simulates — including files with `external:` or `cross:` blocks. Those blocks produce notes in the per-file output but do not cause a skip.

---

## Inactive block tags

Files where a block could not execute show a bracketed tag in the aggregate line:

```text
  02-external-gate/e2e.yaml .............. ✓ steady at cycle 1 (120ms)  [external: inactive]
  03-cross-crd/e2e.yaml .................. ✓ steady at cycle 3 (210ms)  [cross: inactive]
```

The tag means the resource templates and status fields ran, but the block that feeds them data (the HTTP call or informer cross-read) did not. The simulation is still valid for verifying the declarative layer.

---

→ Next: [Limitations](05-limitations.md)
