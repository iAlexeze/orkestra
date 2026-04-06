---
title: "Zero‑Code Operators"
weight: 50
description: "The simplest Orkestra pattern: define a CRD, write a Katalog, run the operator."
---

The simplest Orkestra pattern: define a CRD, write a Katalog, run the operator.

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
spec:
  crds:
    - name: website
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
      reconciler:
        default: true
        onCreate:
          deployments:
            - image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              reconcile: true
          services:
            - port: "80"
              targetPort: "{{ .spec.port }}"
              reconcile: true
```

:::note
Every Website CR automatically creates a Deployment and Service, corrects drift, and cleans up on deletion — all without writing Go.
:::

---

## Related Documentation

- **Concept:** [Katalog](../runtime-manual/concepts/katalog.md)
- **Reference:** [Katalog Reference](../reference/katalog-schema.md)
- **Next Use Case:** [Platform Namespace Provisioning](./platform-namespace-provisioning.md)
