# Declarative Version Conversion for Kubernetes CRDs

*Orkestra Project — March 2026*

---

## Abstract

Kubernetes CRDs support multiple API versions, allowing APIs to evolve without breaking existing clients. However, implementing multi‑version support has remained complex. The standard approach requires writing a conversion webhook in Go, deploying it as a separate service, managing TLS certificates, and maintaining conversion logic across versions. This infrastructure overhead often outweighs the benefit of adding a new version, leading teams to avoid versioning or create separate clusters to isolate incompatible versions.

This paper introduces a declarative alternative. Rather than writing imperative conversion code, users declare field mappings between versions in a YAML file. The conversion webhook is built into the operator runtime, eliminating the need for a separate deployment. The webhook is served over HTTPS. Conversion metrics are automatically exposed, providing visibility into conversion volume, latency, and errors. We demonstrate that declarative version conversion reduces the complexity of multi‑version CRDs from days of development to minutes of configuration.

---

## 1. Introduction

Kubernetes CRDs allow API evolution through multiple versions. A single CRD can serve v1alpha1, v1beta1, and v1 simultaneously. Clients can request any version, and the API server converts between them using a conversion webhook.

This model is powerful but complex. The standard approach, as documented in the Kubebuilder multi‑version tutorial, requires:

- Writing Go conversion functions for every version pair
- Building and deploying a webhook server
- Managing TLS certificates for HTTPS
- Configuring the CRD to point to the webhook
- Maintaining conversion logic across versions

For a simple change—like adding a field or changing a field from a string to a structured type—the infrastructure overhead often exceeds the development cost. Teams frequently avoid versioning entirely or resort to creating separate clusters for different versions.

This paper presents an alternative: declarative version conversion. Users declare how fields map between versions in a YAML file. The operator runtime includes a built‑in conversion webhook that applies these rules. No Go code. No separate deployment. No TLS management. Conversion metrics are automatically exposed.

---

## 2. The Standard Approach: Webhooks in Go

### 2.1 The Conversion Webhook Contract

Kubernetes expects a conversion webhook to implement the `ConversionReview` API. When a client requests a version different from the storage version, the API server sends a `ConversionReview` request to the webhook. The webhook must return the converted objects.

The webhook must be served over HTTPS with a valid certificate trusted by the API server. It must be deployed as a separate service with a stable endpoint that the CRD's `conversion` block can reference.

### 2.2 The Kubebuilder Multi‑Version Tutorial

The Kubebuilder tutorial for multi‑version CRDs shows the complexity. For a CronJob CRD that changes from a string schedule to a structured schedule, the user must:

1. Write conversion functions in Go for v1 → v2 and v2 → v1
2. Write validation and defaulting webhooks for each version
3. Configure the webhook server with TLS certificates
4. Update the CRD's `conversion` block
5. Deploy the webhook server alongside the controller

This is not a minor task. The tutorial itself spans dozens of code blocks and hundreds of lines of Go.

### 2.3 The Infrastructure Tax

Beyond the code, the webhook requires:

- A separate deployment (or a second container in the controller pod)
- TLS certificate generation and rotation
- A service to route traffic to the webhook
- Proper RBAC for the webhook to access the API server (if it needs to)
- Monitoring and alerting for the webhook's health

For teams already struggling with operator sprawl, adding a webhook for version conversion is often the last straw. They either keep the old version forever or avoid versioning entirely.

---

## 3. Declarative Conversion

### 3.1 The Insight

The conversion webhook's job is simple: given an object in one version, produce the same object in another version. This is a mapping problem, not a programming problem. The mapping can be expressed declaratively.

### 3.2 Declarative Rules

In Orkestra's Katalog, conversion rules are declared per kind:

```yaml
conversion:
  - kind: Website
    storageVersion: v1
    paths:
      - from: v1alpha1
        to: v1
        spec:
          image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
          autoscaling:
            enabled: false
      - from: v1
        to: v1alpha1
        spec:
          image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
```

Each path specifies:
- The source version (`from`)
- The target version (`to`)
- A template that constructs the target object from the source object

The template language is the same Go text/template used for reconciliation. Fields that exist in the target but not in the source receive default values (like `autoscaling.enabled: false`). Fields that exist in the source but not in the target are dropped.

### 3.3 The Built‑in Webhook

Orkestra's health server already listens on a port. When conversion is enabled, it adds a `/convert` endpoint that implements the `ConversionReview` API. The CRD's `conversion` block points to this endpoint:

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
```

The webhook is served over HTTPS. This requires users to provide their own certificates.

### 3.4 Automatic Observability

Because the conversion webhook is part of the operator runtime, metrics are automatically exposed:

```
orkestra_conversion_requests_total{kind="Website", from_version="v1alpha1", to_version="v1", result="success"} 47
orkestra_conversion_requests_total{kind="Website", from_version="v1", to_version="v1alpha1", result="success"} 12
orkestra_conversion_duration_seconds_bucket{kind="Website", from_version="v1alpha1", to_version="v1", le="0.001"} 45
```

The `/katalog/{crd}` endpoint also shows conversion statistics:

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

This is the first time version conversion has been made observable.

---

## 4. Implementation

### 4.1 The ConversionReview Handler

Orkestra's health server includes a handler for the `/convert` endpoint:

```go
func (h *HealthServer) conversionHandler(w http.ResponseWriter, r *http.Request) {
    var review ConversionReview
    json.NewDecoder(r.Body).Decode(&review)

    resp := &ConversionReviewResponse{
        UID: review.Request.UID,
        ConvertedObjects: make([]json.RawMessage, len(review.Request.Objects)),
        Result: &Status{Status: "Success"},
    }

    for i, raw := range review.Request.Objects {
        var obj map[string]interface{}
        json.Unmarshal(raw, &obj)

        kind := obj["kind"].(string)
        rules := h.katalog.GetConversionRules(kind)

        converted, _ := applyConversion(obj, rules, review.Request.DesiredAPIVersion)

        out, _ := json.Marshal(converted)
        resp.ConvertedObjects[i] = out
    }

    json.NewEncoder(w).Encode(ConversionReview{Response: resp})
}
```

### 4.2 Applying Conversion Rules

The conversion logic uses the same template resolver as reconciliation:

```go
func applyConversion(obj map[string]interface{}, rules *ConversionRules, targetVersion string) map[string]interface{} {
    sourceVersion := obj["apiVersion"].(string)

    path := rules.FindPath(sourceVersion, targetVersion)
    if path == nil {
        return nil
    }

    resolver := orktmpl.NewResolverFromMap(obj)

    result := copyMap(obj)
    result["apiVersion"] = targetVersion

    if convertedSpec, err := resolver.ResolveMap(path.Spec); err == nil {
        result["spec"] = convertedSpec
    }

    return result
}
```

### 4.3 TLS Configuration

Users can enable the conversion webhook with the following ENV vars:

```bash
ENABLE_CONVERSION=          # Default: false
TLS_CERT=                   # Required if conversion is enabled
TLS_KEY=                    # Required if conversion is enabled
CONVERSION_WINDOW=          # Default: 1000
```

If certificates are not provided, Orkestra logs a warning and the conversion endpoint is not served. In development, users can use self‑signed certificates or disable TLS.

---

## 5. Comparison to the Standard Approach

| Aspect | Standard Approach | Declarative Approach |
|--------|-------------------|----------------------|
| **Code** | Go conversion functions | YAML templates |
| **Webhook deployment** | Separate service | Built‑in to operator |
| **TLS** | Managed separately | Shared with health server |
| **Observability** | Manual instrumentation | Automatic metrics |
| **Version addition** | Write new conversion functions | Add new path to Katalog |
| **Infrastructure** | Webhook server, service, certificates | None |

---

## 6. Use Cases

### 6.1 Adding a New Field

When a new version adds a field that doesn't exist in the old version, the conversion rule provides a default:

```yaml
- from: v1alpha1
  to: v1
  spec:
    image: "{{ .spec.image }}"
    replicas: "{{ .spec.replicas }}"
    autoscaling:
      enabled: false   # default for old resources
```

### 6.2 Removing a Field

When a new version removes a field, the conversion rule drops it when converting down:

```yaml
- from: v1
  to: v1alpha1
  spec:
    image: "{{ .spec.image }}"
    replicas: "{{ .spec.replicas }}"
    # autoscaling dropped — it didn't exist in v1alpha1
```

### 6.3 Restructuring Fields

When a field changes from a string to a structured type, the conversion rule builds the new structure:

```yaml
- from: v1alpha1
  to: v1
  spec:
    schedule:
      minute: "{{ .spec.schedule }}"
      hour: "*"
      dayOfMonth: "*"
      month: "*"
      dayOfWeek: "*"
```

### 6.4 Multiple Versions

With declarative rules, adding a third version is a matter of adding a new path:

```yaml
paths:
  - from: v1alpha1
    to: v1
    # ...
  - from: v1beta1
    to: v1
    # ...
  - from: v1
    to: v1beta1
    # ...
```

No new conversion functions, no new webhook logic, no redeployment.

---

## 7. Limitations

Declarative conversion does not replace imperative conversion for all use cases. Complex transformations that require external state or conditional logic still need imperative code. However, the common case—adding, removing, and restructuring fields—is well‑covered by declarative rules.

The conversion webhook still runs in the operator process. For clusters with extremely high conversion volume, this may impact reconciliation performance. In practice, conversion is infrequent compared to reconciliation, and the impact is minimal.

---

## 8. Conclusion

Multi‑version CRDs have been a powerful but complex feature of Kubernetes. The standard approach of writing conversion webhooks in Go imposes a significant infrastructure tax that often outweighs the benefit of adding a new version.

Declarative version conversion eliminates this complexity. Users declare field mappings between versions in a YAML file. The operator runtime includes a built‑in conversion webhook that applies these rules. The webhook is served over HTTPS and conversion metrics are automatically exposed.

This approach reduces the complexity of multi‑version CRDs from days of development to minutes of configuration. It makes versioning accessible to teams that previously avoided it, and it provides visibility into conversion activity that was previously hidden.

Declarative conversion is the next step in making Kubernetes operators truly declarative: from reconciliation to versioning, the user's intent is expressed in YAML, not code.