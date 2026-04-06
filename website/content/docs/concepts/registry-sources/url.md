---
title: "The `url` field"
weight: 50
description: "The URL of the registry source supports two forms:"
---

The URL of the registry source supports two forms:

## 1. Shorthand — version embedded with `@`

```yaml
- url: ghcr.io/orkestra-sh/orkestra-registry/postgres@v14
- url: https://github.com/myorg/registry@main
- url: https://github.com/myorg/registry@abc123def
```

## 2. Explicit — URL and version separately

```yaml
- url: ghcr.io/orkestra-sh/orkestra-registry/postgres
  version: v14.2.0
```

!!! note
    When `@` is present in the URL, the `version` field is ignored entirely.
    The `@` form is the shorthand — it takes priority. Use the explicit form
    when you want to declare URL and version on separate lines for readability
    in longer Komposers.

## URL formats by source type

| Source | Example URL |
|---|---|
| OCI (GHCR) | `ghcr.io/orkestra-sh/orkestra-registry/postgres` |
| OCI (Docker Hub) | `docker.io/myorg/my-operator` |
| OCI (private) | `registry.myorg.com/operators/internal-crd` |
| GitHub | `https://github.com/myorg/orkestra-registry` |
| GitLab | `https://gitlab.com/myorg/orkestra-registry` |
| Generic Git | `https://git.myorg.com/platform/operators` |

!!! warning "Do not include `oci://` scheme prefixes in URLs"
    Orkestra constructs the correct protocol internally based on the `oci`
    field. Writing `oci://ghcr.io/...` will cause a URL parse error.

```yaml
# Wrong
- url: oci://ghcr.io/orkestra-sh/orkestra-registry/postgres@v14
  oci: true

# Correct
- url: ghcr.io/orkestra-sh/orkestra-registry/postgres@v14
  oci: true
```

