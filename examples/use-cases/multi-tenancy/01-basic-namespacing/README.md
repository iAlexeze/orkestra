# Multi-tenancy 01 — Basic namespacing

Two teams, one runtime. Each Katalog declares `metadata.namespace` and the Control Center renders a separate panel per namespace.

- `platform-team/katalog.yaml` — `namespace: platform-team`, manages a Database CRD
- `product-team/katalog.yaml` — `namespace: product-team`, manages a Website CRD
- `komposer.yaml` — imports both into one runtime

The `/katalog` endpoint groups CRDs by namespace:

```json
{
  "namespaces": {
    "platform-team": { "crds": ["database"], "healthy": true },
    "product-team":  { "crds": ["website"],  "healthy": true }
  }
}
```

`cross:` reads between teams work normally — see example 02 for access control.

## Try it

```bash
ork init my-project --pack use-cases/multi-tenancy/01-basic-namespacing
cd my-project/01-basic-namespacing
ork run -f komposer.yaml
ork control
```
