---
title: "Writing Your First Katalog"
weight: 22
---

# Writing Your First Katalog

Now that you have Orkestra installed and have run the example operator, you’re ready to build your own Katalog from scratch.

A **Katalog** is a declarative file that tells Orkestra:

- Which CRDs you want to manage  
- What resources to create for each CR  
- How reconciliation should behave  

{{< callout type="note" >}}
This guide assumes you have already completed the installation steps and successfully run the example operator.
{{< /callout >}}

---

## The Simplest Katalog

Create a file called `my-katalog.yaml`:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: my-first-katalog
spec:
  crds:
    myapp:
      apiTypes:
        group: demo.myorg.io
        version: v1alpha1
        kind: MyApp
        plural: myapps
      reconciler:
        default: true
```

This Katalog tells Orkestra:

- Watch for `MyApp` resources in the cluster  
- Use the default reconciler  
- Do not create any resources yet  

{{< callout type="tip" >}}
Every CRD entry must define `apiTypes` and a `reconciler`.  
Everything else is optional and added only when needed.
{{< /callout >}}

---

## Adding Resources

Let’s create a Deployment whenever a `MyApp` CR is applied:

```yaml
spec:
  crds:
    myapp:
      apiTypes:
        group: demo.myorg.io
        version: v1alpha1
        kind: MyApp
        plural: myapps
      reconciler:
        default: true
        onCreate:
          deployments:
            - name: "{{ .metadata.name }}-app"
              image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              port: "{{ .spec.port }}"
              namespace: "{{ .metadata.namespace }}"
              reconcile: true
```

{{< callout type="note" >}}
`reconcile: true` means the Deployment will be drift‑corrected on every reconcile, not just created once.
{{< /callout >}}

---

## Understanding Templates

Values inside `{{ }}` are Go templates evaluated against the live CR.

| Template | Resolves To |
|----------|-------------|
| `{{ .metadata.name }}` | The CR's name |
| `{{ .spec.image }}` | The value of `spec.image` |
| `{{ .spec.replicas }}` | The value of `spec.replicas` |
| `{{ .metadata.namespace }}` | The CR's namespace |

{{< callout type="tip" >}}
Templates always evaluate against the **current** version of the CR, so updates to the CR automatically propagate to generated resources.
{{< /callout >}}

---

## Adding a Service

```yaml
onCreate:
  deployments:
    # ... deployment config ...
  services:
    - name: "{{ .metadata.name }}-svc"
      type: "{{ .spec.serviceType }}"
      port: "80"
      targetPort: "{{ .spec.port }}"
      namespace: "{{ .metadata.namespace }}"
      reconcile: true
```

{{< callout type="note" >}}
Services support drift correction as well. If the CR changes, Orkestra updates the Service automatically.
{{< /callout >}}

---

## Adding a Secret

Secrets can be created from static values or copied from existing secrets:

```yaml
onCreate:
  secrets:
    # Static secret
    - name: "{{ .metadata.name }}-creds"
      data:
        USERNAME: admin
        PASSWORD: "{{ .spec.password }}"

    # Copy from existing secret
    - name: db-creds
      fromSecret: master-db-creds
      fromNamespace: platform
      toNamespaces:
        - "{{ .metadata.namespace }}"
      reconcile: true
```

{{< callout type="caution" >}}
When copying secrets, ensure the source secret exists before the CR is reconciled.  
Otherwise reconciliation will fail until the secret becomes available.
{{< /callout >}}

---

## Adding a ConfigMap

```yaml
onCreate:
  configMaps:
    - name: "{{ .metadata.name }}-config"
      data:
        LOG_LEVEL: "{{ .spec.logLevel }}"
        MAX_CONNECTIONS: "{{ .spec.maxConnections }}"
      reconcile: true
```

---

## Conditional Resources

Sometimes you only want to create a resource under certain conditions. Use the `when` block:

```yaml
services:
  - name: "{{ .metadata.name }}-public-svc"
    type: LoadBalancer
    port: "80"
    targetPort: "{{ .spec.port }}"
    when:
      - field: spec.exposePublicly
        equals: "true"
```

{{< callout type="tip" >}}
Conditions allow you to express branching logic declaratively without writing Go hooks.
{{< /callout >}}

---

## Dependencies Between CRDs

If your operator manages multiple CRDs, you can declare dependencies:

```yaml
crds:
  database:
    # ... config ...
  application:
    dependsOn:
      - database
    # ... config ...
```

Orkestra ensures:

- `database` reconciles first  
- `application` waits until `database` is healthy  

{{< callout type="note" >}}
Dependencies apply to reconciliation order, not CR creation order.
{{< /callout >}}

---

## Complete Example

Here is a complete Katalog for a simple web application:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: webapp-katalog
spec:
  crds:
    webapp:
      apiTypes:
        group: demo.myorg.io
        version: v1alpha1
        kind: WebApp
        plural: webapps
      reconciler:
        default: true
        onCreate:
          secrets:
            - name: "{{ .metadata.name }}-creds"
              data:
                API_KEY: "{{ .spec.apiKey }}"
          configMaps:
            - name: "{{ .metadata.name }}-config"
              data:
                LOG_LEVEL: "{{ .spec.logLevel }}"
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              port: "{{ .spec.port }}"
              reconcile: true
          services:
            - name: "{{ .metadata.name }}-svc"
              type: "{{ .spec.serviceType }}"
              port: "80"
              targetPort: "{{ .spec.port }}"
              reconcile: true
              when:
                - field: spec.exposePublicly
                  equals: "true"
```

---

## Next Steps

You now know how to write a Katalog that creates resources from CRs.

Continue with:

**Writing Your First Komposer**  
Learn how to load katalogs from files, Helm charts, and registries — and how to apply overrides.

