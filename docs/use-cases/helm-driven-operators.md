# Helm‑Driven Operator Configuration

Teams using Helm can ship Orkestra Katalogs as chart templates.  
The chart becomes the distribution mechanism; Orkestra becomes the runtime.

```yaml
# charts/platform-crds/templates/katalog.yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
spec:
  crds:
    {{- range .Values.crds }}
    - name: {{ .name }}
      apiTypes:
        group: {{ $.Values.apiGroup }}
        version: {{ $.Values.apiVersion }}
        kind: {{ .kind }}
        plural: {{ .plural }}
      reconciler:
        default: true
    {{- end }}
```

---

## Example `values.yaml`

This is what the platform team ships inside the Helm chart.

```yaml
apiGroup: platform.myorg.io
apiVersion: v1alpha1

crds:
  - name: application
    kind: Application
    plural: applications

  - name: project
    kind: Project
    plural: projects

  - name: managednamespace
    kind: ManagedNamespace
    plural: managednamespaces
```

:::tip
The chart defines the *shape* of the operator.  
The Komposer defines the *environment‑specific behavior*.
:::

---

## How end‑users consume the chart (Komposer)

A Komposer can reference a Helm chart as a source.  
This allows teams to override values without forking the chart.

```yaml
# komposer.yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: platform-komposer

sources:
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 1.2.0
      valueFiles:
        - ./values/production.yaml

spec:
  crds:
    - name: application
      workers: 10
      resync: 30s
```

---

## Example `production.yaml` override

This file is referenced by the Komposer above.

```yaml
crds:
  - name: application
    kind: Application
    plural: applications
    workers: 10
    resync: 30s

  - name: project
    kind: Project
    plural: projects

  - name: managednamespace
    kind: ManagedNamespace
    plural: managednamespaces
```

:::note
    Helm becomes the distribution mechanism; Orkestra becomes the runtime.  
    The platform team ships the chart.  
    Each environment applies overrides through a Komposer.

---

## Related Documentation

- **Concept:** [Komposer Sources](../runtime-manual/concepts/komposer.md#sources)
- **Reference:** [Helm Source](../reference/komposer-schema.md#helm)
- **Next Use Case:** [Multi‑Team Composition](./multi-team-composition.md)
