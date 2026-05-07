---
title: "Introducing Orkestra: Declarative Kubernetes Operators Without the Go"
date: 2025-09-01
description: "Write a YAML file, get a production-grade Kubernetes operator. No controller code, no informers, no boilerplate."
tags: ["release", "announcement"]
author: "Orkestra Team"
---

Kubernetes operators are powerful — but writing one from scratch means dealing with controllers, informers, workqueues, RBAC, CRD schemas, and hundreds of lines of Go before you've even started on your business logic.

Orkestra flips this model. You write a single YAML file — a **Katalog** — and Orkestra generates and runs the full operator: CRD registration, controller setup, reconciliation loop, status propagation, drift correction, and live observability dashboard.

## The core idea

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: website-operator
spec:
  crds:
    website:
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
      operatorBox:
        default: true
        onCreate:
          deployments:
            - image: "{{ .spec.image }}"
```

That's a complete operator. Apply it, and Orkestra registers the `Website` CRD, starts the reconciler, and launches the Control Center dashboard — all in under a minute.

## What's included

- **Zero-code operators** — declare CRD schema, managed resources, and lifecycle in YAML
- **Dependency ordering** — `dependsOn` between CRDs, startup sequencing guaranteed
- **Drift correction** — `reconcile: true` on any resource keeps it in sync
- **Status propagation** — template expressions written automatically to `.status`
- **Control Center** — live dashboard on port 8081, built into the runtime

## Try it

```bash
ork run --file ./katalog.yaml
```

See the [Getting Started guide](/docs/getting-started/) to build your first operator in under 10 minutes.
