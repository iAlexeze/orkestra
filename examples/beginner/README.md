# Beginner Pack

The foundation. Every concept here appears in every more advanced example. Work through these in order — each one builds on the last.

```bash
ork init --pack beginner
```

---

## Examples

| Example | What it teaches |
|---------|-----------------|
| [01 — Hello Website](01-hello-website/README.md) | Your first operator. CRD declaration, Katalog template expressions, owner references, status fields. The mental model everything else builds on. |
| [02 — Website with ServiceAccount](02-with-serviceaccount/README.md) | Three resources from one CR. ServiceAccount wiring, pod identity, reading live cluster state into status via Notes. |
| [03 — Copy Secret Across Namespaces](03-secret-copy/README.md) | Built-in Kubernetes resource management. A Secret distribution operator: copies a Secret from a platform namespace into every team namespace. `fromSecret`, `toNamespaces`. |
| [03b — Copy ConfigMap Across Namespaces](03b-configmap-copy/README.md) | Same distribution pattern applied to ConfigMaps. Statusless resource distribution. Good companion to 03. |

---

## Running an example

```bash
ork init --pack beginner
cd beginner/01-hello-website
ork run
```

No cluster? Add `--dev` to create a temporary kind cluster.

---

## Simulate

Some examples ship with a `simulate.yaml`. Run simulate for a single example:

```bash
cd 02-with-serviceaccount && ork simulate
```

Or run the full beginner suite — all examples that have a simulate.yaml:

```bash
ork simulate -f simulate.yaml
```

No cluster needed. Each run completes in under a second.

---

## E2E

Every example ships with a runnable `e2e.yaml`. Run a single example end-to-end:

```bash
cd 01-hello-website && ork e2e
```

Or run the full beginner suite — all four examples in one kind cluster:

```bash
ork e2e -f e2e.yaml
```

The suite spins up a cluster, runs each example in sequence, and tears everything down. Total runtime: ~5 minutes.
