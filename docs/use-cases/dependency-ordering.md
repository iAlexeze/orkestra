# Multi‑CRD Dependency Ordering

Orkestra ensures CRDs start in the correct order and shut down in reverse.

```yaml
crds:
  project:
  managednamespace:
    dependsOn: [project]
  application:
    dependsOn: [project, managednamespace]
```

:::tip
Missing CRDs don’t block startup — Orkestra activates them when they appear and unblocks dependents automatically.
:::

---

## Related Documentation

- **Concept:** [Dependency Graph](../runtime-manual/concepts/dependency-model.md)
- **Reference:** [CRD Configuration](../reference/katalog-schema.md#crds)
- **Next Use Case:** [Centralised Configuration](./centralized-configuration.md)
