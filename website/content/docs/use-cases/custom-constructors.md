---
title: "Custom Constructors"
weight: 50
description: "Use a constructor when you need:"
---

Use a constructor when you need:

- Complex state machines  
- Custom retry/backoff  
- Multi‑step finalizer orchestration  
- Migration of an existing controller  

```yaml
reconciler:
  default: false
  constructor:
    location: github.com/myorg/reconcilers
    function: NewApplicationReconciler
```

:::caution
You own the reconcile loop — Orkestra handles everything around it (informers, workqueue, metrics, health, leader election).
:::

