# Multi-tenancy

One Orkestra runtime serves multiple teams. Each Katalog declares `metadata.namespace` — a logical tenant scope. The runtime runs all CRDs with independent workers, health tracking, and reconcile loops. The Control Center renders one panel per namespace.

---

## Namespaces and cluster name

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: payments
  namespace: fintech-team
  clusterName: prod-eu
```

`namespace` is a logical grouping — it is not a Kubernetes namespace. Omitting it defaults to `"default"`.

`clusterName` identifies which cluster this Katalog runs in. The Control Center uses it to filter across multiple connected runtimes. When set in the Katalog it takes precedence over the `CLUSTER_NAME` environment variable. When neither is set it is omitted from the response.

Declaring both gives the Control Center full coordinates for every CRD: cluster → namespace → CRD.

---

## Composing namespaced Katalogs

A Komposer imports multiple Katalogs. Each Katalog keeps its own namespace:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: platform
imports:
  files:
    - url: ./platform-team/katalog.yaml
    - url: ./product-team/katalog.yaml
spec:
  crds: {}
```

The `/katalog` endpoint returns a `namespaces` map:

```json
{
  "namespaces": {
    "platform-team": { "crds": ["database", "cache"], "healthy": true },
    "product-team":  { "crds": ["website", "api"],   "healthy": false }
  }
}
```

---

## Cross-read access control

Any CRD can read another CRD's CR state via `cross:` by default. Declare `crossAccess: false` on a Katalog to close it:

```yaml
crossAccess: false

spec:
  crds:
    payment: {}         # closed — inherits Katalog default
    ledger:
      crossAccess: true # open — overrides Katalog default
```

A `cross:` reference to a closed CRD returns `found: "false"` silently. Use `when: cross.xyz.found == "true"` to gate dependent resources.

---

## Try it

```bash
ork init my-project --pack use-cases/multi-tenancy
cd my-project/multi-tenancy

# Follow the steps in the README
```

| | |
|---|---|
| `01-basic-namespacing` | Two teams, separate CC panels |
| `02-cross-access-control` | `crossAccess: false` with CRD-level override |
| `03-shared-platform` | Platform infra consumed by application teams |
