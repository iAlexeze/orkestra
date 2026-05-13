# Disaster Recovery

A cluster can be fully restored from a Katalog.

```bash
ork run --file https://git.company.com/platform/crds/prod-katalog.yaml
```

:::tip
The Katalog *is* the recovery plan — no binary rebuilds or config migrations.
:::

---

## Related Documentation

- **Concept:** [Katalog](../runtime-manual/concepts/katalog.md)
- **Reference:** [Runtime Startup](../reference/runtime.md#startup)
- **Next Use Case:** [Air‑Gapped Environments](./air-gapped.md)
