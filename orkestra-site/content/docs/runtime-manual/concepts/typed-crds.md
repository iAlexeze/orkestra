---
title: "Typed Crds"
weight: 147
---

# **Typed CRDs**

Typed CRDs allow Orkestra operators to use **Go structs**, **strongly‑typed reconcilers**, and **custom logic** alongside declarative templates.  

{{< callout type="note" >}}
They are optional — Orkestra supports both dynamic (YAML‑only) and typed (Go‑based) operators.
{{< /callout >}}

Typed CRDs are ideal when you need:

- custom validation  
- computed defaults  
- complex status fields  
- external API calls  
- advanced reconciliation logic  
- Go hooks or constructors  

{{< callout type="tip" >}}
Dynamic operators remain zero‑code. Typed operators add Go where it matters.
{{< /callout >}}

---

# **Why Typed CRDs Exist**

Dynamic operators are powerful, but some use cases require more control:

- computing derived fields  
- validating nested structures  
- calling external services  
- writing complex status conditions  
- orchestrating multi‑step workflows  
- integrating with existing Go libraries  

{{< callout type="note" >}}
Typed CRDs give you the full power of the Kubernetes API machinery while still benefiting from Orkestra’s declarative model.
{{< /callout >}}

---

# **How Typed CRDs Work**

A typed CRD is defined by setting:

```yaml
apiTypes:
  group: demo.orkestra.io
  version: v1alpha1
  kind: Website
  plural: websites
  location: github.com/myorg/my-operator/pkg/apis/website/v1alpha1
```

{{< callout type="note" >}}
The `location` field tells Orkestra where your Go types live.
{{< /callout >}}

During `ork generate runtime`, Orkestra:

- registers your Go types  
- adds them to the scheme  
- wires them into the runtime registry  
- enables typed hooks and constructors  

Typed CRDs behave exactly like dynamic CRDs — they simply have a richer internal representation.

---

# **Typed Reconciliation**

Typed CRDs unlock two advanced features:

## **1. Go Hooks**

Hooks run before declarative templates:

```yaml
operatorBox:
  hooks:
    location: github.com/myorg/my-operator/pkg/hooks
    function: ReconcileWebsite
```

Hooks receive:

- the typed CR  
- the context  
- the runtime registry  
- the logger  

They can:

- mutate the CR  
- call external APIs  
- compute status  
- create or update resources via the Registry  

---

## **2. Custom Constructors**

Constructors replace the GenericReconciler entirely:

```yaml
operatorBox:
  default: false
  constructor:
    location: github.com/myorg/my-operator/pkg/runtime
    function: NewWebsiteReconciler
```

Use a constructor when:

- you need a fully custom reconciler  
- you want to integrate an existing controller  
- you need advanced orchestration logic  

---

# **Typed CRD Workflow**

A typical typed operator workflow:


1. Scaffold a typed project:

```bash
   ork init my-operator --typed
```

2. Define your Go types under `pkg/apis/...`

3. Reference them in your Katalog:

```yaml
apiTypes:
  group: demo.orkestra.io
  version: v1alpha1
  kind: Website
  plural: websites
  location: github.com/myorg/my-operator/pkg/apis/website/v1alpha1
```

4. Generate runtime wiring:

```bash
ork generate runtime --katalog katalogs/website.yaml
```

5. Run the operator:

```bash
go run ./cmd/orkestra/ run --katalog katalogs/website.yaml
```

{{< callout type="note" >}}
Typed CRDs integrate seamlessly with declarative templates — you can mix both in the same operator.
{{< /callout >}}

---

# **Typed vs Dynamic CRDs**

| Feature | Dynamic | Typed |
|--------|---------|-------|
| YAML‑only | Yes | No |
| Go types | No | Yes |
| Hooks | Optional | Fully supported |
| Constructors | No | Yes |
| External API calls | No | Yes |
| Complex status logic | Limited | Full control |
| Drift correction | Yes | Yes |
| Zero‑code | Yes | No |

{{< callout type="tip" >}}
Use dynamic CRDs for simple declarative operators.  
Use typed CRDs when you need custom logic.
{{< /callout >}}

---

# **When to Choose Typed CRDs**

Choose typed CRDs when:

- you need to compute fields dynamically  
- you need to validate complex structures  
- you need to call external systems  
- you need to write custom status conditions  
- you want to reuse existing Go libraries  
- you want full control over reconciliation  

{{< callout type="tip" >}}
If your operator is mostly declarative, dynamic CRDs are simpler and faster.
{{< /callout >}}

---

# **Summary**

Typed CRDs provide:

- strong typing  
- custom logic  
- Go hooks and constructors  
- full control over reconciliation  
- seamless integration with declarative templates  

They are the bridge between Orkestra’s declarative model and the full power of Kubernetes controller development.

---

## Related Documentation

- [Hooks](./hooks.md)
- [Constructors](./constructors.md)
- [Versioning](./versioning.md)
- [Katalog Schema](../../reference/katalog-schema.md)
