# 📘 **Katalog Schema**  
This schema describes everything a Katalog can contain.

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog

metadata:
  name: string
  description: string

spec:
  finalizers:
    - string

  endpoints:
    enabled: boolean            # default: true
    health: boolean             # default: true
    info: boolean               # default: true

  crds:
    - name: string
      enabled: boolean          # default: true
      namespaced: boolean       # default: true
      workers: integer          # default: 1
      resync: duration          # e.g. "30s", "5m"
      description: string

      dependsOn:
        - string                # list of CRD names

      restrictedNamespaces:
        - string                # namespaces to exclude from cleanup

      apiTypes:
        group: string           # e.g. demo.orkestra.io
        version: string         # e.g. v1alpha1
        kind: string            # e.g. Website
        plural: string          # e.g. websites
        location: string        # optional Go type path

      endpoints:
        enabled: boolean
        health: boolean
        info: boolean

      reconciler:
        default: boolean        # default: true
        finalizers:
          - string

        hooks:
          location: string      # Go package path
          function: string      # exported function name
          alias: string         # optional

        constructor:
          location: string      # Go package path
          function: string
          alias: string

        onCreate:
          deployments:
            - <<DeploymentTemplate>>
          services:
            - <<ServiceTemplate>>
          configMaps:
            - <<ConfigMapTemplate>>
          secrets:
            - <<SecretTemplate>>
          jobs:
            - <<JobTemplate>>

        onReconcile:
          deployments:
            - <<DeploymentTemplate>>
          services:
            - <<ServiceTemplate>>
          ...

        onDelete:
          deployments:
            - <<DeploymentTemplate>>
          services:
            - <<ServiceTemplate>>
          ...

      queue:
        default: boolean        # default: false
        maxQueueDepth: integer  # default: unlimited
```

---

## **Template Schema (Shared Across Resources)**

All resource templates share this structure:

```yaml
name: string
namespace: string
reconcile: boolean
labels:
  - key: string
    value: string
annotations:
  - key: string
    value: string

when:
  - field: string              # dot path, e.g. spec.enabled
    operator: string           # equals | notEquals | contains | prefix | suffix | exists | notExists | gt | lt
    value: string              # optional if operator is exists/notExists
    equals: string             # shorthand for operator: equals
```

---

## **DeploymentTemplate**

```yaml
name: string
image: string
replicas: string | int
port: string | int
namespace: string
reconcile: boolean
labels: [...]
annotations: [...]
when: [...]
```

---

## **ServiceTemplate**

```yaml
name: string
type: string
port: string | int
targetPort: string | int
namespace: string
reconcile: boolean
labels: [...]
annotations: [...]
when: [...]
```

---

## **ConfigMapTemplate**

```yaml
name: string
namespace: string
data:
  key: string
reconcile: boolean
labels: [...]
annotations: [...]
when: [...]
```

---

## **SecretTemplate**

```yaml
name: string
namespace: string
data:
  key: string
reconcile: boolean
labels: [...]
annotations: [...]
when: [...]
```

---

## **JobTemplate**

```yaml
name: string
namespace: string
image: string
command:
  - string
args:
  - string
reconcile: boolean
labels: [...]
annotations: [...]
when: [...]
```

---

