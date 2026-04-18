# Environment-specific Tuning

Different environments need different settings. A Komposer lets you
compose the shared Katalog with environment-specific overrides without
forking it.

```yaml
# production-komposer.yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: production-komposer

sources:
  files:
    - https://internal.company.com/platform/crds/standard-katalog.yaml

spec:
  crds:
    # Override — production needs more workers
    application:
      workers: 10
      resync: 30s
      apiTypes:
        group: platform.myorg.io
        version: v1alpha1
        kind: Application
        plural: applications
      operatorBox:
        default: true
```

:::note
Development uses `workers: 2`. Production uses `workers: 10`. The same
source Katalog, two different overrides, no fork.
:::

---

## Related Documentation

- **Concept:** [Komposer](../runtime-manual/concepts/komposer.md)
- **Reference:** [Komposer Overrides](../reference/komposer-schema.md#overrides)
- **Next Use Case:** [Helm‑Driven Operators](./helm-driven-operators.md)
