# Motif — The Reusable Primitive of Orkestra

*Orkestra Project — April 2026*

---

## The musical hierarchy

In music, a motif is the smallest structural unit with thematic identity. It
is a short phrase — four notes, eight bars — that recurs, transforms, and
composes into larger structures. A symphony is not written as a single
declaration. It is assembled from motifs into movements into a complete work.

Orkestra follows the same structure:

```
Motif     — smallest reusable unit. Declared inputs. One concern.
    ↓
Katalog   — operator declaration. One or more CRDs. Imports Motifs.
    ↓
Komposer  — platform declaration. One or more Katalogs. Composes everything.
```

A Motif is to a Katalog what a Katalog is to a Komposer. Each layer composes
the one below it. The smallest thing is complete and reusable. The largest
thing is assembled from complete, reusable parts.

---

## What a Motif is

A Motif is a Orkestra Kind — a first-class artifact with its own schema,
versioning, and OCI distribution. It declares:

- **inputs** — named, typed, with defaults and descriptions
- **resources** — the Orkestra resource blocks (deployments, statefulsets,
  services, secrets, configmaps, etc.) that use `inputs.*` for values
- **status** — optional status field declarations
- **metadata** — name, version, description, author, license (same as Katalog)

A Motif cannot run alone. It has no CRD entry. It must be imported by a
Katalog or another Motif that provides its inputs.

---

## Motif schema

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Motif
metadata:
  name: postgres
  version: v16
  description: >
    PostgreSQL StatefulSet with PVC, headless Service, and pgAdmin.
    Works with postgres:14 through postgres:17.
  author: orkspace
  license: Apache-2.0

inputs:
  # Required inputs — must be provided by the consumer
  - name: image
    description: PostgreSQL image (e.g. postgres:16)
    required: true

  - name: passwordSecretName
    description: Name of the Secret containing POSTGRES_PASSWORD
    required: true

  # Optional inputs — have defaults
  - name: user
    description: Database superuser username
    default: "postgres"

  - name: volumeSize
    description: PVC storage size
    default: "10Gi"

  - name: adminEmail
    description: pgAdmin login email
    default: "admin@example.com"

  - name: adminPassword
    description: pgAdmin login password
    default: "admin"

resources:
  statefulsets:
    - name: "{{ .metadata.name }}-postgres"
      image: "{{ inputs.image }}"              # ← inputs.*, not spec.*
      replicas: 1
      serviceName: "{{ .metadata.name }}-postgres-headless"
      volumeClaims:
        - name: pgdata
          size: "{{ inputs.volumeSize }}"
          mountPath: /var/lib/postgresql/data
      env:
        - name: POSTGRES_USER
          value: "{{ inputs.user }}"
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: "{{ inputs.passwordSecretName }}"
              key: POSTGRES_PASSWORD
      reconcile: true

  services:
    - name: "{{ .metadata.name }}-postgres-headless"
      clusterIP: None
      port: 5432
      reconcile: true
    - name: "{{ .metadata.name }}-postgres"
      port: 5432
      reconcile: true

  deployments:
    - name: "{{ .metadata.name }}-pgadmin"
      image: "dpage/pgadmin4:latest"
      env:
        - name: PGADMIN_DEFAULT_EMAIL
          value: "{{ inputs.adminEmail }}"
        - name: PGADMIN_DEFAULT_PASSWORD
          value: "{{ inputs.adminPassword }}"
      reconcile: true

  services:
    - name: "{{ .metadata.name }}-pgadmin-svc"
      port: 80
      reconcile: true

status:
  fields:
    - path: postgresReady
      value: "{{ replicasReady .children.statefulset }}"
    - path: connectionString
      value: >
        postgres://{{ inputs.user }}@{{ .metadata.name }}-postgres.{{ .metadata.namespace }}.svc.cluster.local:5432
    - path: pgadminUrl
      value: "{{ serviceLoadBalancerHost .children.service }}"
```

Inside a Motif, `.metadata.name` and `.metadata.namespace` refer to the
instantiating resource — the CR or ConfigMap that imported this Motif. `inputs.*`
refers to the Motif's own declared interface. These are the only two contexts
a Motif has. It does not know whether it is being instantiated by a CRD with
`spec.image` or a ConfigMap with `data.postgresImage`. That translation is the
consumer's responsibility, expressed in `with:`.

---

## Importing a Motif — the `with:` block

The `with:` block mirrors GitHub Actions' `with:` syntax deliberately. Developers
who have written a composite action already understand: declare the step,
provide the inputs, leave optional ones at their default.

```yaml
# In a Katalog operatorBox:
operatorBox:
  imports:
    - motif: oci://ghcr.io/orkspace/orkestra-services/postgres:v16
      with:
        image: "{{ .spec.postgresImage }}"
        passwordSecretName: "{{ .metadata.name }}-secrets"
        user: "{{ .spec.postgresUser | default \"postgres\" }}"
        volumeSize: "{{ .spec.storage | default \"10Gi\" }}"
        # adminEmail and adminPassword use their Motif defaults
```

The `with:` values are template expressions evaluated in the Katalog's context
— the CR being reconciled. The result binds to the Motif's `inputs.*`
namespace for the duration of that reconcile.

**Required inputs not provided in `with:` are a validation error** — caught by
`ork validate` and at Katalog startup, not at reconcile time.

**Optional inputs not provided in `with:` use their Motif defaults.**

---

## Composing Motifs

A Katalog can import multiple Motifs. Each import is independent — resources
from different Motifs do not conflict because each uses `.metadata.name` as a
prefix, which is the name of the CR being reconciled.

```yaml
# A complete application operator importing three Motifs
operatorBox:
  imports:
    - motif: oci://ghcr.io/orkspace/orkestra-services/postgres:v16
      with:
        image: "{{ .spec.database.postgresImage }}"
        passwordSecretName: "{{ .metadata.name }}-db-secrets"
        volumeSize: "{{ .spec.database.volumeSize }}"

    - motif: oci://ghcr.io/orkspace/orkestra-services/redis:v7
      with:
        image: "{{ .spec.cache.redisImage }}"
        volumeSize: "{{ .spec.cache.volumeSize }}"

    - motif: oci://ghcr.io/orkspace/orkestra-services/kafka:v3
      with:
        image: "{{ .spec.messaging.kafkaImage }}"

  # Plus the application's own resources
  onCreate:
    when:
      - field: "{{ replicasReady .children.statefulset }}"
        equals: "true"
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        envFrom:
          - secretRef:
              name: "{{ .metadata.name }}-db-secrets"
        reconcile: true
```

A Motif can also import other Motifs — a `postgres-with-backup` Motif could
import the base `postgres` Motif and add a CronJob for backup scheduling.
The composition is recursive.

---

## The two access surfaces

The same postgres Motif, two consumers, different vocabularies.

### Platform engineer path

```yaml
# PostgresCluster CRD, full operator
apiVersion: data.platform.io/v1alpha1
kind: PostgresCluster
metadata:
  name: payments-db
spec:
  postgresImage: "postgres:16"
  postgresUser: "payments"
  storage: "100Gi"

# The Katalog imports the Motif and binds spec.* to inputs.*
imports:
  - motif: oci://ghcr.io/orkspace/orkestra-services/postgres:v16
    with:
      image: "{{ .spec.postgresImage }}"
      user: "{{ .spec.postgresUser }}"
      volumeSize: "{{ .spec.storage }}"
      passwordSecretName: "{{ .metadata.name }}-credentials"
```

Full Orkestra lifecycle. Drift correction. Deletion protection. Ordered
shutdown. Health tracking. The platform engineer adds what they need on top of
the Motif.

### Developer path (ork doctor)

```yaml
# ConfigMap as CRD — ork doctor generates this
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-app
  labels:
    ork.io/app: my-app
data:
  postgresImage: "postgres:16"
  postgresVolumeSize: "10Gi"
  postgresUser: "myusername"

# ork doctor expands the Motif into the Katalog at generation time
# The developer never sees the import — it becomes inline resources
# in .orkestra/katalog.yaml
```

The developer sees only `app.yaml`. The Motif expansion happens at
`ork doctor init` time — the generated Katalog contains the expanded resources
directly, no runtime import. The developer path has no dependency on Motif
resolution at reconcile time.

---

## Motif vs Katalog vs Komposer

| | Motif | Katalog | Komposer |
|---|---|---|---|
| **Kind** | `Motif` | `Katalog` | `Komposer` |
| **Declares CRDs** | No | Yes | No |
| **Runs alone** | No | Yes | Via Katalogs |
| **Has inputs** | Yes (`inputs:`) | Via CRD spec | Via Katalog params |
| **Distributed as** | OCI artifact | OCI artifact / file | File / OCI |
| **Consumer** | Katalog authors | `ork run` | `ork run` |
| **Purpose** | Reusable primitive | Operator declaration | Platform declaration |

---

## Distribution — OCI as the packaging primitive

Motifs are distributed as OCI artifacts, the same as Katalog patterns in the
registry. The same `ork registry push/pull` commands work for both. The media
type distinguishes them:

```
application/vnd.orkestra.katalog.v1    ← Katalog
application/vnd.orkestra.motif.v1      ← Motif
```

`ork registry list` shows both. `ork registry list --type motif` filters to
Motifs. `ork registry info postgres:v16` shows whether the artifact is a Motif
or a Katalog and displays its inputs.

The `orkestra-services` repository contains Motifs. The `orkestra-registry`
repository contains Katalogs — some of which import Motifs from
`orkestra-services`.

---

## Validation

`ork validate -f katalog.yaml` validates all imported Motifs at static analysis
time:

- All required inputs in each `with:` block are provided
- All provided inputs are declared in the Motif schema
- Template expressions in `with:` values are syntactically valid
- No circular imports

Validation errors surface before any cluster interaction. A Katalog with a
missing required input fails `ork validate` with a clear error:

```
Error: import 'postgres:v16' is missing required input 'passwordSecretName'
  at: spec.crds.payments.operatorBox.imports[0]
  Motif requires: image, passwordSecretName
  Provided: image
  Missing: passwordSecretName
```

---

## The composition story

```
ork doctor reads docker-compose.yaml
  ↓ detects postgres:16
  ↓ fetches Motif from orkestra-services/postgres:v16
  ↓ expands inputs at generation time
  ↓ produces .orkestra/katalog.yaml with inline resources
  ↓ produces app.yaml with developer-facing keys
  Developer touches only app.yaml

Platform engineer writes PostgresCluster Katalog
  ↓ imports Motif from orkestra-services/postgres:v16
  ↓ binds spec.* fields to inputs.* via with:
  ↓ adds backup, monitoring, deletion protection on top
  ↓ publishes to orkestra-registry
  Other engineers pull and use PostgresCluster CRD

Community improves postgres Motif
  ↓ better health checks, shutdown handling
  ↓ both consumers benefit automatically
  ↓ developer path: re-run ork doctor init to regenerate
  ↓ platform path: update Motif version in with: block
```

One implementation. Every consumer benefits. The Motif is implemented once.