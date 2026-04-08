---
title: "Hooks"
weight: 119
---

# Hooks

Hooks let you attach custom Go logic to Orkestra’s declarative reconciliation model.  
They run alongside templates and can read or modify resources, call external systems, and compute status.

{{< callout type="note" >}}
Hooks are optional. They are most useful when you need logic that cannot be expressed purely in YAML.
{{< /callout >}}

---

## Why hooks exist

Declarative templates are powerful for:

- creating Kubernetes resources  
- enforcing drift correction  
- expressing conditional provisioning  

But some behaviors require code:

- calling external APIs  
- computing derived fields  
- validating complex combinations of fields  
- writing rich status conditions  
- orchestrating multi‑step workflows  

{{< callout type="tip" >}}
Hooks provide a focused way to add code without abandoning the declarative model.
{{< /callout >}}

---

## Declaring hooks in the Katalog

Hooks are configured per‑CRD in the Katalog:

```yaml
reconciler:
  hooks:
    location: github.com/myorg/my-operator/pkg/hooks
    function: ReconcileWebsite
    alias: websiteHooks
```

- `location` is the Go module path where the hook function lives.  
- `function` is the exported Go function name.  
- `alias` is an optional label used in generated code and logs.

When hooks are declared, you must run:

```bash
ork generate runtime --katalog <path>
```

to generate the runtime registry wiring.

---

## Hook execution model

During reconciliation, Orkestra:

1. Loads the CR from the informer cache  
2. Converts it to the internal version  
3. Runs the hook (if configured)  
4. Applies declarative templates (if configured)  
5. Updates status and metrics  

Hooks can:

- read and modify the CR  
- interact with the Kubernetes API  
- call external services  
- write status conditions  
- decide whether to requeue  

They do not replace templates; they complement them.

---

## Hook function shape

A typical hook function looks like this (conceptually):

```go
func ReconcileWebsite(ctx context.Context, r *RuntimeContext, obj *v1alpha1.Website) error {
    // read spec
    // compute status
    // call external APIs
    // update resources via the registry
    return nil
}
```

The exact signature depends on the runtime helper types you define, but the pattern is:

- context  
- runtime or registry handle  
- typed CR object  

---

## When to use hooks

Use hooks when:

- you need to call external systems (for example, DNS, payment providers, internal APIs)  
- you need to compute complex status conditions  
- you need to validate or normalize fields beyond what CRDs can express  
- you want to implement custom backoff or retry logic  

If your operator is purely declarative, you can skip hooks entirely.

---

## Hooks and typed CRDs

Hooks work best with typed CRDs:

- the hook receives a typed Go struct  
- you get compile‑time safety  
- you can reuse existing Go libraries  

You can technically use hooks with dynamic CRDs, but typed CRDs are the recommended pairing.

---

## Summary

Hooks provide:

- targeted custom logic  
- integration with external systems  
- richer validation and status handling  
- a bridge between declarative templates and imperative code  

They let you extend Orkestra’s model without giving up its benefits.

---

## Related Documentation

- [Typed CRDs](./typed-crds.md)
- [Constructors](./constructors.md)
- [Katalog Schema](../../reference/katalog-schema.md)
