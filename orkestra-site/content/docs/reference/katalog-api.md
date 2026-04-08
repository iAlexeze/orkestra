---
title: "Katalog Api"
weight: 93
---

# Orkestra Runtime API Reference

The Orkestra operator exposes a small, stable HTTP API for inspecting CRDs,
their reconcilers, and their health status.

---

## `GET /katalog`

Returns a list of all CRDs currently managed by the operator.

### Response

```json
{
  "total": 3,
  "crds": [
    {
      "name": "website",
      "mode": "dynamic",
      "gvk": { ... },
      "workers": 2,
      "resync": "30s",
      "healthy": true,
      "uptime": "12m",
      "endpoints": {
        "health": "/katalog/website/health",
        "info": "/katalog/website"
      }
    }
  ]
}
```

---

## `GET /katalog/{crd}`

Returns detailed configuration and runtime state for a single CRD.

### Response fields

- **mode** — dynamic or typed  
- **workers / resync / queueDepth** — resolved values with source  
- **reconciler** — hooks, finalizers, constructor  
- **resourceCount** — number of objects in informer cache  
- **healthy / errorRate** — live health metrics  

---

## `GET /katalog/{crd}/health`

Returns the health status of the CRD reconciler.

### Healthy example

```json
{
  "name": "website",
  "healthy": true,
  "started": true,
  "uptime": "5m",
  "errorRate": 0.0,
  "lastReconcile": "2026-03-20T04:00:00Z"
}
```

### Degraded example

```json
{
  "name": "website",
  "healthy": false,
  "message": "website degraded",
  "consecutiveFails": 5,
  "errorRate": 0.83,
  "status": 503
}
```