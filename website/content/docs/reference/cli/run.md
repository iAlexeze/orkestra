---
title: "ork run"
date: 2026-05-25
weight: 96
---

Start the Orkestra Runtime.

```bash
ork run --file <path>
```

Merges and validates before starting workers.

Endpoints exposed:

```text
/health
/ready
/metrics
/katalog
/katalog/{crd}
/katalog/{crd}/cr
/katalog/{crd}/cr/<ns>/<name>
/katalog/{crd}/health
```
