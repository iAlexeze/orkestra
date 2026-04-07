---
title: "Conditional Webhooks"
weight: 111
---

# Orkestra Conversion Webhook  
### Declarative, Optional, and Secure CRD Version Conversion

Orkestra introduces a **conditional conversion webhook** that enables fully declarative CRD version conversion without requiring users to write Go code, maintain versioned structs, or implement manual conversion functions. This mechanism is optional, secure, and activated only when explicitly enabled.

This document explains:

- **Why** the conversion webhook exists  
- **What** problem it solves  
- **How** it is implemented inside Orkestra  
- **How** it is enabled using 
- **How** TLS is provided by the user for secure webhook communication  

---

## Why Orkestra Has a Conversion Webhook

Kubernetes supports multiple versions of a CRD (e.g., `v1alpha1`, `v1`, `v2`), but it requires a **conversion webhook** to translate objects between versions. Traditionally, this forces operator authors to:

- Write versioned Go structs  
- Maintain manual conversion functions  
- Register schemes and deep‑copy logic  
- Ship a webhook server  
- Manage TLS certificates  
- Handle storage version semantics  

This is one of the **most painful** parts of building Kubernetes operators.

Orkestra eliminates all of that.

{{< callout type="tip" >}}
### Orkestra’s goal:
Make CRD versioning fully declarative — no Go code, no conversion functions, no boilerplate.
{{< /callout >}}

To achieve this, Orkestra needs a webhook endpoint that Kubernetes can call whenever it needs to convert an object between versions. But unlike traditional operators, Orkestra’s webhook is:

- **Declaratively driven**  
- **Runtime‑enabled**  
- **User‑controlled**  
- **Secure by design**  

This is where the conditional conversion webhook comes in.

---

## What Problem the Conditional Webhook Solves
With Orkestra, users can define conversion rules directly in the Katalog:

```yaml
conversion:
  storageVersion: v1
  paths:
    - from:
        version: v1alpha1
        spec:
          image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
          seo:
            enabled: false
    - to:
        version: v1alpha1
        spec:
          image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
          theme: "default"
```

Orkestra interprets these rules and performs conversion automatically.

### The webhook is the bridge  
Kubernetes → Orkestra → Declarative conversion rules → Kubernetes

The conditional webhook allows Orkestra to:

- Convert CRs between versions  
- Store CRs in the canonical storage version  
- Serve CRs in any version  
- Keep multi‑version CRDs fully functional  
- Avoid writing any Go conversion code  

This is the foundation of Orkestra’s **zero‑code operator model**.

---

## How the Conditional Webhook Works Internally

The webhook is implemented inside Orkestra’s `HealthServer` component.  
It exposes two HTTP servers:

| Purpose | Port | Protocol |
|--------|------|----------|
| Health + readiness | `8080` | HTTP |
| Conversion webhook | `8443` | HTTPS |

The HTTPS server is **not started by default**.

### It is only started when:

1. `ENABLE_CONVERSION=true` and/or `ENABLE_ADMISSION_WEBHOOK=true`
2. Valid TLS certificate + key paths are provided

Either flag alone is sufficient. Both flags share the same HTTPS server on `:8443`.

| Flag | Routes registered |
|---|---|
| `ENABLE_CONVERSION=true` | `/convert` |
| `ENABLE_ADMISSION_WEBHOOK=true` | `/validate`, `/mutate` |
| Both | All three routes |

If neither flag is set, Orkestra runs normally without exposing port 8443.

---

## Why TLS Is Required (and Provided by the User)

Kubernetes **requires** conversion webhooks to use HTTPS with valid certificates.  
This ensures:

- API server → webhook communication is encrypted  
- The webhook identity is verifiable  
- No man‑in‑the‑middle attacks  
- No plaintext CRD data on the network  

Orkestra does **not** generate TLS certificates automatically.  
Instead, the user provides them via a Kubernetes Secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: orkestra-tls
  namespace: orkestra
type: kubernetes.io/tls
data:
  tls.crt: <base64>
  tls.key: <base64>
```

These are mounted into the container:

```yaml
volumeMounts:
  - name: tls
    mountPath: /tls
    readOnly: true
```

And passed to Orkestra via environment variables:

```yaml
env:
  - name: TLS_CERT
    value: "/tls/tls.crt"
  - name: TLS_KEY
    value: "/tls/tls.key"
```

This gives users full control over certificate rotation, trust chains, and CA bundles.

---

## How to Enable the Conversion Webhook

To activate the conversion webhook, set:

```yaml
env:
  - name: ENABLE_CONVERSION
    value: "true"
```

And mount TLS:

```yaml
volumeMounts:
  - name: tls
    mountPath: /tls
    readOnly: true
```

Orkestra will:

1. Detect `ENABLE_CONVERSION=true`
2. Validate that `TLS_CERT` and `TLS_KEY` are provided
3. Start the HTTPS server on port 8443
4. Register the `/convert` handler
5. Serve conversion requests from the Kubernetes API server

To also enable admission webhooks (validation and mutation), add:

```yaml
env:
  - name: ENABLE_ADMISSION_WEBHOOK
    value: "true"
```

The same HTTPS server serves all three routes. Either flag alone starts it.

If neither flag is set and TLS is not provided, the HTTPS server is **not started**, and Orkestra continues running on HTTP only.

---

## Summary

Here’s the entire story in one clean block:

- Orkestra supports **declarative CRD version conversion** and **admission-time policy**
- Kubernetes requires a webhook for both conversion and admission enforcement
- Orkestra provides a **conditional HTTPS server** that only runs when enabled
- Conversion is activated via `ENABLE_CONVERSION=true`
- Admission webhooks are activated via `ENABLE_ADMISSION_WEBHOOK=true`
- Both require user-provided TLS certificates and share the same server on `:8443`
- The `/convert`, `/validate`, and `/mutate` endpoints are registered only when their
  respective feature is enabled
- All behavior is declared in the Katalog — no Go code, no webhook servers to maintain

This is one of the core innovations that makes Orkestra a **zero‑code operator engine**.
