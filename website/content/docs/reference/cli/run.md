---
title: "ork run"
weight: 41
---

Start the Orkestra Runtime.

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
