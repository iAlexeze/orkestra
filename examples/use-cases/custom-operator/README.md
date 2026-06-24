# Custom Target Use Cases

`custom.target: kubernetes` — use `ork e2e` as a test harness for any Kubernetes operator,
not just Orkestra. Bundle generation and Orkestra helm install are skipped; cluster
setup, assertions, and cleanup all run unchanged.

| Example | What it shows |
|---------|---------------|
| [01 — Pure Custom (cert-manager)](01-pure-custom/README.md) | cert-manager installed via `setup.helm`. Applies a `Certificate` CR, asserts the TLS Secret is created. No Orkestra involved. |
| [02 — Side-by-Side (Migration Parity)](02-side-by-side/README.md) | Same `Website` CRD tested two ways — Orkestra katalog and `custom.target: kubernetes` with your operator. Identical assertions. When both pass, migration is verified. |

---

## When to use `custom.target: kubernetes`

- Testing an operator you didn't write (FluxCD, cert-manager, Crossplane, ArgoCD)
- Migration testing — verify your old and new operators produce identical results
- Two-operator composition — assert two operators interact correctly
- Any time you want Orkestra's assertion infrastructure without Orkestra's reconcile loop

See [Custom Target](https://orkestra.sh/docs/reference/schema/e2e/custom-target).

---

## E2E

Run a single example:

```bash
cd 01-pure-custom && ork e2e
```

Run the full suite:

From the root directory, run:

```bash
ork e2e
```
