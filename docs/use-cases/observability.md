# Observability

Every Orkestra operator exposes the same endpoints:

```
GET /health
GET /ready
GET /metrics
GET /katalog
GET /katalog/{crd}
GET /katalog/{crd}/health
```

!!! tip
    Metrics integrate directly with Prometheus and Grafana.

---

## Related Documentation

- **Concept:** [Observability](../concepts/observability.md)
- **Reference:** [Metrics & Health](../reference/runtime.md#observability)
- **Next Use Case:** [Registry‑Powered Operators](./registry.md)

