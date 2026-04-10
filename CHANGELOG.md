# Changelog

## [Unreleased] — Templating with Environment Variables

```yaml
onCreate:
  secrets:
    - name: "{{ .metadata.name }}-credentials"
      data:
        username: "{{ .spec.username }}"
        password: "{{ randomAlphanumeric 32 }}"
  deployments:
    - name: "{{ .metadata.name }}"
      env:
        USERNAME:
          secretKeyRef:
            name: "{{ .metadata.name }}-credentials"
            key: username
      envFrom:
        - secretRef: "{{ .metadata.name }}-credentials"
```

### Added
- Template resolution for `env` and `envFrom` in Deployment hooks.
- Support for referencing secrets/configmaps created in the same reconcile cycle via `secretKeyRef`, `configMapKeyRef`, and `envFrom`.

### Fixed
- Deployments now correctly mount environment variables defined in the Katalog.