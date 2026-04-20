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
/katalog/{crd}/cr
/katalog/{crd}/cr/<ns>/<name>
/katalog/{crd}/health
```

---

## Related Documentation

- [Runtime](../../runtime-manual/concepts/runtime.md)
- [Metrics](../metrics.md)
