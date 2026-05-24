---
title: "ork run"
date: 2026-05-20
weight: 51
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
