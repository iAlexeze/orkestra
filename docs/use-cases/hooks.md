# Go Hooks

Use hooks when you need:

- Type‑safe access to the CR  
- External API calls  
- Status writes  
- Direct access to the Registry  

```yaml
reconciler:
  default: true
  hooks:
    location: github.com/myorg/hooks
    function: WebsiteHooks
```

:::note
Hooks run inside the GenericReconciler — you keep finalizers, metrics, events, and the workqueue.
:::

---

## Related Documentation

- **Concept:** [Hooks](../runtime-manual/concepts/hooks.md)
- **Reference:** [Hook Configuration](../reference/katalog-schema.md#hooks)
- **Next Use Case:** [Custom Constructors](./custom-constructors.md)
