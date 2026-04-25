---
title: "Komposer Schema"
weight: 97
---

# Komposer Schema

A Komposer composes multiple Katalogs from different sources — files, URLs,
Helm charts, and registry entries — into a single unified operator
configuration. It is the declarative way to build platform-wide operator
layers from reusable pieces.

The Komposer is the distribution unit. Platform teams publish Katalogs.
Teams compose them into Komposers. Environment-specific overrides are inline
`spec.crds` entries that win over any source definition.

---

## Document structure

```yaml
apiVersion: orkestra.orkspace.io/v1    # required
kind: Komposer                               # required

metadata:
  name: string        # required
  description: string # optional

sources:              # required — at least one source must be declared
  files: [...]
  helm: [...]
  registry: [...]

spec:
  crds:               # optional — inline CRD overrides, merged last
    - ...
```

---

## `metadata`

### `metadata.name`

**Type:** `string` | **Required:** yes

Unique identifier. Used in logs and the health endpoint.

### `metadata.description`

**Type:** `string` | **Required:** no

Human-readable description. Shown in `ork status` and `ork describe`.

---

## `sources`

At least one source type must be declared. Multiple source types can be
combined freely. Sources are resolved in this order:

1. `sources.registry` — pulled first
2. `sources.files` — resolved next
3. `sources.helm` — rendered and extracted last
4. `spec.crds` — inline overrides, applied after all sources, always win

CRD names must be unique across all sources combined. A duplicate CRD name
across two non-inline sources is a fatal error.

{{< callout type="note" >}}
`spec.crds` is the only exception to the duplicate rule. An inline
override with the same name as a source CRD replaces the source
definition silently. This is the intended override mechanism.
{{< /callout >}}

---

## `sources.files[]`

Local files, remote URLs, or environment variable references.

### Simple form

```yaml
sources:
  files:
    - ./katalogs/website.yaml
    - https://platform.myorg.io/crds/database.yaml
    - $SECURITY_KATALOG_URL        # resolved from environment at startup
```

### Authenticated form

```yaml
sources:
  files:
    - url: https://private.myorg.io/katalogs/internal.yaml
      auth:
        type: bearer
        fromEnv: INTERNAL_KATALOG_TOKEN
```

| Field | Type | Required | Description |
|---|---|---|---|
| `url` | string | yes | Local path, remote URL, or `$ENV_VAR` reference |
| `auth.type` | string | no | `bearer`, `github`, or `basic` |
| `auth.fromEnv` | string | for bearer/github | Environment variable containing the token |
| `auth.usernameFromEnv` | string | for basic | Environment variable containing the username |
| `auth.passwordFromEnv` | string | for basic | Environment variable containing the password |

**Auth types:**

| Type | Header sent | Use case |
|---|---|---|
| `bearer` | `Authorization: Bearer <token>` | Generic REST APIs, Artifactory |
| `github` | `Authorization: Bearer <token>` | GitHub raw URLs, private repos |
| `basic` | `Authorization: Basic <b64>` | HTTP Basic auth, Nexus |

{{< callout type="warning" >}}
Environment variables are resolved at startup — not at build time.
If `$SECURITY_KATALOG_URL` is unset when `ork run` starts, it resolves
to an empty string and the source is skipped with a warning.
{{< /callout >}}

{{< callout type="warning" >}}
A Komposer file declared as a `sources.files` entry is a fatal error.
Only Katalog files are valid file sources. A Komposer cannot source
another Komposer through the files block.
{{< /callout >}}

    ```
    error: "platform.yaml" sources.files["team.yaml"]: a Komposer cannot
    source another Komposer — only Katalog files are valid sources
    ```

---

## `sources.helm[]`

Renders a Helm chart and extracts any Katalog templates from the output.
The chart must contain at least one template with `kind: Katalog`.

```yaml
sources:
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 2.1.0
      valueFiles:
        - ./values/production.yaml
        - $TEAM_VALUES_FILE
      values:
        workers: 4
        resync: 30s
```

| Field | Type | Required | Description |
|---|---|---|---|
| `repo` | string | yes | Helm repo URL, local chart directory, or Git URL (`.git` suffix) |
| `chart` | string | yes | Chart name within the repo, or subdirectory path for Git sources |
| `version` | string | yes* | Chart version tag. For local charts, optional. |
| `path` | string | no | Path within the Git repo to the chart directory |
| `valueFiles` | `[]string` | no | Local paths, remote URLs, or `$ENV_VAR` references to values files |
| `values` | `object` | no | Inline values. Applied after `valueFiles`. Equivalent to `helm --set`. |

\* Required for remote Helm repos and Git sources. Optional for local directories.

**Repo source types:**

| Repo format | Mechanism |
|---|---|
| `https://charts.myorg.io` | Remote Helm repo — downloaded via Helm protocol |
| `./charts/platform-crds` | Local directory — loaded directly |
| `https://github.com/myorg/charts.git` | Git repo — shallow cloned at `version` ref |
| `git@github.com:myorg/charts.git` | Git SSH — cloned at `version` ref |

{{< callout type="warning" title="Chart must produce at least one Katalog template" >}}
If the rendered chart contains no documents with `kind: Katalog`, the
Helm source fails with:
{{< /callout >}}

    ```
    error: chart "platform-crds" produced no Katalog templates — ensure at
    least one template has kind: Katalog
    ```

{{< callout type="tip" >}}
To use a Helm chart as a Katalog source, create a `templates/katalog.yaml`
in the chart that renders the `Katalog` document using Helm's templating.
Workers, resync, and other per-CRD settings can be driven by Helm values.
{{< /callout >}}

---

## `sources.registry[]`

Pulls a complete operator pattern from a registry — OCI or Git — validates
its five-file structure, and loads either `katalog.yaml` or `komposer.yaml`.

```yaml
sources:
  registry:
    - url: ghcr.io/konduktor-io/orkestra-registry/postgres@v14
      oci: true

    - url: https://github.com/myorg/registry@main
      auth:
        type: github
        fromEnv: GITHUB_TOKEN
```

| Field | Type | Default | Required | Description |
|---|---|---|---|---|
| `url` | string | — | yes | Registry URL. Supports `@version` shorthand. |
| `version` | string | `main`/`latest` | no | Version, branch, tag, or SHA. Ignored when `@` is in `url`. |
| `oci` | bool | `false` | no | Pull as OCI artifact (ORAS). False = Git or raw HTTP. |
| `useKomposer` | bool | `false` | no | Load `komposer.yaml` instead of `katalog.yaml`. |
| `auth.type` | string | — | no | `bearer`, `github`, or `basic` |
| `auth.fromEnv` | string | — | for bearer/github | Environment variable containing the token |
| `auth.usernameFromEnv` | string | — | for basic | Environment variable containing the username |
| `auth.passwordFromEnv` | string | — | for basic | Environment variable containing the password |

**URL shorthand:**

```yaml
# These are equivalent:
- url: ghcr.io/konduktor-io/orkestra-registry/postgres@v14
  oci: true

- url: ghcr.io/konduktor-io/orkestra-registry/postgres
  version: v14
  oci: true
```

**Version defaults:**

| `oci` value | Default version |
|---|---|
| `true` | `latest` |
| `false` | `main` |

**Required pattern files (validated at pull time):**

Every registry source must contain all five files, each non-empty:

| File | Purpose |
|---|---|
| `crd.yaml` | The CRD definition to install |
| `katalog.yaml` | Operator behavior and templates |
| `komposer.yaml` | Example showing how to import the pattern |
| `cr.yaml` | Example custom resource |
| `README.md` | Documentation |

{{< callout type="warning" title="Fail fast on missing or empty files" >}}
Orkestra refuses to load a pattern that is missing any of the five required
files, or where any file is empty. This check runs during `ork validate`.
{{< /callout >}}

For full registry source documentation, see [Registry Sources](../runtime-manual/concepts/registry.md).

---

## `spec.crds[]` — inline overrides

CRD entries in `spec.crds` on a Komposer are overrides. They are merged last
and always win on name conflict with any source.

Use this block for environment-specific adjustments — not for defining new
CRDs from scratch. New CRDs belong in a Katalog file.

```yaml
spec:
  crds:
    # Override workers for production
    postgres:
      workers: 8
      resync: 30s

    # Disable a CRD from a sourced Katalog in this environment
    debug-dashboard:
      enabled: false
```

All fields from the Katalog CRD entry schema are valid here. Only the
fields you declare are overridden — unset fields inherit from the source
definition.

{{< callout type="note" title="Inline CRDs can also define new CRDs" >}}
If the `name` in `spec.crds` does not match any source CRD, it is added
as a new CRD entry. This is valid — a Komposer can define CRDs inline
without requiring a source file.
{{< /callout >}}

{{< callout type="warning" title="Duplicate names within `spec.crds` are an error" >}}
A name that appears twice in the inline block is always an error,
regardless of whether it matches a source CRD.
{{< /callout >}}

    ```
    error: "platform.yaml" spec.crds: duplicate CRD "postgres" — each CRD
    name must be unique within the inline block
    ```

---

## Error reference

### Source resolution errors

```
error: reading "https://private.myorg.io/katalog.yaml": authentication
required (401) — check that auth credentials are set and have not expired
```
The remote file requires auth. Add an `auth` block and set the environment variable.

```
error: reading "https://private.myorg.io/katalog.yaml": access denied (403)
— check that the token has sufficient permissions
```
Token is valid but lacks read permission. Check repository/registry access.

```
error: reading "https://private.myorg.io/katalog.yaml": not found (404)
— check the URL is correct and the file exists
```
URL is incorrect or the file was moved. Verify the path.

```
warning: $SECURITY_KATALOG_URL is unset — source skipped
```
An environment variable reference resolved to empty. Set the variable and restart.

### Komposer-in-files error

```
error: "platform.yaml" sources.files["team-komposer.yaml"]: a Komposer
cannot source another Komposer — only Katalog files are valid sources
```
A file declared in `sources.files` is a Komposer, not a Katalog. Only Katalog
files are valid file sources. If you want to compose Komposers, use
`sources.registry` with `useKomposer: true`.

### Helm errors

```
error: "platform.yaml" sources.helm[0]: chart "platform-crds" produced no
Katalog templates — ensure at least one template has kind: Katalog
```
The Helm chart rendered successfully but contained no Katalog documents.
Add a `templates/katalog.yaml` to the chart.

```
error: "platform.yaml" sources.helm[0]: rendering chart "platform-crds":
values file "./values/missing.yaml" not found
```
A declared values file does not exist at the given path.

### Registry errors

```
error: registry "ghcr.io/myorg/postgres"@v14 failed structure validation:
  missing: cr.yaml
  empty:   README.md
```
Pattern is incomplete. See [Publishing a Pattern](../runtime-manual/concepts/registry-sources//publishing-a-pattern.md).

```
error: no registry URL configured for this source.
Set the registry URL using one of:
  1. ORK_REGISTRY environment variable
  2. Explicit url in the source block
```
Registry source has no URL and `ORK_REGISTRY` is not set.

### Merge errors

```
error: "platform.yaml": CRD "postgres" declared in both "files/website.yaml"
and "registry/postgres:v14" — each CRD name must appear in exactly one
non-inline source
```
Two sources provide the same CRD name. Move the override to `spec.crds`.

```
error: "platform.yaml" spec.crds: duplicate CRD "postgres" — each CRD name
must be unique within the inline block
```
The same name appears twice in `spec.crds`.

### Dependency graph errors

```
error: circular dependency detected across Komposer sources:
  application → namespace → project → application
```
`dependsOn` declarations across merged CRDs form a cycle. Remove the cycle.
