---
title: "Index"
weight: 133
---

# Registry Sources

Registry sources allow Orkestra to pull operator patterns from external locations
(OCI registries or Git-based repositories) and load them into a Komposer. They
are declared under `sources.registry` and provide the CRD definitions,
reconcile templates, conversion rules, and example resources that make up an
operator pattern.

```yaml
sources:
  registry:
    - url: ghcr.io/orkestra-sh/orkestra-registry/postgres@v14
      oci: true
```

This pulls the `postgres` pattern at version `v14`, validates its structure, and
loads its `katalog.yaml` into your Komposer — ready for inline overrides.

---

## Sections

- Pattern structure
- URL formats and shorthand rules
- Version semantics
- OCI vs Git sources
- Loading katalog.yaml vs komposer.yaml
- Authentication
- Error reference
- Composition with other source types

Each section is documented in its own page for clarity and maintainability.

---

## Related Documentation

- [Typed CRDs](../typed-crds.md)
- [Katalog Schema](../../../reference/katalog-schema.md)
- [Komposer Schema](../../../reference/komposer-schema.md)
- [Registry Schema](../../../reference/registry-schema.md)