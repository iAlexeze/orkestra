---
title: "Evolve APIs Without Webhooks or Code"
weight: 3
description: "Orkestra lets you define and evolve multi-version Kubernetes CRDs entirely in YAML."
---

Orkestra lets you define and evolve multi-version Kubernetes CRDs entirely in YAML.

No conversion webhooks.
No Go code.
No extra services.

You declare versions and how they map — Orkestra handles conversion automatically.

---

## Why This Exists

Kubernetes supports multi-version CRDs, but in practice teams avoid them.

Why?

Because versioning requires:

* Writing conversion logic in Go
* Running a separate webhook service
* Managing TLS certificates
* Maintaining bidirectional mappings across versions

For most teams, this is too much overhead for what should be a simple change.

So APIs don’t evolve — they stagnate.

---

## What Orkestra Does Differently

Orkestra treats **each version as a first-class operator** and lets you define:

* Versions as separate entries
* Conversion rules as YAML
* Shared logic using anchors

All conversion is handled internally by the runtime.

---

:::important[Conversion Requirements]
To enable conversion, add the following to Orkestra on startup:
- ENABLE_CONVERSION=true     to create`/convert` endpoint
- TLS_CERT=/path/to/tls.crt
- TLS_KEY=/path/to/tls.key

Steps to generate the certificates can be found [here](./certificate-with-cert-manager.md).
:::

## Example: Website CRD (v1alpha1 → v1)

### What changed

* `v1alpha1` → has `theme`
* `v1` → replaces it with `seo`

---

### Katalog

```yaml
# Shared templates

# Common reconciler template for both versions
commonoperatorBox: &commonReconciler
  operatorBox:
    default: true
    onCreate:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        port: "{{ .spec.port }}"
        reconcile: true

# Common spec mapping used in conversion paths
commonSpec: &commonSpec
  image: "{{ .spec.image }}"
  replicas: "{{ .spec.replicas }}"
  port: "{{ .spec.port }}"

# Common Type
commonType: &commonType
  group: demo.orkestra.io
  kind: Website
  plural: websites

# Versions
storageVersion: &storageVersion v1
alpha: &alpha v1alpha1

# ── Katalog ─────────────────────────────────────────────────────────────
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: website-multiversion
spec:
  crds:
    # v1alpha1
    - name: website-v1alpha1
      <<: *commonReconciler
      apiTypes:
        <<: *commonType
        version: *alpha

    # v1 (storage version)
    # Declare conversion paths here
    - name: website-v1
      <<: *commonReconciler
      apiTypes:
        <<: *commonType
        version: *v1

      conversion:
        storageVersion: *v1
        paths:
          # Up: v1alpha1 → v1
          - from: *alpha
            to: *v1
            spec:
              <<: *commonSpec
              seo:
                enabled: false

          # Down: v1 → v1alpha1
          - from: *v1
            to: *alpha
            spec:
              <<: *commonSpec
              theme: "default"
```

---

## How It Works

### 1. Each version is independent

Each version gets:

* its own informer
* its own worker pool
* its own health + metrics

But they share the same runtime.

---

### 2. Conversion is just mapping

You define how fields translate:

* Add field → provide default
* Remove field → omit it
* Move field → template it

No functions. No handlers.

---

### 3. One storage version

Only one version is stored in etcd:

```yaml
conversion:
  storageVersion: v1
```

Everything else is converted automatically.

---

## Running It

```bash
kubectl apply -f crd.yaml
ork run --file katalog.yaml
```

That’s it.

---

## What You Get Automatically

* Conversion webhook (built-in)
* Metrics for every conversion
* Health endpoints per version
* Zero additional services

---

## Adding a New Version

To add `v2`:

1. Add a new CRD entry
2. Add conversion paths
3. Restart Orkestra

No code. No webhook. No redeploy.

---

## Key Takeaway

Multi-version CRDs used to be an infrastructure problem.

With Orkestra, they’re just YAML.

