# Progressive Rollout of Operator Behavior

Canary operator behavior across clusters by pointing to different Katalog URLs.

```bash
# 90% stable
ork run --katalog https://config.company.com/stable/katalog.yaml

# 10% candidate
ork run --katalog https://config.company.com/candidate/katalog.yaml
```

!!! note
    Compare reconcile latency, error rates, and resource usage before promoting.

---

## Related Documentation

- **Concept:** [Katalog Lifecycle](../concepts/katalog.md#lifecycle)
- **Reference:** [Runtime Behavior](../reference/runtime.md)
- **Next Use Case:** [Disaster Recovery](./disaster-recovery.md)

