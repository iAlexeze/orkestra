# Motif

A Motif is the smallest reusable unit in Orkestra's composition model. It declares named inputs and resource blocks. It cannot run alone — it must be imported by a Katalog that provides its inputs via `with:`.

```text
Motif     — smallest reusable unit. Declared inputs. One concern.
    ↓
Katalog   — operator declaration. Imports Motifs.
    ↓
Komposer  — platform declaration. Composes Katalogs.
```

## Wire format

```yaml
apiVersion: orkestra.orkspace.io/v1   # required
kind: Motif                            # required

metadata:
  name: postgres                       # required
  version: v16
  description: PostgreSQL StatefulSet with PVC, headless Service, and pgAdmin.
  author: orkspace
  license: Apache-2.0
  tags:
    - database

inputs:
  - name: image
    description: PostgreSQL image (e.g. postgres:16)
    required: true

  - name: volumeSize
    description: PVC storage size
    default: "10Gi"

resources:
  onCreate:
    ...
  statefulsets:
    ...
  custom:
    ...
  services:
    ...

status:
  ...

admission:
  validation:
    ...
  mutation:
    ...
```

## `metadata`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Motif identifier. Used as the registry artifact name. |
| `version` | no | Semver or tag. Shown in `ork registry list`. |
| `description` | no | Short description shown in the registry UI. |
| `author` | no | Author or org name. |
| `license` | no | SPDX license identifier (e.g. `Apache-2.0`). |
| `tags` | no | List of tags for discoverability. |

## `inputs`

Named parameters the Motif exposes to consumers. Referenced inside resources as `inputs.<name>`.

```yaml
inputs:
  - name: image
    description: PostgreSQL image (e.g. postgres:16)
    required: true

  - name: volumeSize
    description: PVC storage size
    default: "10Gi"

  - name: user
    default: "postgres"
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Input identifier. Referenced as `inputs.<name>` in templates. |
| `description` | no | What this input controls. |
| `required` | no | When `true`, the importer must provide this input in `with:`. Missing required inputs are caught by `ork validate` and at startup — not at reconcile time. |
| `type` | no | Type hint (`string`, `int`, `bool`). Reserved for future enforcement. |
| `default` | no | Value used when the input is not provided in `with:`. Only valid when `required` is `false`. |

## `resources`

Resource blocks the Motif contributes to the CRD entry. Follows the same schema as `operatorBox` resources.

```yaml
resources:
  onCreate:
    secrets:
      - name: "{{ .metadata.name }}-creds"
        once: true
        data:
          password: "{{ randAlphaNum 20 }}"

  statefulsets:
    - name: "{{ .metadata.name }}-postgres"
      image: "{{ inputs.image }}"
      replicas: 1
      volumeClaims:
        - name: pgdata
          size: "{{ inputs.volumeSize }}"
          mountPath: /var/lib/postgresql/data
      reconcile: true

  services:
    - name: "{{ .metadata.name }}-postgres"
      port: 5432
      reconcile: true
```

| Block | Merged into | Description |
|-------|-------------|-------------|
| `resources.onCreate` | CRD `onCreate` | Resources that run only on CR creation. Never updated on subsequent reconciles. Secrets with `once: true` belong here. |
| All other resource blocks (`statefulsets`, `services`, etc.) | CRD `onReconcile` | Merged into the normal reconcile phase. |

Template context inside a Motif:
- `.metadata.name`, `.metadata.namespace` — the CR being reconciled (not the Motif itself)
- `inputs.<name>` — the Motif's own input values, bound by the consumer's `with:`

A Motif does not know what CRD is importing it. The mapping from CR fields to inputs is the consumer's responsibility, expressed in `with:`.

## `status`

Same schema as the Katalog [status](05-status.md) block. Fields declared here are contributed to the parent CRD's status.

```yaml
status:
  fields:
    - path: postgresReady
      value: "{{ replicasReady .children.statefulset }}"
    - path: connectionString
      value: "postgres://{{ inputs.user }}@{{ .metadata.name }}-postgres.{{ .metadata.namespace }}.svc.cluster.local:5432"
```


## `admission`

Validation and mutation rules scoped to this Motif. Merged with the parent CRD's admission rules.

```yaml
admission:
  validation:
    rules:
      - field: spec.image
        prefix: "myregistry.com/"
        action: deny

  mutation:
    rules:
      - field: spec.replicas
        default: "2"
```


## Importing a Motif

Motifs are imported at the CRD entry level via `imports:`, alongside `operatorBox`.

```yaml
spec:
  crds:
    database:
      apiTypes:
        ...
      imports:
        # Local file (development)
        - motif: ./motifs/postgres/motif.yaml
          with:
            image: "postgres:16"
            passwordSecretName: "{{ .metadata.name }}-secrets"

        # OCI registry
        - motif: oci://ghcr.io/orkspace/orkestra-services/postgres:v16
          oci: true
          with:
            image: "{{ .spec.postgresImage }}"
            passwordSecretName: "{{ .metadata.name }}-secrets"
            volumeSize: "{{ .spec.storage | default \"10Gi\" }}"

        # Git URL
        - motif: https://github.com/myorg/postgres-motif@main
          with:
            image: "{{ .spec.postgresImage }}"
```

### `imports` fields

| Field | Required | Description |
|-------|----------|-------------|
| `motif` | yes | File path, OCI reference (`oci://...`), or Git URL. Use `@version` shorthand to pin: `postgres-motif@v16`. |
| `version` | no | Explicit version (tag, branch, SHA). Ignored when `@` shorthand is used in `motif`. Defaults to `latest` (OCI) or `main` (Git). |
| `oci` | no | `true` to pull via OCI/ORAS protocol. |
| `auth` | no | Credentials for the registry. Same auth model as `imports.files` in a Komposer. |
| `with` | no | Input bindings. Values are Go templates evaluated in the CR's reconcile context. |

### `with:` binding rules

- Required inputs not in `with:` → validation error (caught at startup, not at reconcile time)
- Optional inputs not in `with:` → Motif default is used
- `with:` values are Go templates: `"{{ .spec.postgresImage }}"`, `"{{ .metadata.name }}-secrets"`

### Composing multiple Motifs

A Katalog can import any number of Motifs. Each import is independent — resources do not conflict because each uses `.metadata.name` as a prefix.

```yaml
spec:
  crds:
    myapp:
      imports:
        - motif: oci://ghcr.io/orkspace/orkestra-services/postgres:v16
          oci: true
          with:
            image: "{{ .spec.database.image }}"
            passwordSecretName: "{{ .metadata.name }}-db-secrets"

        - motif: oci://ghcr.io/orkspace/orkestra-services/redis:v7
          oci: true
          with:
            image: "{{ .spec.cache.image }}"
            volumeSize: "{{ .spec.cache.volumeSize }}"

      # The operator's own resources alongside the imported Motifs
      operatorBox:
        deployments:
          - name: "{{ .metadata.name }}"
            image: "{{ .spec.image }}"
            reconcile: true
```

A Motif can itself import other Motifs — a `postgres-with-backup` Motif could import the base `postgres` Motif and add a backup CronJob.

## Pattern directory structure

```text
postgres/
  motif.yaml      # required — the Motif spec
  README.md       # optional — shown in registry UI
  example/
    katalog.yaml  # optional — example Katalog importing this Motif
```

`motif.yaml` is the only required file. The registry identifies this pattern as `kind: Motif` from the file content.

---

## See also

- [katalog.md](01-katalog.md) — where `operatorBox.imports` is declared
- [operatorbox.md](04-operatorbox.md) — full operatorBox schema
- [komposer.md](13-komposer.md) — composing multiple Katalogs
- [index.md](index.md) — full schema reference index
