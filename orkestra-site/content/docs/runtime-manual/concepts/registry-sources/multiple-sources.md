---
title: "Multiple Sources"
weight: 134
---

# Composing multiple registry sources

Registry entries compose with all other source types in a Komposer. Sources
are processed in declaration order within each source type, with this merge
priority:

```
sources.registry   →  loaded first
sources.files      →  loaded next
sources.helm       →  loaded next
spec.crds          →  inline overrides, loaded last — always win on name conflict
```

```yaml
sources:
  registry:
    - url: ghcr.io/orkestra-sh/orkestra-registry/postgres@v14
      oci: true
    - url: ghcr.io/orkestra-sh/orkestra-registry/redis@v7
      oci: true
  files:
    - ./internal-crds.yaml
spec:
  crds:
    postgres:
      workers: 8   # overrides the registry pattern's default
```

{{< callout type="note" >}}
Duplicate CRD names across sources are an error unless one of the
duplicates is in `spec.crds` — the inline override block. An override
replaces the source definition silently. A duplicate between two
non-inline sources is always an error.
{{< /callout >}}
