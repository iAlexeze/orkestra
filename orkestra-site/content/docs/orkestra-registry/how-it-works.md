---
title: "How It Works"
weight: 51
---

# How OrkestraRegistry Works

OrkestraRegistry has a layered architecture. Each layer has a distinct
responsibility. Together they take you from raw resource implementations
to a composable, versioned, ecosystem-scale operator library.

---

## Layer 1 — Internal resource library

The foundation lives inside the Orkestra codebase at `pkg/orkestra-registry/`.
It is a Go library of create, update, delete, and resolve functions for every
common Kubernetes resource type.

```
pkg/orkestra-registry/
  deployments/        Deployment create/update/delete/resolve
  services/           Service create/update/delete/resolve
  secrets/            Secret create/update/delete + cross-namespace copy
  configmaps/         ConfigMap create/update/delete + merge pattern
  serviceaccounts/    ServiceAccount create/delete
  jobs/               Job create/delete (used by onDelete)
  cronjobs/           CronJob create/update/delete/resolve
  pods/               Pod create/update/delete
  template/           Resolver — evaluates template expressions
```

When a Katalog declares:

```yaml
onCreate:
  deployments:
    - image: "{{ .spec.image }}"
      replicas: "{{ .spec.replicas }}"
      reconcile: true
```

The GenericReconciler resolves the template expressions against the live CR,
then calls `orkdeploy.Create`. The registry function handles the Kubernetes
API call, sets owner references, applies system labels, and returns. The
reconciler never constructs a Kubernetes object directly.

{{< callout type="note" >}}
This layer is stable and maintained as part of Orkestra core. It evolves
when new resource types are added. If you need a resource type not yet
supported, see [Adding a Resource Type](https://github.com/ialexeze/orkestra/tree/main/pkg/orkestra-registry/CONTRIBUTING.md/#adding-a-new-resource-type).
{{< /callout >}}

---

## Layer 2 — Public Katalog registry

The public registry is a Git repository (`orkestra-sh/registry`)
that holds the source of truth for community-maintained operator patterns.

### Repository structure

```
orkestra-core/
  postgres/
    v14/
      crd.yaml          the CRD to install
      katalog.yaml      operator behavior — reconcile templates, conversion rules
      komposer.yaml     example import showing how to consume this pattern
      cr.yaml           example custom resource to test with
      README.md         what this pattern does, fields, overrides
  redis/
    v7/
      ...
  monitoring/
    v0.1/
      ...

typed-extensions/
  hooks/
    postgres-hooks/
      v1.0.0/
        go.mod
        hooks.go        implements OnCreate, OnReconcile, OnDelete
        README.md
  constructors/
    ...
```

:::tip[standard-layout]
Every pattern in `orkestra-core/` contains exactly five files. This is
the standard. A pattern is not ready for the registry until it has all five.
:::

### Core Katalog patterns

A core Katalog is a complete, production-ready operator pattern expressed
entirely in YAML. It requires no Go code from the consumer. The Katalog
declares the CRDs, the reconcile templates, and the conversion rules. The
consumer imports it, overrides what they need, and runs it.

```yaml
# orkestra-core/postgres/v14/katalog.yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: postgres-v14
  description: PostgreSQL 14 operator — declarative, production-hardened

spec:
  crds:
    postgres:
      apiTypes:
        group: postgres.orkestra.io
        version: v1
        kind: Postgres
        plural: postgreses
      workers: 2
      resync: 1m
      reconciler:
        default: true
        onCreate:
          deployments:
            - image: "postgres:{{ .spec.version | default \"14\" }}"
              replicas: "1"
              reconcile: true
          services:
            - port: "5432"
              targetPort: "5432"
              reconcile: true
          secrets:
            - name: "{{ .metadata.name }}-credentials"
              data:
                POSTGRES_DB: "{{ .spec.database }}"
                POSTGRES_USER: "{{ .spec.user }}"
```

### Typed extensions

When a use case cannot yet be expressed declaratively — for example, creating
a database user inside PostgreSQL on CR creation — a typed extension provides
the answer. These are Go hooks versioned as independent modules.

```go
// typed-extensions/hooks/postgres-hooks/v1.0.0/hooks.go
func PostgresHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*postgresv1.Postgres]{
        OnCreate: func(ctx context.Context, obj *postgresv1.Postgres) error {
            // create the actual database user inside postgres
            return createDatabaseUser(ctx, obj)
        },
    }
}
```

Consumers reference the typed extension in their Katalog:

```yaml
reconciler:
  default: true
  hooks:
    location: github.com/orkestra-sh/registry/typed-extensions/hooks/postgres-hooks
    version: v1.0.0
    function: PostgresHooks
```

{{< callout type="note" title="The promotion path" >}}
Typed extensions are not permanent. When the declarative model expands
to cover a hook's use case — or when a hook becomes widely used enough
to warrant built-in support — it is promoted. The typed extension is
deprecated with a pointer to the new declarative pattern. The hook code
remains available but the README marks it superseded.
{{< /callout >}}

---

## Layer 3 — OCI distribution

All patterns in `orkestra-core/` are automatically published as OCI artifacts
by a CI pipeline in the registry repository.

### Artifact structure

Each version directory becomes one OCI artifact. The artifact contains the
five standard files with the directory structure preserved:

```
postgres:v14
  ├── crd.yaml
  ├── katalog.yaml
  ├── komposer.yaml
  ├── cr.yaml
  └── README.md
```

Tags follow the pattern `<name>:<version>`. The `latest` tag points to the
highest stable semantic version of each pattern.

### Publishing pipeline

When a commit lands on the main branch of the registry repository, the CI
pipeline:

1. Detects which version directories changed
2. Validates each changed directory contains the five required files
3. Runs `orks validate` against the Katalog
4. Packs the directory into an OCI artifact using ORAS
5. Pushes to `ghcr.io/orkestra-sh/registry/<name>:<version>`
6. Updates the Artifact Hub metadata index

{{< callout type="caution" >}}
Version tags are immutable. Once `postgres:v14.0.0` is pushed, it cannot
be overwritten. This guarantees that pinned version references in
Komposers never change behaviour unexpectedly. Patch releases create new
tags (`postgres:v14.0.1`) — they do not overwrite the previous tag.
{{< /callout >}}

---

## How Orkestra resolves patterns at runtime

When `ork run` reads a Komposer with OCI references, the resolution sequence is:

1. Parse all `sources.oci` entries
2. Check the local cache at `~/.orkestra/registry/` for each artifact
3. Pull uncached artifacts from the OCI registry
4. Unpack each artifact into a temporary directory
5. Parse the `katalog.yaml` from each artifact as a source file
6. Merge all sources following standard Komposer merge rules
7. Apply inline `spec.crds` overrides last

The merged, validated configuration is then used to start the runtime —
exactly as if all the sources had been local files.

```
registry:
  url: ghcr.io/orkestra-sh/registry/postgres:v14
  oci: true
    │
    ▼
~/.orkestra/registry/cache/postgres-v14/
    │
    ▼
katalog.yaml parsed as Katalog source
    │
    ▼
Merger — merged with other sources, inline overrides applied
    │
    ▼
Orkestra runtime starts with merged configuration
```

{{< callout type="tip" >}}
The local cache means Orkestra works offline after the first pull. This
is intentional — operator startups in production should not depend on
OCI registry availability.
{{< /callout >}}

---

## The ork registry CLI

The `ork registry` subcommand provides a dedicated interface for working with
OCI patterns.

{{< callout type="warning" title="In development" >}}
The `ork registry` CLI commands are currently in active development and
have not yet reached stable. The interface described below reflects the
planned API. OCI artifacts can be consumed today via direct `oci`
references in Komposers using ORAS or standard OCI tooling.
{{< /callout >}}

**Planned commands:**

```bash
# Authenticate with an OCI registry
ork registry login ghcr.io

# Publish a pattern
ork registry push ghcr.io/myorg/my-pattern:v1 ./my-pattern/v1

# Pull a pattern to inspect locally
ork registry pull ghcr.io/myorg/my-pattern:v1 ./my-pattern

# List patterns in a registry
ork registry list ghcr.io/orkestra-sh/registry

# Search for a pattern by keyword
ork registry search postgres

# Show pattern metadata without pulling files
ork registry info ghcr.io/orkestra-sh/registry/postgres:v14
```

Updates will be published in the changelog when these commands reach stable.

---

<!-- **Next:** [The Vision →](./vision.md) -->
