---
title: "ork run"
weight: 31
---

# ork run

Start the Orkestra operator runtime.

```bash
ork run --file <path>
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

- [Runtime Reference](../runtime/)
- [Metrics](../metrics/)
