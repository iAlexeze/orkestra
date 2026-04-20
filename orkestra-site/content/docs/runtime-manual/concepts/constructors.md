---
title: "Constructors"
weight: 115
---

# Constructors

Constructors let you replace Orkestra’s GenericReconciler with a fully custom reconciler implementation written in Go.

{{< callout type="tip" >}}
Where hooks extend the default behavior, constructors **take over** reconciliation for a CRD.
{{< /callout >}}

---

## Why constructors exist

Most operators can be expressed using:

- declarative templates  
- conditional provisioning  
- hooks for targeted logic  

However, some use cases require full control:

- integrating an existing controller or operator  
- implementing complex state machines  
- managing non‑Kubernetes backends with custom semantics  
- using advanced caching or sharding strategies  

{{< callout type="note" >}}
Constructors give you the ability to plug in a custom reconciler while still benefiting from Orkestra’s Katalog, Komposer, and runtime registry.
{{< /callout >}}

---

## Declaring a constructor in the Katalog

Constructors are configured per‑CRD:

```yaml
operatorBox:
  default: false
  constructor:
    location: github.com/myorg/my-operator/pkg/runtime
    function: NewWebsiteReconciler
    alias: websiteReconciler
```

- `default: false` tells Orkestra not to use the GenericReconciler.  
- `location` is the Go module path where your constructor lives.  
- `function` is the exported Go function that returns a reconciler.  
- `alias` is an optional label used in generated code and logs.

After declaring a constructor, run:

```bash
ork generate registry --katalog <path>
```

to generate the runtime registry wiring.

---

## Constructor function shape

A constructor typically returns a controller‑runtime‑style operatorBox:

```go
func NewWebsiteReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
    // set up clients, caches, dependencies
    return &WebsiteReconciler{
        client: mgr.GetClient(),
        scheme: mgr.GetScheme(),
    }, nil
}
```

Orkestra calls this constructor during startup and registers the resulting reconciler for the CRD.

The exact signature depends on how you structure your runtime package, but the pattern is:

- accept a runtime or manager context  
- return a reconciler and an error  

---

## Behavior differences vs hooks

| Aspect | Hooks | Constructors |
|--------|-------|--------------|
| GenericReconciler used | Yes | No |
| Templates supported | Yes | Only if you call them yourself |
| Control level | Partial | Full |
| Integration effort | Low | Higher |
| Best for | Extending behavior | Replacing behavior |

{{< callout type="tip" >}}
Hooks are additive. Constructors are authoritative.
{{< /callout >}}

If you use a constructor, you are responsible for:

- reconciliation logic  
- status updates  
- drift correction (if desired)  
- error handling and requeue behavior  

---

## When to use constructors

Use constructors when:

- you already have a controller you want to reuse  
- you need behavior that does not fit the GenericReconciler model  
- you want to manage non‑Kubernetes systems with custom semantics  
- you need advanced performance tuning or sharding strategies  

{{< callout type="tip" >}}
If you can express your behavior with templates and hooks, prefer those first — they are simpler and more declarative.
{{< /callout >}}

---

## Constructors and typed CRDs

Constructors are almost always used with typed CRDs:

- your reconciler works with typed Go structs  
- you can reuse existing controller‑runtime patterns  
- you get full control over reconciliation while still using Orkestra for configuration and wiring  

The Katalog defines the CRDs and wiring; your constructor defines the behavior.

---

## Summary

Constructors provide:

- full control over reconciliation  
- integration with existing controllers  
- the ability to replace the GenericReconciler entirely  
- a way to use Orkestra as a runtime and configuration layer around custom logic  

They are the most powerful — and most advanced — extension point in Orkestra.

---

## Related Documentation

- [Typed CRDs](./typed-crds.md)
- [Hooks](./hooks.md)
- [Katalog Schema](../../reference/katalog-schema.md)
