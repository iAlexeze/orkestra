# Secret Distribution Across Namespaces

Distribute a source Secret to multiple namespaces and keep them in sync.

```yaml
secrets:
  - name: db-credentials
    fromSecret: master-db-creds
    fromNamespace: platform
    toNamespaces:
      - "{{ .metadata.namespace }}"
      - monitoring
      - staging
    reconcile: true
```

:::note
When the source Secret rotates, all copies update automatically. Owner references ensure cleanup on CR deletion.
:::

---

## Related Documentation

- **Concept:** [Registry](../runtime-manual/concepts/registry.md)
- **Reference:** [Secret Operations](../reference/registry-schema.md#secrets)
- **Next Use Case:** [Dependency Ordering](./dependency-ordering.md)
