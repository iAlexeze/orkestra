---
title: "Writing Your First Komposer"
weight: 26
---

# Writing Your First Komposer

A **Komposer** tells Orkestra *where* to load katalogs from.  
While a Katalog defines **what** your operator does, the Komposer defines **where those katalogs come from**.

Komposers support multiple source types:

- Local files  
- Remote URLs  
- Helm charts  
- Git registries (public or private)  
- Multiple sources merged together  
- Inline overrides  

{{< callout type="note" >}}
Komposers do not perform reconciliation.  
They only load katalogs and merge them into a final, resolved state.
{{< /callout >}}

---

## The Simplest Komposer

Create a file called `komposer.yaml`:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: my-first-komposer

sources:
  files:
    - ./my-katalog.yaml
```

This Komposer tells Orkestra:

- Load a katalog from a local file  
- Use it exactly as written  
- No overrides, no merging, no registry lookups  

{{< callout type="tip" >}}
This is the recommended starting point.  
Always begin with a single-file Komposer to validate your katalog.
{{< /callout >}}

---

## Loading Multiple Files

You can load multiple katalogs from multiple files:

```yaml
sources:
  files:
    - ./katalogs/app.yaml
    - ./katalogs/database.yaml
    - ./katalogs/cache.yaml
```

{{< callout type="note" >}}
When multiple katalogs define the same CRD, Orkestra merges them in order.  
Later entries override earlier ones.
{{< /callout >}}

---

## Loading from a Remote URL

You can load katalogs directly from a URL:

```yaml
sources:
  files:
    - https://example.com/katalogs/webapp.yaml
```

{{< callout type="caution" >}}
Remote URLs must return raw YAML.  
HTML pages, redirects, or GitHub “blob” URLs will fail.
{{< /callout >}}

---

## Loading from a Helm Chart

Komposers can load katalogs packaged inside Helm charts:

```yaml
sources:
  helm:
    - repo: https://charts.myorg.io
      chart: platform-katalogs
      version: 1.2.0
```

{{< callout type="note" >}}
Orkestra extracts only katalog files from the chart.  
Other chart templates are ignored.
{{< /callout >}}

---

## Loading from a Git Registry

Komposers can pull katalogs from Git repositories.  
This is the most powerful and common source type.

```yaml
sources:
  registry:
    - url: https://github.com/myorg/orkestra-registry
      katalog:
        webapp:
          branch: main
        database:
          version: v1.0.3
```

This tells Orkestra:

- Clone the repository  
- Load the katalog named `webapp` from the `main` branch  
- Load the katalog named `database` from tag `v1.0.3`  

{{< callout type="tip" >}}
If you omit `url`, Orkestra uses the `ORK_REGISTRY` environment variable as the default registry.
{{< /callout >}}

---

## Private Registries

Private GitHub or GitLab registries require authentication:

```yaml
sources:
  registry:
    - url: https://github.com/myorg/private-registry
      auth:
        type: github
        fromEnv: GITHUB_TOKEN
      katalog:
        internal-app:
          branch: main
```

{{< callout type="warning" >}}
Never hardcode tokens in your Komposer.  
Always load them from environment variables.
{{< /callout >}}

---

## Mixing Multiple Source Types

Komposers can combine any number of sources:

```yaml
sources:
  files:
    - ./local/base.yaml

  helm:
    - repo: https://charts.myorg.io
      chart: platform-katalogs
      version: 3.0.0

  registry:
    - url: https://github.com/myorg/orkestra-registry
      katalog:
        webapp:
          branch: main
```

{{< callout type="note" >}}
Sources are merged in the order they appear.  
Later sources override earlier ones.
{{< /callout >}}

---

## Inline Overrides

You can override katalog fields directly inside the Komposer:

```yaml
spec:
  crds:
    webapp:
      workers: 4
      operatorBox:
        default: true
```

Inline overrides apply **after** all sources are merged.

{{< callout type="tip" >}}
Use inline overrides for environment‑specific settings such as worker counts, namespaces, or resource limits.
{{< /callout >}}

---

## Complete Example

Here is a Komposer that loads katalogs from multiple sources and applies overrides:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: platform-komposer

sources:
  files:
    - ./local/base.yaml

  helm:
    - repo: https://charts.myorg.io
      chart: platform-katalogs
      version: 3.0.0

  registry:
    - url: https://github.com/myorg/orkestra-registry
      katalog:
        webapp:
          branch: main
        database:
          version: v1.2.0

spec:
  crds:
    webapp:
      workers: 6
      operatorBox:
        default: true
```

This Komposer:

- Loads a local katalog  
- Loads a Helm chart  
- Loads two katalogs from a Git registry  
- Overrides the `webapp` CRD to use 6 workers  

---

## Next Steps

You now know how to load katalogs from files, URLs, Helm charts, and registries — and how to override them.

Continue with:

**Basic Reconciliation**  
Learn how Orkestra watches CRDs, processes CRs, and applies your katalog to the cluster.
