# ork run

Start the Orkestra operator runtime.

```bash
ork run --katalog <path>
```

Merges and validates before starting workers.

Endpoints exposed:

```
/health
/ready
/metrics
/katalog
/katalog/{crd}
/katalog/{crd}/health
```

---

## Related Documentation

- [Runtime](../concepts/runtime.md)
- [Metrics](../reference/metrics.md)
- [ork status](./status.md)
