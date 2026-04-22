---
title: "Multi Team Composition"
weight: 178
---

# Multi‑Team Composition

Large organisations have platform, application, and security teams — each owning their own CRDs.

```yaml
sources:
  files:
    - ./katalogs/namespaces.yaml
    - https://raw.github.com/myorg/app-crds/main/katalog.yaml
    - $SECURITY_KATALOG_URL
```

:::tip
Each team owns its Katalog. The Komposer composes them. No cross‑repo access required.
:::

---

## Related Documentation

- **Concept:** [Komposer](../runtime-manual/concepts/komposer.md)
- **Reference:** [File & URL Sources](../reference/komposer-schema.md#files)
- **Next Use Case:** [Progressive Rollout](./progressive-rollout.md)

