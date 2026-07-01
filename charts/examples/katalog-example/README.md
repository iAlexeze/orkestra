# orkestra-katalog-example

A minimal Helm chart that renders an **Orkestra Katalog**.  
This chart demonstrates how platform teams can package CRD behavior definitions inside a Helm chart so they can be consumed through `sources.helm` in a Komposer file.

This example is intentionally small and is meant as a reference for building your own catalog‑producing charts.

---

## What This Chart Provides

- A rendered **Katalog** resource containing multiple CRD entries.
- A simple, declarative way to define CRD metadata (group, version, kind, plural, workers, resync, reconciler mode, etc.).
- A pattern for shipping Orkestra CRD definitions via Helm instead of raw YAML.

> [!NOTE]
> This chart does **not** install CRDs or operators.  
> It only emits a **Katalog** manifest.

---

## Values

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `apiGroup` | string | `demo.orkestra.io` | API group applied to all CRDs in this chart. |
| `apiVersion` | string | `v1alpha1` | API version applied to all CRDs. |
| `crds` | list | — | List of CRD entries to include in the generated Katalog. |

Each entry under `crds` produces one `CRDEntry` in the final Katalog.

Example:

```yaml
crds:
  - name: database
    enabled: true
    kind: Database
    plural: databases
    namespaced: false
    workers: 2
    resync: 1m
    reconciler:
      default: true
      mode: dynamic
    description: Manages a database instance.
```

---

## **Using With Komposer**

```yaml
sources:
  helm:
    - repo: ghcr.io/orkspace/charts
      chart: orkestra-katalog-example
      path: katalog-example
      version: 0.1.0
```

Komposer will pull the chart, render the Katalog, and merge it with other sources.
