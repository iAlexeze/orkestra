# Advanced Usecases — Orkestra Declarative Patterns

Six examples showing what Orkestra can express declaratively that previously required custom Go code. Each example isolates one pattern. Example 06 combines all five.

| Example | Pattern | What it replaces |
|---|---|---|
| [01 — Multi-Region](01-multi-region/README.md) | `forEach` over a list | `for _, region := range regions { client.Create(...) }` |
| [02 — External Gate](02-external-gate/README.md) | `external:` health check | `http.Get + if status != 200 { return err }` |
| [03 — Cross-CRD](03-cross-crd/README.md) | `cross:` observation | `client.Get(otherCRD) + if notFound { requeue }` |
| [04 — Once Secret](04-once-secret/README.md) | `once:` + random generation | `crypto/rand + secretExists check` |
| [05 — or](05-or/README.md) | OR conditions | `if phase == Failed \|\| phase == Succeeded` |
| [06 — Full Stack](06-full-stack/README.md) | All five combined | Everything above in one CR |

Each subfolder has its own `katalog.yaml` and `README.md` — run any example in isolation from inside the subfolder. To run all six at once as one operator:

```bash
cd full-stack-app
```

Examples 02 and 06 call external HTTP endpoints — start the mock dev server first:

```bash
ork run --dev-server
```

---

**Next step:** [multi-region-map](../multi-region-map/README.md) — forEach over a map with per-region replica counts and ports, plus a real app you can port-forward to.

---

## Troubleshooting

### Duplicate CRD error when running `ork validate` from this directory

```
duplicate CRD "managed-database": defined in ".../03-cross-crd/katalog.yaml"
and ".../06-full-stack/katalog.yaml" — names must be unique across all imports
```

This happens when `06-full-stack/katalog.yaml` has the `managed-database:` block **uncommented**. The root composite already imports `managed-database` from `03-cross-crd` — having it declared again in `06-full-stack` creates a collision.

**Fix:** comment out the `managed-database:` block in `06-full-stack/katalog.yaml` and re-run `ork validate` from the root:

```yaml
# 06-full-stack/katalog.yaml
spec:
  crds:
    # managed-database:       ← keep this commented when running from full-stack-app/
    #   crdFile: ../03-cross-crd/crd-managed-database.yaml
    #   ...

    full-stack-app:
      ...
```

The `managed-database:` block only needs to be uncommented when running `06-full-stack` **in isolation** (i.e. `cd 06-full-stack && ork validate`). See [06-full-stack/README.md](06-full-stack/README.md#step-1--validate).
