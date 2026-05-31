# Advanced Usecases — Orkestra Declarative Patterns

Six examples showing what Orkestra can express declaratively that previously required custom Go code. Each example isolates one pattern. Example 06 combines all five.

| Example | Pattern | What it replaces |
|---|---|---|
| [01 — Multi-Region](01-multi-region/README.md) | `forEach` over a list | `for _, region := range regions { client.Create(...) }` |
| [02 — External Gate](02-external-gate/README.md) | `external:` health check | `http.Get + if status != 200 { return err }` |
| [03 — Cross-CRD](03-cross-crd/README.md) | `cross:` observation | `client.Get(otherCRD) + if notFound { requeue }` |
| [04 — Once Secret](04-once-secret/README.md) | `once:` + random generation | `crypto/rand + secretExists check` |
| [05 — anyOf](05-anyof/README.md) | OR conditions | `if phase == Failed \|\| phase == Succeeded` |
| [06 — Full Stack](06-full-stack/README.md) | All five combined | Everything above in one CR |

Each subfolder has its own `katalog.yaml` and `README.md` — run any example in isolation from inside the subfolder. To run all six at once as one operator:

```bash
cd full-stack-app
```

Examples 02 and 06 call external HTTP endpoints — start the mock dev server first:

```bash
ork run --dev-server
```

Example 06 depends on the `ManagedDatabase` from example 03 — apply `03-cross-crd/database-cr.yaml` before applying the full-stack CR.

---

**Next step:** [multi-region-map](../multi-region-map/README.md) — forEach over a map with per-region replica counts and ports, plus a real app you can port-forward to.
