# Use Cases

Real-world patterns that combine multiple Orkestra features. Each example is self-contained and focused on one scenario.

```bash
ork init my-operator --pack use-cases
```

---

## Examples

| Example | Pattern | What you learn |
|---------|---------|----------------|
| [full-stack-app](./full-stack-app/) | Multi-feature composition | `forEach`, `external`, `cross`, `once`, `or` in one CR |
| [multi-region-map](./multi-region-map/) | Multi-target deployment | `forEach` over a map — deploy to N regions from one CR |
| [crd-conversion](./crd-conversion/) | Schema evolution | Multi-version CRDs with and without a conversion webhook |
| [custom-operator](./custom-operator/) | Third-party test harness | `spec.custom.target: kubernetes` — use `ork e2e` to test any operator |
| [external](./external/) | External gates | Gate resource creation on upstream health checks via `external:` |
| [multi-tenancy](./multi-tenancy/) | Namespace isolation | Per-tenant configuration, `allowedNamespaces`, RBAC isolation |
| [enrich](./enrich/) | Data injection | Inject data from external sources into CR status via `resolver:` |
| [normalize](./normalize/) | Input canonicalization | Validate and normalise CR fields before reconciliation |
| [profiles](./profiles/) | Environment profiles | Apply different resource configurations based on environment |
| [temporal](./temporal/) | Time-dependent workloads | `timeInWindow`, `weekday`, `nextCron` — schedule-driven provisioning, maintenance windows, per-timezone scaling |
| [workload-autoscaler](./workload-autoscaler/) | Declarative replica autoscaling | `autoscale:` on a Deployment — time, external, and cross-operator signals; no KEDA, no ScaledObject |

---

## Run all examples

```bash
ork e2e -f e2e.yaml
```

Each example also has its own `e2e.yaml`:

```bash
cd full-stack-app && ork e2e
cd multi-region-map && ork e2e
```
