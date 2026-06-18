# Orkestra Templates

Ready-to-use starting points for every Orkestra file type. Copy the one you need, replace the `<placeholders>`, and go.

---

## Templates

| File | Use when |
|------|----------|
| [`katalog.yaml`](./katalog.yaml) | Building an operator for one or more CRDs |
| [`komposer.yaml`](./komposer.yaml) | Composing multiple Katalogs into one runtime |
| [`motif.yaml`](./motif.yaml) | Defining a reusable resource pattern |
| [`e2e.yaml`](./e2e.yaml) | Writing a declarative end-to-end test |
| [`simulate.yaml`](./simulate.yaml) | Verifying reconcile logic without a cluster |

---

## Quick start

```bash
# Copy the template you need into your project
cp templates/katalog.yaml my-operator/katalog.yaml

# Validate before running
ork validate -f my-operator/katalog.yaml

# Run the operator
ork run -f my-operator/katalog.yaml
# username:password → orkestra
```

---

## Which one do I need?

**`katalog.yaml`** — start here. One file, one or more CRDs, a full operator.

**`komposer.yaml`** — when you want to combine multiple Katalogs (local files, OCI registry, Helm) into a single runtime and apply per-environment overrides.

**`motif.yaml`** — when you have a resource pattern (e.g. a StatefulSet + headless Service + PVC) that you want to reuse across multiple operators without copy-pasting.

**`e2e.yaml`** — when you want a reproducible test that spins up a kind cluster, applies your operator, asserts expectations, and tears down. Runs the same way locally and in CI.

**`simulate.yaml`** — when you want to verify reconcile logic without a cluster. Runs N cycles, records every Kubernetes op (create/update/delete), and asserts the sequence matches your expectations. Fast feedback, no infra.
