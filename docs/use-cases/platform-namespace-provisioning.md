# Platform Namespace Provisioning

A universal platform engineering pattern: create namespaces, copy secrets, configure metadata, and create workload identities.

```yaml
onCreate:
  configMaps:
    - name: "{{ .metadata.name }}-config"
      namespace: "{{ .spec.targetNamespace }}"
      data:
        ENVIRONMENT: "{{ .spec.environment }}"
        LOG_LEVEL: "{{ .spec.logLevel }}"
        TEAM: "{{ .spec.team }}"
      reconcile: true

  secrets:
    - name: registry-pull-secret
      fromSecret: docker-registry-creds
      fromNamespace: platform
      namespace: "{{ .spec.targetNamespace }}"
      reconcile: true

  serviceAccounts:
    - name: "{{ .spec.team }}-sa"
      namespace: "{{ .spec.targetNamespace }}"
```

:::tip[What you just built]
This replaces multiple controllers (Namespace Configurator, ClusterSecret, custom scripts) with a single declarative entry.
:::

---

## Related Documentation

- **Concept:** [Templating Engine](../runtime-manual/concepts/templating.md)
- **Reference:** [Registry Reference](../reference/registry-schema.md)
- **Next Use Case:** [Secret Distribution](./secret-distribution.md)
