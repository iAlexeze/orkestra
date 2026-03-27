# **Komposer Schema**

The Komposer merges multiple Katalog sources — files, URLs, Helm charts, and Registry entries — into a single, unified operator configuration.

It is the declarative way to:

- compose multiple teams’ Katalogs  
- apply environment‑specific overrides  
- select CRD versions  
- override workers, resync intervals, and reconciler behavior  
- pull Katalogs from Git, Helm, or the Orkestra Registry  

Below is the full schema.

---

```yaml
apiVersion: orkestra.konductor.io/v1Alpha     # Required
kind: Komposer   # Required

metadata:
  name: string    # Required
  description: string # (optional but encouraged)

# -------------------------------------------------------------------
# Sources: where Katalogs come from
# -------------------------------------------------------------------
sources:

  # -----------------------------
  # File & URL sources
  # -----------------------------
  files:
    - string |                     # simple path or URL
      url: string                  # explicit URL
      path: string                 # local file path
      auth:                        # optional authentication
        type: string               # bearer | github | basic
        fromEnv: string            # environment variable containing token

  # -----------------------------
  # Helm chart sources
  # -----------------------------
  helm:
    - repo: string                 # Helm repo URL or local path
      chart: string                # chart name
      version: string              # chart version
      valueFiles:                  # list of values.yaml overrides
        - string

  # -----------------------------
  # Orkestra Registry sources
  # -----------------------------
  registry:
    - katalog: string              # name of the katalog in the registry
      version: string              # semantic version or tag
      description: string          # optional human description
      alias: string                # optional alias for merging
      # future: constraints, channels, etc.

# -------------------------------------------------------------------
# Spec: CRD-level overrides
# -------------------------------------------------------------------
spec:
  crds:
    - name: string                 # CRD name (matches katalog)
      enabled: boolean             # enable/disable this CRD
      namespaced: boolean          # override namespaced mode
      workers: integer             # worker count override
      resync: duration             # resync period override
      description: string          # human description

      # -------------------------
      # API type overrides
      # -------------------------
      apiTypes:
        group: string
        version: string
        kind: string
        plural: string
        location: string           # optional Go package for hooks/constructors

      # -------------------------
      # Reconciler overrides
      # -------------------------
      reconciler:
        default: boolean           # use GenericReconciler
        hooks:
          location: string         # Go module path
          function: string         # exported function name
          alias: string            # optional alias for merging
        constructor:
          location: string         # Go module path
          function: string         # exported constructor
          alias: string

      # -------------------------
      # Dependency overrides
      # -------------------------
      dependsOn:
        - string                   # list of CRD names
```