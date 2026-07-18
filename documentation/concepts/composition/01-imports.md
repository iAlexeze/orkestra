# imports:

`imports:` brings in a separately maintained unit — a Motif, a Katalog, a test suite. The imported unit lives outside the current file, may have its own version, and can be shared across many consumers. It is the cross-unit composition primitive.

---

## Where imports: appears

### 1. Katalog — `spec.imports` (Motif, Katalog-level)

Merges a Motif's **notes** and **profiles** into the Katalog-wide registries. Every CRD in the Katalog can call the merged notes and reference the merged profiles.


```yaml
spec:
  imports:
    - motif: ./motifs/team-notes/motif.yaml
    - motif: oci://ghcr.io/myorg/patterns/motifs/tenant-isolation:v1.2.0
```

What is merged: `notes.*`, `profiles.*`

→ [Motif schema](../../reference/schema/01-motif/index.md) | [Notes concept](../notes/index.md) | [Profiles concept](../profiles/10-user-defined-profiles.md)

---

### 2. Katalog — `spec.crds[name].imports` (Motif, CRD-level)

Merges a Motif's **resources**, **status**, and **admission** into a specific CRD entry. The Motif declares named inputs; the Katalog binds live CR field values to those inputs via `with:`.

```yaml
spec:
  crds:
    platform:
      imports:
        - motif: oci://ghcr.io/orkspace/patterns/motifs/postgres:v0.1.0
          with:
            image: "{{ .spec.image }}"
            volumeSize: "{{ .spec.storage | default \"10Gi\" }}"
```

What is merged: `resources.onCreate`, `resources.onReconcile`, `status`, `admission`

The `with:` bindings are Go template expressions evaluated against the live CR at reconcile time. Inputs not bound in `with:` use their declared `default` from the Motif.

→ [Motif schema](../../reference/schema/01-motif/index.md)

---

### 3. Komposer — `imports:` (Katalog aggregator)

Composes multiple Katalogs into one runtime from three source types — local files, OCI registry, and Helm charts — combined in a single `imports:` block:

```yaml
imports:
  files:
    - ./website-katalog.yaml
    - ./api-katalog.yaml

  registry:
    - url: ghcr.io/orkspace/orkestra-registry/patterns/katalogs/postgres:v1.0.0
      oci: true

  helm:
    - repo: https://github.com/myorg/charts.git
      chart: my-operator
      path: charts/my-operator
      version: v1.2.0
```

A Komposer can also declare `spec.crds` overrides that win over any imported source — environment-specific tuning without touching the source Katalogs.

→ [Komposer schema](../../reference/schema/03-komposer/index.md)

---

### 4. E2E — `imports:` (E2E aggregator)

Composes multiple E2E files into one test suite. Each imported file runs its own scenarios; the aggregator runs them all under one name.

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: temporal-suite
imports:
  - ./01-business-hours/e2e.yaml
  - ./02-maintenance-window/e2e.yaml
  - ./03-regional-peak/e2e.yaml
```

→ [E2E schema](../../reference/schema/04-e2e/index.md)

---

### 5. Simulate — `imports:` (Simulate aggregator)

Same pattern as E2E — composes multiple Simulate files into one simulation run. No cluster required.

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Simulate
metadata:
  name: temporal-suite-sim
imports:
  - ./01-business-hours/simulate.yaml
  - ./02-maintenance-window/simulate.yaml
  - ./03-regional-peak/simulate.yaml
```

→ [Simulate schema](../../reference/schema/05-simulate/index.md)

---

## Motif sources

When using Motifs in `spec.imports` or `spec.crds[name].imports`, three source types are supported:

| Source | Syntax | Use case |
|--------|--------|----------|
| Local file | `motif: ./path/to/motif.yaml` | Same repository, active development |
| OCI image | `motif: oci://ghcr.io/org/patterns/motifs/name:tag` | Versioned, distributed across teams |
| Git | `motif: git://github.com/org/repo//path/to/motif.yaml@ref` | Pinned to a branch or commit |

Local file Motifs support `include:` (paths resolve relative to the Motif file). OCI and Git Motifs are pre-packaged artifacts — `include:` is a local authoring feature that has already been expanded before publishing.

---

## What each import context contributes

| Context | Notes | Profiles | Resources | Status | Admission |
|---------|-------|----------|-----------|--------|-----------|
| `spec.imports` | ✓ | ✓ | — | — | — |
| `spec.crds[name].imports` | — | — | ✓ | ✓ | ✓ |
| Komposer `imports:` | via Katalog | via Katalog | via Katalog | via Katalog | via Katalog |

<!-- → Back: [Composition overview](index.md) | Next: [include:](02-include.md) -->
