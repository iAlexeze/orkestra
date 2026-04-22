---
title: "The Katalog Manifest"
weight: 1
description: "Understanding the core Katalog YAML specification."
---

The `Katalog` manifest is the core configuration file that defines how Orkestra should reconcile one or more CRDs.  
Each CRD entry describes:

- the API type  
- worker settings  
- lifecycle hooks  
- generated resources (deployments, services, secrets, certs, configmaps, jobs, etc.)  
- dependency ordering  
- resync intervals  
- finalizers  

Everything is declarative.

---

## Katalog Structure

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: <katalog-name>
  version: <version>
  author: <author>
  description: <description>

spec:
  finalizers: []        # optional
  providers: []         # future use

  crds:                 # map of CRD entries
    <crdName>:
      apiTypes:
      operatorBox:
      dependsOn:
      workers:
      resync:
      enabled:
      finalizers:
      ...
```

---

# CRD Entry Structure

Each CRD is defined under `spec.crds` as a **map**, not a list.

Example:

```yaml
spec:
  crds:
    website:
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites
```

Orkestra uses this to:

- register the CRD  
- generate RBAC  
- create the reconciler  
- expose it in Control Center  

---

# Reconciler Section

This is where Orkestra defines what to create when a CR instance appears.

```yaml
operatorBox:
  onCreate:
    deployments:
      - image: "{{ .spec.image }}"
    services:
      - port: 80
    secrets:
      - name: "{{ .metadata.name }}-creds"
        once: true
        rotateAfter: 30s
        data:
          password: "{{ randomAlphanumeric 32 }}"
    certificates:
      - name: "{{ .metadata.name }}-tls"
        rotateAfter: 24h
```

Supported generators include:

- `deployments`
- `services`
- `configmaps`
- `secrets`
- `certificates`
- `jobs`
- `serviceAccounts`
- `roles`
- `roleBindings`
- and more

All templated with Go templates.

---

# Renewable Secrets & Certificates

Your new feature:

```yaml
secrets:
  - name: "{{ .metadata.name }}-creds"
    once: true
    rotateAfter: 30s
    data:
      password: "{{ randomAlphanumeric 32 }}"
```

Same applies to certificates:

```yaml
certificates:
  - name: "{{ .metadata.name }}-tls"
    rotateAfter: 24h
```

Orkestra automatically:

- tracks rotation timestamps  
- regenerates values  
- updates child resources  
- updates status conditions  

---

# dependsOn (3 formats)

### 1. List format (defaults to `started`)

```yaml
dependsOn:
  - database
```

### 2. Simple map

```yaml
dependsOn:
  database: healthy
```

### 3. Structured map

```yaml
dependsOn:
  database:
    condition: healthy
```

---

# Workers & Resync

```yaml
workers: 5
resync: 30s
```

---

# Finalizers

```yaml
finalizers:
  - cleanup-child-resources
```

---

# Observability

Everything is automatically exposed in:

- Control Center  
- Metrics
- Endpoints  
- Events  

No extra config needed.

---

## Next steps

- [Resource Templates](/docs/basics/resource-templates/) — advanced templating patterns
- [Status Management](/docs/basics/status/) — custom status fields and conditions
- [Lifecycle Hooks](/docs/basics/hooks/) — pre/post reconciliation hooks
