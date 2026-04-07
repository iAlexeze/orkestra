# The `auth` field

Optional authentication for the registry. Credentials are resolved from
environment variables — never from literal values in YAML.

```yaml
auth:
  type: github
  fromEnv: GITHUB_TOKEN
```

## Supported auth types

| Type | Fields | Used for |
|---|---|---|
| `github` | `fromEnv` | GitHub private repos and GHCR |
| `bearer` | `fromEnv` | Generic bearer token APIs |
| `basic` | `usernameFromEnv`, `passwordFromEnv` | Private OCI registries, Artifactory, Nexus |

```yaml
# GitHub token — private repo or GHCR
auth:
  type: github
  fromEnv: GITHUB_TOKEN

# Bearer token — generic API
auth:
  type: bearer
  fromEnv: MY_REGISTRY_TOKEN

# Basic auth — private OCI registry
auth:
  type: basic
  usernameFromEnv: REGISTRY_USER
  passwordFromEnv: REGISTRY_PASSWORD
```

## Security by Design

!!! warning "Never put credentials in YAML"
    The `auth` block resolves values from environment variables. There is no
    field to put a literal token. If you find yourself attempting to paste a
    token into a Komposer, use `fromEnv` and set the environment variable
    instead.

```yaml
# Wrong — and unsupported
auth:
  type: bearer
  token: ghp_myactualtoken123   # this field does not exist

# Correct
auth:
  type: bearer
  fromEnv: GITHUB_TOKEN        # set GITHUB_TOKEN in your environment
```

!!! tip "In Kubernetes deployments"
    Mount credentials as environment variables from a Kubernetes Secret:

```yaml
env:
  - name: GITHUB_TOKEN
    valueFrom:
      secretKeyRef:
        name: orkestra-registry-creds
        key: github-token
```

