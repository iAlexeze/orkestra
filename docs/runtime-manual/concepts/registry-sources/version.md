# The `version` field

The version, tag, branch, or SHA to pull from the registry.

```yaml
- url: ghcr.io/orkestra-sh/orkestra-registry/postgres
  version: v14.2.0   # OCI tag

- url: https://github.com/myorg/registry
  version: main      # Git branch

- url: https://github.com/myorg/registry
  version: v1.0.0    # Git tag

- url: https://github.com/myorg/registry
  version: abc123    # Git SHA (full or partial)
```

## Defaults when omitted

| `oci` field | Default version |
|---|---|
| `true` | `latest` |
| `false` | `main` |

!!! tip "Pin versions in production"
    Tracking `main` or `latest` in production means your operator behavior
    can change when the upstream pattern is updated. This is usually not
    what you want at runtime. Pin to a specific version tag for stability,
    and upgrade deliberately.

```yaml
# Production — pinned
- url: ghcr.io/orkestra-sh/orkestra-registry/postgres@v14.2.0
  oci: true

# Development — track latest
- url: ghcr.io/orkestra-sh/orkestra-registry/postgres@latest
  oci: true
```

!!! warning "OCI version tags are immutable"
    Once `postgres:v14.2.0` is published to an OCI registry, it cannot be
    overwritten. This is a registry convention, not an Orkestra constraint.
    It guarantees that pinned version references never change behavior
    unexpectedly. If you discover a bug in a published version, publish a
    new patch tag (`v14.2.1`) — do not overwrite the old one.

