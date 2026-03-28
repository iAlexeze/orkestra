# Registry

The Orkestra Registry is the engine behind all declarative resource operations.  
It provides a consistent, production‑grade implementation for creating, updating, deleting, and copying Kubernetes resources — all driven by Katalog templates or Go hooks.

The Registry is what makes Orkestra operators **safe**, **idempotent**, and **predictable**.

---

## What the Registry Does

The Registry provides implementations for:

- Deployments  
- StatefulSets  
- DaemonSets  
- Services  
- ConfigMaps  
- Secrets  
- Namespace provisioning  
- Multi‑namespace secret distribution  
- Drift correction  
- Finalizers  
- Status helpers  

!!! note
    Templates and hooks do not talk to the Kubernetes API directly.  
    They call the Registry, which handles idempotency, patching, ownership, and drift correction.

---

## Why the Registry Exists

Kubernetes controllers must handle:

- Create vs update logic  
- Owner references  
- Drift correction  
- Finalizers  
- Status writes  
- Error handling  
- Idempotency  
- Race conditions  
- Ordering  
- Event recording  

The Registry centralizes all of this so operator authors don’t have to.

---

## Example: Creating a Deployment

```yaml
deployments:
  - name: "{{ .metadata.name }}"
    namespace: "{{ .metadata.namespace }}"
    image: "{{ .spec.image }}"
    replicas: "{{ .spec.replicas }}"
    reconcile: true
```

This becomes a call to:

```
registry.Deployments.CreateOrUpdate(...)
```

The Registry ensures:

- The Deployment exists  
- It matches the template  
- Drift is corrected  
- Owner references are set  
- No duplicate resources are created  

---

## Example: Copying Secrets

```yaml
secrets:
  - name: db-creds
    fromSecret: master-db-creds
    fromNamespace: platform
    toNamespaces:
      - staging
      - production
    reconcile: true
```

This uses:

```
registry.Secrets.CopyToNamespaces(...)
```

---

## Registry Guarantees

- **Idempotent** — running the same reconcile twice produces the same result  
- **Safe** — no duplicate resources, no partial updates  
- **Declarative** — templates describe the desired state  
- **Consistent** — all CRDs use the same underlying logic  
- **Observable** — all operations emit metrics and events  

---

## When to Use the Registry Directly

Use the Registry directly inside Go hooks when:

- You need external API calls + Kubernetes resources  
- You need to compute status fields  
- You need custom logic before or after resource creation  
- You want full type‑safe access to the CR  

---
## Related Documentation
- [Orkestra Registry](../../orkestra-registry/index.md)
- [Registry Schema](../../reference/registry-schema.md)