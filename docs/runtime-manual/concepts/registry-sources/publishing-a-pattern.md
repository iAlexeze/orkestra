# Publishing a Pattern

A registry pattern is a directory with five files that declares how an
Orkestra operator should behave for a specific CRD. This guide covers how
to structure a pattern, validate it locally, and publish it to an OCI or
Git registry for others to consume.

---

## Pattern directory structure

Every pattern must follow this structure exactly:

```
my-operator/
  v1.0.0/
    crd.yaml
    katalog.yaml
    komposer.yaml
    cr.yaml
    README.md
```

Version directories use semantic versioning. A registry repository may
contain multiple versions of the same pattern:

```
postgres/
  v14.0.0/
    crd.yaml
    ...
  v14.1.0/
    crd.yaml
    ...
  v15.0.0/
    crd.yaml
    ...
```

:::warning[Never overwrite a published version]
Once a version directory is pushed and a consumer has pinned to it,
that version is a contract. Create a new version directory for changes.
Never modify a published version's files.
:::

---

## The five required files

### `crd.yaml`

The CRD definition that consumers must install before running the operator.

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: postgreses.postgres.orkestra.io
spec:
  group: postgres.orkestra.io
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                version:
                  type: string
                  default: "14"
                database:
                  type: string
                user:
                  type: string
  names:
    kind: Postgres
    plural: postgreses
    singular: postgres
  scope: Namespaced
```

!!! tip
    Include the OpenAPI v3 schema in the CRD. It enables `kubectl` validation
    at apply time and makes the spec fields discoverable without reading the
    Katalog.

### `katalog.yaml`

The operator behavior. Reconcile templates, conversion rules, workers,
resync interval, and dependency ordering.

```yaml
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
      operatorBox:
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
```

### `komposer.yaml`

An example showing how to import this pattern and what overrides make sense.
This is documentation through example — it should demonstrate the patterns
a real consumer would use.

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: postgres-example
  description: Example Komposer showing how to consume the postgres pattern

sources:
  registry:
    - url: ghcr.io/orkestra-sh/orkestra-registry/postgres@v14.0.0
      oci: true

spec:
  crds:
    # Production overrides — recommended for non-development environments
    postgres:
      workers: 4        # default is 2 — increase for higher throughput
      resync: 30s       # default is 1m — reduce for faster drift detection
```

:::note
The `komposer.yaml` in a published pattern references the pattern
itself — by its published OCI tag or Git URL. This means consumers can
run it directly as a quick-start without writing their own Komposer.
:::

---


### `cr.yaml`

An example custom resource. It must work with zero additional configuration
after installing the CRD and running the operator.

```yaml
apiVersion: postgres.orkestra.io/v1
kind: Postgres
metadata:
  name: my-postgres
  namespace: default
spec:
  version: "14"
  database: myapp
  user: myapp_user
```

:::tip
Use the simplest possible CR that demonstrates a successful reconcile.
Do not require fields that need pre-existing secrets or external systems
to already exist. 
:::

The consumer should be able to run:

```bash
kubectl apply -f crd.yaml           # apply the crd
ork run --katalog komposer.yaml     # run orkestra against the komposer (this is how consumers will use it)

kubectl apply -f cr.yaml            # apply the cr example

kubectl get deployments             # should show the postgres deployment
```

### `README.md`

Documentation. This is what a consumer reads before deciding whether to
adopt your pattern.

A good README covers:

- **What this operator does** — one paragraph, plain language
- **Field reference** — a table of every spec field with type, default, and description
- **Quick start** — the three commands to go from zero to running
- **Recommended production overrides** — which fields to change for non-development
- **Known limitations** — what the pattern does not cover
- **Version history** — what changed between pattern versions

!!! warning
    A pattern with a sparse or empty README is a pattern consumers cannot
    trust. They have no way to know what spec fields are required, what
    defaults are applied, or what the operator will create in their cluster.

---

## Validate locally before publishing

Run `ork validate` against your katalog before publishing. It checks:
- The katalog YAML is valid
- All required fields are present
- Template expressions are well-formed
- The five required files exist and are non-empty (when run from the pattern directory)

```bash
cd my-operator/v1.0.0
ork validate --katalog katalog.yaml
```

Also test with the example CR:

```bash
kubectl apply -f crd.yaml           # apply the crd
ork run --katalog komposer.yaml     # run orkestra against the komposer (this is how consumers will use it)

kubectl apply -f cr.yaml            # apply the cr example

kubectl get deployments             # verify resources were created
kubectl delete -f cr.yaml           # verify cleanup runs correctly
```

---

## Publishing to OCI

Push the version directory as an OCI artifact using ORAS:

```bash
cd my-operator/v1.0.0

# Push all files as a single artifact
oras push ghcr.io/myorg/my-operator:v1.0.0 --artifact-type application/vnd.orkestra.katalog.v1.tar+gzip \
  crd.yaml:application/yaml \
  katalog.yaml:application/yaml \
  komposer.yaml:application/yaml \
  cr.yaml:application/yaml \
  README.md:text/markdown

# Also tag as latest if this is the current stable version
oras tag ghcr.io/myorg/my-operator:v1.0.0 latest
```

> The artifact type is required: --artifact-type application/vnd.orkestra.katalog.v1.tar+gzip

:::tip[Automate publishing with GitHub Actions]
The `orkestra-sh/orkestra-registry` repository includes a reference
CI pipeline that detects changed version directories and publishes them
automatically. Copy the workflow file as a starting point for your own
registry.
:::

---

## Publishing to a Git registry

Push the pattern directory to your Git repository. The structure is all
that is required — no additional packaging step.

```
registry/
  katalogs/
    my-operator/
      v1.0.0/
        crd.yaml
        katalog.yaml
        komposer.yaml
        cr.yaml
        README.md
```

Consumers reference it by the repository URL and a branch, tag, or SHA:

```yaml
sources:
  registry:
    - url: https://github.com/myorg/operator-registry@v1.0.0
      auth:
        type: github
        fromEnv: GITHUB_TOKEN
```

---

## Semantic versioning

Pattern versions follow semantic versioning (`MAJOR.MINOR.PATCH`):

- **Patch** (`v14.0.1`) — bug fixes that do not change field names or defaults
- **Minor** (`v14.1.0`) — new optional fields, new defaults, additive changes
- **Major** (`v15.0.0`) — breaking changes: renamed fields, removed fields,
  changed behavior that may require consumer Komposer updates or version conversion

When you make a breaking change, add a migration note to the README that
explains what consumers need to update in their `spec.crds` overrides.

!!! warning
    A breaking change without a migration note is a breaking change without
    a fix. Consumers will upgrade, see errors, and have no path forward.
    Write the migration note before publishing the new version.
