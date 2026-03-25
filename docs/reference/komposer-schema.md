# 🎼 **Komposer Schema**

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer

metadata:
  name: string
  description: string

sources:
  files:
    - string | 
      url: string
      auth:
        type: string            # bearer | github | basic
        fromEnv: string         # environment variable name

  helm:
    - repo: string              # path or URL
      chart: string
      version: string
      valueFiles:
        - string

  registry:
    - katalog: string
      version: string
      description: string

spec:
  crds:
    - name: string
      enabled: boolean
      namespaced: boolean
      workers: integer
      resync: duration
      description: string

      apiTypes:
        group: string
        version: string
        kind: string
        plural: string
        location: string

      reconciler:
        default: boolean
        hooks:
          location: string
          function: string
          alias: string
        constructor:
          location: string
          function: string
          alias: string

      dependsOn:
        - string
```
