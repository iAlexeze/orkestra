# Multi‑Version CRDs Explained

Kubernetes allows CRDs to have multiple API versions:

- `v1alpha1` (experimental)
- `v1beta1` (evolving)
- `v1` (stable)

Orkestra supports multiple versions **simultaneously**, with automatic conversion between them.

This page explains how it works.

---

## Why Multiple Versions?

Multiple versions allow you to:

- Evolve your API safely  
- Introduce new fields  
- Deprecate old fields  
- Maintain backward compatibility  

Users can choose the version they prefer.

---

## Storage Version

Kubernetes stores CRs in a **single version**, called the *storage version*.

Example:

- You define `v1alpha1` and `v1`
- You choose `v1` as the storage version
- Kubernetes stores everything as `v1`

Even if the user applies a `v1alpha1` object.

---

## Conversion Webhooks

When a user requests a different version:

- Kubernetes calls Orkestra’s conversion webhook  
- Orkestra converts the object between versions  
- The user receives the version they asked for  

Example:

```
kubectl get website my-blog --api-version=demo.orkestra.io/v1alpha1
```

Kubernetes:

1. Reads the stored `v1` object  
2. Sends it to Orkestra  
3. Orkestra converts it to `v1alpha1`  
4. Kubernetes returns it to the user  

---

## Declarative Conversion Rules

You define conversion rules in your katalog:

```yaml
conversion:
  from: v1alpha1
  to: v1
  rules:
    - map: theme -> seo.enabled=false
```

Orkestra handles:

- Conversion logic  
- Validation  
- Metrics  
- Error handling  
- Webhook registration  
- TLS  
- CA bundles  

No Go code required.

---

## Why This Matters

Multi‑version support allows you to:

- Introduce new API versions without breaking users  
- Migrate CRs gradually  
- Support multiple clients  
- Evolve your operator safely  

Orkestra makes this process:

- Declarative  
- Observable  
- Testable  
- Automatic  

:::tip[Practical Usecase]
See how it works in the next page
:::