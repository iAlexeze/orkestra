# How OrkestraRegistry Works

Three layers — each with a distinct role.

---

## Layer 1 — Internal resource library

`pkg/orkestra-registry/` is the Go library that executes what your Katalog declares. When you write `deployments:` in an operatorBox, this is what runs. It handles the Kubernetes API call, sets owner references, and applies system labels.

Supported types: Deployment, Service, Secret, ConfigMap, StatefulSet, Job, CronJob, ReplicaSet, Pod, Ingress, HPA, PDB, PVC, PV, Namespace, Role, RoleBinding, ServiceAccount, and any CRD via the dynamic client.

Template expressions in resource fields have access to the full [Orkestra Notes](../notes/index.md) library — cron parsing, semver, string manipulation, math, conditionals, Kubernetes object introspection, and more.

→ [Contributing a new resource type](../../contributing/contributing-registry.md)

---

## Layer 2 — Public registry

A Git repository (`orkspace/orkestra-registry`) of community-maintained patterns and motifs, published as OCI artifacts.

**Patterns** live under `patterns/katalogs/`. A pattern is a Katalog that declares how a CRD should reconcile. The only required file is `katalog.yaml`. Optional files: `crd.yaml`, `cr.yaml`, `README.md`, `e2e.yaml`.

**Motifs** live under `patterns/motifs/`. A motif declares named inputs and resource templates that get merged into any operatorBox that imports them. Required file: `motif.yaml`. Optional: `README.md`.

**Typed extensions** are Go hooks for use cases that cannot yet be expressed in YAML. When a hook's use case becomes expressible declaratively, it is promoted to a pattern and deprecated.

---

## Layer 3 — OCI distribution

Patterns and motifs are published as OCI artifacts. Version tags are immutable — `postgres:v14.0.0` once pushed cannot be overwritten. The local cache at `~/.orkestra/registry/<host>/<repo>/<tag>/` means `ork run` works offline after the first pull.

Discovery uses a managed index at `registry/index:latest` — `ork registry push` updates it automatically.

---

## Importing a pattern

```yaml
imports:
  registry:
    # Short name — resolves to ghcr.io/orkspace/orkestra-registry/patterns/katalogs/postgres:v14
    - url: postgres:v14

    # Explicit OCI reference
    - url: ghcr.io/orkspace/orkestra-registry/patterns/katalogs/redis
      version: v7
      oci: true

    # @ shorthand — version inline
    - url: ghcr.io/myorg/patterns/redis@v7
      oci: true

    # Load komposer.yaml from the pulled artifact instead of katalog.yaml
    - url: ghcr.io/myorg/my-pattern:v2
      oci: true
      useKomposer: true

    # Git source
    - url: https://github.com/myorg/internal-patterns
      version: main
```

Motifs are imported at the CRD level in a Katalog — alongside `operatorBox`, not inside it. A motif contributes resources, status configuration, and admission rules (validation and mutation) to the CRD that imports it.

```yaml
spec:
  crds:
    myresource:
      imports:
        - motif: ghcr.io/orkspace/orkestra-registry/patterns/motifs/postgres-pdb@v1
          with:
            maxUnavailable: "1"
        # Short name — resolves to the default motif registry
        - motif: postgres-pdb:v1
          with:
            maxUnavailable: "1"
      operatorBox:
        ...
```

---

## CLI

```bash
# Cache a pattern locally
ork registry pull postgres:v14
ork registry pull ghcr.io/myorg/patterns/redis:v7

# Pull all OCI refs declared in a file
ork registry pull -f katalog.yaml
ork registry pull -f komposer.yaml

# Force re-pull, bypassing cache
ork registry pull postgres:v14 --refresh

# Inspect metadata without downloading files
ork registry info postgres:v14

# List patterns in the default registry
ork registry list
ork registry list --motifs        # motifs only
ork registry list --katalogs      # patterns only
ork registry list ghcr.io/myorg/patterns

# Publish a pattern (runs e2e.yaml gate if present)
ork registry push postgres:v14 ./patterns/postgres/v14
ork registry push --no-e2e postgres:v14 ./patterns/postgres/v14
```

Authentication uses `~/.docker/config.json`. Run `docker login ghcr.io` before pushing.

Override the default registries:

```bash
export ORK_REGISTRY=oci://myregistry.internal/patterns/katalogs
export ORK_MOTIFS_REGISTRY=oci://myregistry.internal/patterns/motifs
```
