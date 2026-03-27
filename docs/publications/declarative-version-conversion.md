# Declarative Version Conversion for Kubernetes CRDs

*Orkestra Project — March 2026*

---

## Abstract

Kubernetes CRDs support multiple simultaneous API versions, enabling APIs
to evolve without breaking existing clients. The standard implementation
requires writing conversion functions in a programming language, deploying
a separate webhook server, managing TLS certificates, and maintaining
conversion logic as versions accumulate. The operational overhead frequently
exceeds the benefit of versioning, leading teams to avoid it entirely.

This paper presents declarative version conversion — a model in which field
mappings between versions are expressed in YAML and evaluated at runtime by
the operator's template resolver. The conversion webhook is built into the
operator runtime, sharing its HTTPS server and requiring no separate
deployment. Conversion metrics are automatically exposed alongside
reconciliation metrics.

We demonstrate this approach in production: a live two-version CRD (v1alpha1
and v1) processing 62 conversions with zero failures at sub-millisecond
average latency, with no conversion code written and no webhook infrastructure
deployed. We argue that this approach is not a simplification of conversion
but a direct consequence of treating each CRD version as a first-class
operator with its own dedicated runtime stack.

---

## 1. Introduction

### 1.1 The Multi-Version Problem

Kubernetes CRDs can serve multiple API versions simultaneously. A cluster
running a CRD at `v1alpha1` can introduce `v1` without requiring clients to
migrate immediately. The API server stores all objects in the designated
storage version and converts them on demand when clients request other versions.

This model is powerful. In practice, it is rarely used. The reason is the
conversion webhook — the mechanism through which the API server requests
conversion from one version to another.

### 1.2 The Standard Implementation

A conversion webhook must implement the `ConversionReview` protocol: receive
a JSON object at one version, return it at another. The implementation
requires:

- Conversion functions written in Go for every (from, to) version pair
- A webhook server deployed as a separate binary
- TLS certificates valid for the webhook's service DNS name
- Certificate rotation before expiry
- A `ValidatingWebhookConfiguration` or CRD `conversion` block pointing to
  the server
- Monitoring for the webhook's availability and latency

The Kubebuilder multi-version tutorial walks through this process across
dozens of code blocks. For a change as simple as adding a field, teams
routinely spend a week or more on the webhook infrastructure before writing
any conversion logic.

### 1.3 The Consequence: Teams Avoid Versioning

The infrastructure tax produces a predictable response: teams avoid
introducing new versions. They keep `v1alpha1` in production indefinitely,
using it as if it were stable. They resist deprecating old fields because
the alternative — a new version with conversion — is too expensive. APIs
ossify at the alpha stage.

Some teams resort to separate clusters for different versions. This solves
the compatibility problem but introduces a cluster management problem that
is arguably worse.

---

## 2. The Architecture: Each Version as Its Own Operator

### 2.1 The Super-Operator Model

Orkestra treats each CRD as a complete, isolated operator with its own
informer, workqueue, worker pool, and reconciler. This is the super-operator
model: each CRD gets everything a traditional operator provides, hosted by
a shared runtime that contributes the orchestration infrastructure.

Multi-version CRDs extend this naturally: each version of a CRD is a
separate Katalog entry with its own operator stack.

```yaml
crds:
  - name: website-v1alpha1
    apiTypes:
      group: demo.orkestra.io
      version: v1alpha1
      kind: Website
    reconciler:
      default: true
      # ... v1alpha1 reconcile templates

  - name: website-v1
    apiTypes:
      group: demo.orkestra.io
      version: v1
      kind: Website
    reconciler:
      default: true
      # ... v1 reconcile templates
    conversion:
      storageVersion: v1
      paths: [...]
```

`website-v1alpha1` has its own informer watching `demo.orkestra.io/v1alpha1`.
`website-v1` has its own informer watching `demo.orkestra.io/v1`. They do
not share workers, queues, or reconcile logic. The version boundary is a
first-class operator boundary.

This is why declarative conversion is natural here: conversion rules sit
alongside reconcile templates because both are properties of the same CRD
version entry, evaluated by the same template resolver against the same
object representation.

### 2.2 The Template Resolver

Orkestra's template resolver evaluates Go `text/template` expressions against
the raw `map[string]interface{}` of a Kubernetes object. It is the same
resolver used for reconcile templates:

```yaml
- name: "{{ .metadata.name }}"
  image: "{{ .spec.image }}"
```

Conversion rules use the same syntax:

```yaml
spec:
  image: "{{ .spec.image }}"
  seo:
    enabled: false   # static default — not a template expression
```

Fields with no counterpart in the target version receive static defaults.
Fields with direct counterparts are templated through. Fields that exist
in the source but should not appear in the target are simply omitted from
the path spec.

---

## 3. Declarative Conversion Rules

### 3.1 Path Structure

Each conversion path declares an explicit `from` version, an explicit `to`
version, and the output spec in the target version's format:

```yaml
conversion:
  storageVersion: v1
  paths:
    - from: v1alpha1
      to: v1
      spec:
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        port: "{{ .spec.port }}"
        seo:
          enabled: false        # v1 adds this field; v1alpha1 has no value for it

    - from: v1
      to: v1alpha1
      spec:
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        port: "{{ .spec.port }}"
        theme: "default"        # v1alpha1 has this field; v1 does not
```

Both endpoints are declared explicitly. There is no inference about which
direction a path flows. The `from` and `to` fields are bare version strings
— the conversion handler extracts the version from the full apiVersion string
sent by Kubernetes.

### 3.2 Coverage of Common Cases

**Adding a field (up-conversion):** Provide a static default in the path
spec. Objects converted from an older version receive the default; objects
already at the new version retain their declared value.

**Removing a field (down-conversion):** Omit the field from the path spec.
It is not present in the output.

**Restructuring a field:** Use a template expression to extract the value
from its old location and a nested map to place it at its new location:

```yaml
# v1alpha1 had spec.schedule as a cron string
# v1 has spec.schedule.cron
spec:
  schedule:
    cron: "{{ .spec.schedule }}"
```

**Multiple versions:** Add a path for each (from, to) pair. No additional
code, no redeployment:

```yaml
paths:
  - from: v1alpha1
    to: v1
    spec: { ... }
  - from: v1beta1
    to: v1
    spec: { ... }
  - from: v1
    to: v1beta1
    spec: { ... }
```

---

## 4. The Conversion Webhook

### 4.1 Implementation

Orkestra's health server conditionally starts an HTTPS listener when
`ENABLE_CONVERSION=true`. The `/convert` endpoint implements the
`ConversionReview` protocol:

```
POST /convert
Content-Type: application/json

{
  "apiVersion": "apiextensions.k8s.io/v1",
  "kind": "ConversionReview",
  "request": {
    "uid": "...",
    "desiredAPIVersion": "demo.orkestra.io/v1alpha1",
    "objects": [{ ... }]
  }
}
```

The handler:
1. Decodes the `ConversionReview`
2. Extracts the bare source version from the object's `apiVersion` field
3. Extracts the bare target version from `desiredAPIVersion`
4. Looks up the conversion rules for the object's `kind`
5. Finds the `(from, to)` path
6. Resolves each field in the path spec using the template resolver
7. Returns the converted objects

If source and target version are the same, the object is returned unchanged.

### 4.2 TLS Configuration

The conversion webhook requires HTTPS with a certificate trusted by the
Kubernetes API server. Users provide certificate and key paths:

```
ENABLE_CONVERSION=true
TLS_CERT=/tls/tls.crt
TLS_KEY=/tls/tls.key
```

Certificate generation and renewal remain the user's responsibility. For
clusters with cert-manager, a `Certificate` resource pointed at the Orkestra
service is the standard approach. The TLS infrastructure is shared with any
other HTTPS serving the operator requires — it is not webhook-specific.

The CRD's conversion block points to Orkestra's service:

```yaml
conversion:
  strategy: Webhook
  webhook:
    clientConfig:
      service:
        name: orkestra
        namespace: orkestra-system
        path: /convert
        port: 8443
      caBundle: <base64-encoded-ca>   # from the TLS certificate
    conversionReviewVersions: ["v1"]  # versions the API server understands
```

---

## 5. Observability

### 5.1 Metrics

Conversion metrics are automatically exposed alongside reconciliation metrics:

```
orkestra_conversion_requests_total{kind, from_version, to_version, result}
orkestra_conversion_duration_seconds{kind, from_version, to_version}
orkestra_conversion_active_requests{kind}
```

These metrics answer questions that were previously impossible to answer
without custom instrumentation: how often is each conversion path invoked,
what is the conversion latency distribution, which conversions are failing.

### 5.2 Health API

The `/katalog/{crd}` endpoint includes conversion statistics:

```json
{
  "name": "website-v1",
  "conversion": {
    "enabled": true,
    "total": 62,
    "success": 62,
    "failures": 0,
    "avgLatencyMs": 0.5,
    "p95LatencyMs": 1.2
  }
}
```

---

## 6. Production Results

The following data is from a live deployment of the Website CRD with two
active versions: `demo.orkestra.io/v1alpha1` and `demo.orkestra.io/v1`.

### 6.1 Health API

```json
// /katalog/website-v1alpha1
{
  "name": "website-v1alpha1",
  "gvk": "demo.orkestra.io/v1alpha1, Kind=Website",
  "mode": "dynamic",
  "conversion": {
    "enabled": true,
    "total": 62,
    "success": 62,
    "failures": 0,
    "avgLatencyMs": 0,
    "p95LatencyMs": 0
  },
  "resourceCount": 2,
  "workers": 3,
  "workersActive": 0,
  "started": false
}

// /katalog/website-v1
{
  "name": "website-v1",
  "gvk": "demo.orkestra.io/v1, Kind=Website",
  "mode": "dynamic",
  "conversion": {
    "enabled": true,
    "total": 14,
    "success": 14,
    "failures": 0
  },
  "resourceCount": 2,
  "workers": 3,
  "workersActive": 3,
  "healthy": true,
  "started": true
}
```

`website-v1alpha1` shows `started: false` — its workers have not started
because it is not the storage version and has no pending reconciliation.
Its informer is running, its conversion metrics are accumulating, but its
worker pool is idle. This is the correct behavior: the runtime allocates
resources proportional to demand.

### 6.2 Metrics

```
# Conversion requests — all successful
orkestra_conversion_requests_total{kind="Website",from="v1alpha1",to="v1",result="success"} 14
orkestra_conversion_requests_total{kind="Website",from="v1",to="v1alpha1",result="success"} 17

# Conversion latency — sub-millisecond for the majority
orkestra_conversion_duration_seconds_bucket{from="v1alpha1",to="v1",le="0.001"} 13
orkestra_conversion_duration_seconds_bucket{from="v1",to="v1alpha1",le="0.001"} 14
orkestra_conversion_duration_seconds_sum{from="v1alpha1",to="v1"} 0.007
orkestra_conversion_duration_seconds_sum{from="v1",to="v1alpha1"} 0.019
```

### 6.3 Summary

62 total conversions. Zero failures. Average latency under one millisecond.
Zero lines of Go written. Zero additional deployments. Zero TLS certificates
managed beyond the shared HTTPS certificate.

The conversion rules in YAML total 20 lines. The equivalent Go implementation
— conversion functions, webhook server, TLS setup, CRD configuration — would
be several hundred.

---

## 7. Limitations

### 7.1 Conversion-Time Availability

The conversion webhook is in the path of API server requests. If the Orkestra
process is unavailable, conversions fail and clients cannot access objects
in non-storage versions. This is mitigated by leader election with multiple
replicas, warm caches, and Kubernetes' own retry logic for conversion errors.
It is a real concern for clusters requiring continuous availability of all
CRD versions.

### 7.2 Complex Transformations

Declarative conversion covers field mapping — adding, removing, renaming,
restructuring, and defaulting fields. It does not cover transformations
requiring external state, database lookups, or conditional logic that cannot
be expressed in Go templates. These cases still require imperative code.
Orkestra accommodates them through Go hooks that can perform conversion logic
and return the converted object.

### 7.3 Metadata and Status

The current implementation converts `spec` only. `metadata` and `status`
pass through unchanged. For most versioning scenarios this is sufficient —
schema changes typically affect spec. Metadata and status conversion, if
needed, requires Go hooks.

---

## 8. Conclusion

Multi-version CRDs have been a powerful but underutilised feature of
Kubernetes. The standard implementation imposes infrastructure overhead that
exceeds the benefit for all but the most resource-intensive teams.

Declarative version conversion, as implemented in Orkestra, demonstrates
that conversion is a field mapping problem, not an infrastructure problem.
When the operator runtime treats each CRD version as its own first-class
operator entry, conversion rules are a natural extension of reconcile
templates — evaluated by the same resolver, declared in the same Katalog,
observable through the same metrics endpoint.

Production results confirm the approach: zero conversion failures, sub-
millisecond latency, and no additional infrastructure beyond the shared
HTTPS certificate that the operator already requires.

Versioning a CRD should take minutes, not weeks. Declarative conversion
makes that possible.

---

*Orkestra — Declarative Operators for Kubernetes*
*March 2026 — https://github.com/iAlexeze/orkestra*