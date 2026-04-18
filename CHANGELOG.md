# **Changelog**

## **Added — forEach Map Capability**
Orkestra now supports **map‑based iteration** in `forEach` blocks, allowing CR authors to express **per‑key configuration** instead of being limited to list iteration.

`spec.regions` may now be either:

### **1. A list (uniform configuration)**  
```yaml
spec:
  regions:
    - us-east-1
    - eu-west-1
    - ap-southeast-1
```

### **2. A map (per‑region configuration)**  
```yaml
spec:
  regions:
    us-east-1:
      replicas: 3
      port: 8080
    eu-west-1:
      replicas: 1
```

Inside the `forEach` block:

- `.item` → region name  
- `.value` → region‑specific config map  

This enables patterns such as:

```yaml
replicas: "{{ or .value.replicas .spec.defaultReplicas }}"
port:     "{{ default .value.port .spec.defaultPort }}"
```

This enhancement unlocks **per‑region overrides**, **heterogeneous fan‑out**, and **fine‑grained multi‑resource expansion** without writing Go code.

---

## **Improved — Service Generation in forEach Blocks**
Fixed an issue where Services generated inside a `forEach` block did not correctly apply labels/selectors when iterating over multiple regions.

### **Fixes include:**

- Correct propagation of `.item` into `labels:` and `selector:`  
- Ensuring Services select only the Pods for their corresponding region  
- Eliminating cross‑region collisions in multi‑Deployment fan‑out  
- Ensuring stable naming and deterministic reconciliation  

Example of the corrected pattern:

```yaml
services:
  - name: "{{ .metadata.name }}-{{ .item }}-svc"
    selector:
      app: "{{ .metadata.name }}-{{ .item }}"
    forEach:
      field: spec.regions
      as: item
```

This ensures each Service is tightly bound to its region‑specific Deployment.

---

## **Enhanced — Multi‑Region Demo Katalog**
The multi‑region demo now showcases:

- **map‑aware forEach**  
- **per‑region replicas and ports**  
- **region‑scoped Deployments and Services**  
- **cleaner label/selector wiring**  
- **default fallbacks via `or` and `default` notes**  

This makes the example a canonical reference for advanced declarative fan‑out patterns.
