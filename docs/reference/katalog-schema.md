# **Katalog Schema**

The Katalog defines the full behavior of an Orkestra operator — CRDs, reconcilers, templates, hooks, constructors, conversion rules, conditions, endpoints, and lifecycle behavior.

Below is the complete schema.

```yaml
apiVersion: orkestra.konductor.io/v1Alpha     # Required
kind: Katalog   # Required

metadata:
  name: string    # Required
  description: string # (optional but encouraged)

spec:

  # ---------------------------------------------------------------
  # Global finalizers applied to all CRDs
  # ---------------------------------------------------------------
  finalizers:
    - string

  # ---------------------------------------------------------------
  # Operator endpoints (health, info, katalog)
  # ---------------------------------------------------------------
  endpoints:
    enabled: boolean              # default: true
    health: boolean               # default: true
    info: boolean                 # default: true

  # ---------------------------------------------------------------
  # CRD Definitions
  # ---------------------------------------------------------------
  crds:
    - name: string
      enabled: boolean            # default: true
      namespaced: boolean         # default: true
      workers: integer            # default: 1
      resync: duration            # e.g. "30s", "5m"
      description: string

      # -------------------------
      # Dependency ordering
      # -------------------------
      dependsOn:
        - string

      # -------------------------
      # Namespace restrictions
      # -------------------------
      restrictedNamespaces:
        - string

      # -------------------------
      # API type definition
      # -------------------------
      apiTypes:
        group: string
        version: string
        kind: string
        plural: string
        location: string          # optional Go type path

      # -------------------------
      # Per‑CRD endpoints
      # -------------------------
      endpoints:
        enabled: boolean
        health: boolean
        info: boolean

      # -----------------------------------------------------------
      # Version Conversion (Up/Down/Defaulting)
      # -----------------------------------------------------------
      conversion:
        from: string              # source version, e.g. v1alpha1
        to: string                # target version, e.g. v1beta1

        # Field mapping (old → new)
        mapping:
          sourceField: targetField
          # e.g. "spec.replicas": "spec.scale.replicas"

        # Default values for missing fields
        defaults:
          fieldPath: value
          # e.g. "spec.scale.min": "1"

        # Fields to drop when converting
        drop:
          - string

        # Optional type/structure transforms
        transforms:
          fieldPath:
            type: string          # object | string | int | bool
            fields:               # nested fields for objects
              fieldName: string

      # -----------------------------------------------------------
      # Reconciler Configuration
      # -----------------------------------------------------------
      reconciler:
        default: boolean          # default: true

        finalizers:
          - string                # CRD‑specific finalizers

        # -----------------------
        # Go hooks
        # -----------------------
        hooks:
          location: string        # Go module path
          function: string        # exported function name
          alias: string

        # -----------------------
        # Custom constructor
        # -----------------------
        constructor:
          location: string        # Go module path
          function: string        # exported constructor
          alias: string

        # -----------------------
        # Declarative templates
        # -----------------------
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
          configMaps:
            - <<ConfigMapTemplate>>
          secrets:
            - <<SecretTemplate>>
          jobs:
            - <<JobTemplate>>

        onDelete:
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

      # -----------------------------------------------------------
      # Queue Configuration
      # -----------------------------------------------------------
      queue:
        default: boolean          # default: false
        maxQueueDepth: integer    # default: unlimited
```

---

# **Template Schema (Shared Across All Resources)**

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

# -----------------------------------------------------------
# Conditional Provisioning
# -----------------------------------------------------------
when:
  - field: string                # dot path, e.g. spec.enabled
    operator: string             # equals | notEquals | contains | prefix | suffix | exists | notExists | gt | lt
    value: string                # optional for exists/notExists
    equals: string               # shorthand for operator: equals
```

---

# **DeploymentTemplate**

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

# **ServiceTemplate**

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

# **ConfigMapTemplate**

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

# **SecretTemplate**

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

# **JobTemplate**

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
