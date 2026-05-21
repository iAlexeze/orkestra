---
title: "Declarative State Machines in Orkestra"
date: 2025-10-15
description: "How Orkestra models operator lifecycle as explicit state machines defined in YAML — and why that makes operators dramatically more reliable."
tags: ["engineering", "concepts"]
author: "Orkestra Team"
---

One of Orkestra's core design principles is that every CRD lifecycle should be an **explicit state machine** — not implicit logic buried in Go code.

In traditional operators, state transitions are encoded in the `Reconcile` function. You have to read the code to understand what states exist, what triggers transitions, and what happens on failure. Orkestra makes all of this visible in the Katalog.

## State machines in YAML

```yaml
spec:
  crds:
    database:
      operatorBox:
        default: true
        onCreate:
          deployments:
            - image: "postgres:{{ .spec.version }}"
              reconcile: true
        status:
          fields:
            - path: phase
              value: "Running"
            - path: endpoint
              value: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc"
```

The `onCreate`, `onReconcile`, and `onDelete` hooks define the state transitions. The `status.fields` declarations define what gets written after each successful transition.

## Why it matters

When your state machine is in YAML:

1. **It's auditable** — any engineer can read the Katalog and understand the full lifecycle
2. **It's testable** — state transitions can be validated without running a full cluster
3. **It's composable** — Katalogs can be layered via the Komposer registry

Read more in the [Katalog concepts documentation](/docs/concepts/katalog/).
