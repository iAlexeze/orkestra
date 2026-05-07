---
title: "Progressive Rollout of Operator Behavior"
weight: 50
description: "Canary operator behavior across clusters by pointing to different Katalog URLs."
---

Canary operator behavior across clusters by pointing to different Katalog URLs.

```bash
# 90% stable
ork run --file https://config.company.com/stable/katalog.yaml

# 10% candidate
ork run --file https://config.company.com/candidate/katalog.yaml
```

:::note
Compare reconcile latency, error rates, and resource usage before promoting.
:::

---

## Related Documentation

- **Concept:** [Katalog Lifecycle](../runtime-manual/concepts/katalog.md#lifecycle)
- **Reference:** [Runtime Behavior](../reference/runtime.md)
- **Next Use Case:** [Disaster Recovery](./disaster-recovery.md)

