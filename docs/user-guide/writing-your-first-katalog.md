# Writing Your First Katalog

Now that you have Orkestra installed and have run the example operator, let's build your own Katalog from scratch.

A **Katalog** is a YAML file that declares:
- What CRDs you want to manage
- What resources to create for each CR
- How to reconcile them

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
    - name: myapp
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
- Use the default reconciler (no custom code needed)
- Do nothing else (no resources created)

---

## Adding Resources

Let's create a Deployment when a `MyApp` CR is applied:

```yaml
spec:
  crds:
    - name: myapp
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

### Understanding Templates

Values inside `{{ }}` are Go templates evaluated against the live CR:

| Template | Resolves To |
|----------|-------------|
| `{{ .metadata.name }}` | The CR's name |
| `{{ .spec.image }}` | The value of `spec.image` in the CR |
| `{{ .spec.replicas }}` | The value of `spec.replicas` |
| `{{ .metadata.namespace }}` | The CR's namespace |

Static values are used as-is.

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

---

## Adding a Secret

Secrets can be created from static data or copied from existing secrets:

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

The `reconcile: true` flag ensures the secret stays in sync with its source.

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

The service is only created when `spec.exposePublicly: true` in the CR.

---

## Dependencies Between CRDs

If your operator manages multiple CRDs, you can declare dependencies:

```yaml
crds:
  - name: database
    # ... config ...
  - name: application
    dependsOn:
      - database
    # ... config ...
```

Orkestra ensures `database` starts before `application`.

---

## Complete Example

Here's a complete Katalog for a simple web application:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: webapp-katalog
spec:
  crds:
    - name: webapp
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

You now know how to write a Katalog that creates resources from CRs. Next, learn how to add **Go hooks** for custom logic when templates aren't enough.



---
