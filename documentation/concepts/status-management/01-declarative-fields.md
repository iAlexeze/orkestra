# Declarative Status Fields

Declare status fields in the Katalog. Values support the same Go template expressions as `onCreate` templates — resolved against the live CR at reconcile time.

```yaml
operatorBox:
  status:
    fields:
      - path: phase
        value: "Running"

      - path: observedReplicas
        value: "{{ .spec.replicas }}"

      - path: endpoint
        value: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"

      - path: version
        value: "{{ .spec.version }}"

      - path: database.host        # nested — becomes status.database.host
        value: "{{ .spec.host }}"

      - path: database.port
        value: "{{ .spec.port }}"
```

After a successful reconcile:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
  observedGeneration: 1
  phase: Running
  observedReplicas: "2"
  endpoint: my-site.orkestra.svc.cluster.local
  version: "1.25"
  database:
    host: db.platform.svc
    port: "5432"
```

---

## Rules

**Paths are relative to `status`.** `phase` writes to `status.phase`. `database.host` writes to `status.database.host`. Dot-notation works at any depth.

**Fields are only written on successful reconcile.** If the reconcile fails partway through, the declarative fields are not updated — only the `Ready` condition is written (as `False`). This prevents misleading status when cluster state is partial.

---

## CRD schema tip

!!! tip "Document your status fields in the CRD schema"
    Declare the status fields in the CRD's OpenAPIV3Schema to enable `kubectl` validation. Use `x-kubernetes-preserve-unknown-fields: true` to accept any field without enumerating every one:
    ```yaml
    status:
      type: object
      x-kubernetes-preserve-unknown-fields: true
    ```
