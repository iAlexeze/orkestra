# ONCOP — Orkestra Native Cross‑Operator Protocol
### *Specification v1.0 — April 2026*  
### *Status: Draft Standard*

---

## **1. Introduction**

The Orkestra Native Cross‑Operator Protocol (ONCOP) defines a standard mechanism for one operator (the *observer*) to retrieve structured state from another operator (the *subject*) across process boundaries. ONCOP provides:

- a typed observation model  
- deterministic URL inference  
- consistent JSON response shapes  
- caching semantics  
- fallback behavior  
- compatibility with non‑Orkestra operators  

ONCOP is transport‑agnostic but defined over HTTP/1.1 and HTTP/2 for this version.

This document specifies the protocol, URL structure, request/response semantics, error handling, and integration rules.

---

## **2. Terminology**

- **Observer** — the operator performing a cross‑operator read.  
- **Subject** — the operator exposing ONCOP endpoints.  
- **CRD** — Kubernetes CustomResourceDefinition managed by the subject.  
- **CR** — a specific instance of a CRD.  
- **Cross Declaration** — a Katalog `cross:` entry requesting ONCOP data.  
- **Source** — the `source:` block specifying host, type, and caching.  
- **Type** — the ONCOP observation surface (`metrics`, `health`, `cr`, `info`, `events`).  
- **Namespace** — Kubernetes namespace of the CR.  
- **Name** — name of the CR.  

---

## **3. Protocol Overview**

ONCOP defines five observation surfaces:

| Type | Description |
|------|-------------|
| **metrics** | Operator‑level metrics for the CRD |
| **health** | Operator health and last error |
| **cr** | CR‑specific detail: status, spec, children, metrics |
| **info** | CRD‑level info: list, metrics, children |
| **events** | CR‑scoped event stream |

Each type maps to a deterministic URL shape under the subject operator’s `/katalog/` namespace.

The observer constructs the URL using:

- `source.host`  
- `source.type`  
- `decl.crd`  
- `decl.selector.name`  
- `decl.selector.namespace`  

The observer then performs an HTTP GET and injects the resulting JSON into `.cross.<as>`.

---

## **4. URL Construction**

### **4.1 Base URL**

```
<host>/katalog/<crd>
```

Where:

- `<host>` is `source.host` without trailing slash  
- `<crd>` is the CRD name from the cross declaration  

### **4.2 URL Shapes by Type**

#### **4.2.1 metrics**

```
GET <host>/katalog/<crd>
```

Returns operator‑level metrics for the CRD.

#### **4.2.2 health**

```
GET <host>/katalog/<crd>/health
```

Returns operator health state.

#### **4.2.3 cr**

```
GET <host>/katalog/<crd>/cr/<namespace>/<name>
```

Returns CR‑specific detail.

#### **4.2.4 info**

```
GET <host>/katalog/<crd>
```

Returns CRD‑level info (same endpoint as metrics; response includes both).

#### **4.2.5 events**

```
GET <host>/katalog/<crd>/cr/<namespace>/<name>/events
```

Returns CR‑scoped events.

---

## **5. Request Semantics**

### **5.1 HTTP Method**

All ONCOP requests use:

```
GET
```

No request body is permitted.

### **5.2 Headers**

The observer MAY send:

```
Authorization: Bearer <token>
Accept: application/json
User-Agent: Orkestra/<version>
```

### **5.3 Caching**

If `source.cacheFor` is specified, the observer MUST cache the response for the specified duration.

Cache keys are:

```
<type>:<crd>:<namespace>:<name>:<host>
```

---

## **6. Response Semantics**

### **6.1 Content Type**

```
Content-Type: application/json
```

### **6.2 Response Shapes**

#### **6.2.1 metrics**

```json
{
  "metrics": {
    "<metricName>": <value>,
    ...
  }
}
```

#### **6.2.2 health**

```json
{
  "state": "healthy" | "degraded" | "error",
  "lastError": "<string>"
}
```

#### **6.2.3 cr**

```json
{
  "status": { ... },
  "spec": { ... },
  "children": {
    "<kind>": {
      "<name>": { ... }
    }
  },
  "metrics": { ... }
}
```

#### **6.2.4 info**

```json
{
  "crd": "<crd>",
  "metrics": { ... },
  "children": { ... }
}
```

#### **6.2.5 events**

```json
{
  "events": [
    {
      "timestamp": "<RFC3339>",
      "type": "<Normal|Warning>",
      "reason": "<string>",
      "message": "<string>"
    }
  ]
}
```

---

## **7. Error Handling**

### **7.1 HTTP Errors**

| Code | Meaning |
|------|---------|
| 404 | CR or CRD not found |
| 401 | Unauthorized |
| 403 | Forbidden |
| 500 | Operator internal error |
| 503 | Operator unavailable |

### **7.2 Observer Behavior**

If ONCOP returns an error:

1. Observer MUST log the error  
2. Observer MUST NOT retry immediately  
3. Observer MUST fall back to:  
   - raw endpoint (if provided)  
   - empty result  

Empty result shape:

```json
{
  "found": "false",
  "status": {},
  "spec": {},
  "children": {}
}
```

---

## **8. Integration with Katalog**

A cross declaration:

```yaml
cross:
  - crd: loader
    selector:
      name: my-loader
      namespace: default
    source:
      host: "http://loader:8080"
      type: cr
      cacheFor: 10s
    as: loaderCRInfo
```

MUST result in:

```
.cross.loaderCRInfo → JSON response
```

The resolver MUST expose:

- `.cross.<as>.status.*`
- `.cross.<as>.spec.*`
- `.cross.<as>.children.*`
- `.cross.<as>.metrics.*`

Autoscale conditions MUST be able to reference:

```
cross.<crd>.metrics.<metricName>
```

---

## **9. Security Considerations**

- Operators SHOULD require authentication for ONCOP endpoints.  
- Tokens SHOULD be short‑lived.  
- Operators MUST NOT expose sensitive data in ONCOP responses.  
- Cross‑cluster ONCOP SHOULD use TLS.  

---

## **10. Versioning**

ONCOP is versioned independently of Orkestra.

This document defines:

```
ONCOP/1.0
```

Future versions MAY add:

- streaming endpoints  
- PATCH‑based deltas  
- typed schemas  
- CRD introspection  

Backward compatibility MUST be preserved for all URL shapes.

---

## **11. IANA Considerations**

ONCOP registers the following media type:

```
application/vnd.orkestra.oncop+json
```

Used for all ONCOP responses.

---

## **12. Conclusion**

ONCOP provides a stable, typed, declarative protocol for cross‑operator observation in Orkestra. It enables autoscaling, status propagation, dependency tracking, and multi‑operator composition without coupling or hard‑coded URLs.

It is the observation substrate of the Orkestra platform.
