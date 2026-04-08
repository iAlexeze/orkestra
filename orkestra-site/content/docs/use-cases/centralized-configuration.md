---
title: "Centralized Configuration"
weight: 169
---

# Centralised Operator Configuration (GitOps)

Katalogs are files. Files live in Git. When your CRD configuration is in
Git, it becomes a GitOps artifact — versioned, reviewed, and auditable.

**Platform teams publish standard configurations:**

```
https://internal.company.com/platform/crds/standard-katalog.yaml
```

**Every team consumes it:**

```bash
ork run --katalog https://internal.company.com/platform/crds/standard-katalog.yaml
```

:::note
Changes to the Katalog propagate to every cluster that consumes it on
the next Orkestra restart. No binary rebuilds. No deployments. One file
change, everywhere updated.
:::

---

## Related Documentation

- **Concept:** [Komposer](../runtime-manual/concepts/komposer.md)
- **Reference:** [Komposer Reference](../reference/komposer-schema.md)
- **Next Use Case:** [Environment Overrides](./environment-overrides.md)
