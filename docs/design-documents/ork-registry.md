# ork registry — Design

*April 2026*

---

## One principle

The registry CLI follows the same principle as the rest of Orkestra:
the declarative artifact (the Katalog) is the source of truth.
The CLI moves artifacts between local directories and OCI registries.
It does not manage state, does not require a server, and does not
maintain a registry of registries.

---

## The artifact format

A pattern is a directory containing exactly five files:

```
postgres/
  katalog.yaml     ← the operator declaration (required)
  pattern.yaml     ← registry metadata: name, version, description, tags
  README.md        ← human documentation
  cr.yaml          ← example CR for ork init and testing
  crd.yaml         ← CRD schema (required for typed patterns)
```

This is an OCI artifact with `application/vnd.orkestra.pattern.v1` media type.
Every file is a layer. The manifest carries the pattern metadata as annotations.

---

## Commands

```bash
ork registry push <name>:<version> <dir>
ork registry pull <name>:<version>
ork registry info <name>:<version>
ork registry list [registry-url]
```

No login command — credentials come from `~/.docker/config.json`.
No search command — deferred to Artifact Hub integration in v1.1.

---

## Reference resolution

A bare name resolves against the configured registry:

```bash
# These are equivalent:
ork registry pull postgres:v14
ork registry pull oci://ghcr.io/orkspace/orkestra-registry/postgres:v14

# Override registry:
export ORKESTRA_REGISTRY=oci://myregistry.internal/patterns
ork registry pull postgres:v14
# → pulls from oci://myregistry.internal/patterns/postgres:v14
```

Resolution order:
1. Full OCI reference (starts with `oci://`) — used as-is
2. `ORKESTRA_REGISTRY` env var + `/name:version`
3. Default: `oci://ghcr.io/orkspace/orkestra-registry/name:version`

---

## Authentication

Reads `~/.docker/config.json` directly via ORAS.
Respects `DOCKER_CONFIG` env var.
No separate `ork registry login` — `docker login ghcr.io` is sufficient.

For CI:
```bash
echo "$GITHUB_TOKEN" | docker login ghcr.io -u $GITHUB_ACTOR --password-stdin
ork registry push postgres:v14 ./postgres/
```

---

## Local cache

Pulled patterns are cached in `~/.orkestra/registry/`:

```
~/.orkestra/registry/
  ghcr.io/
    orkspace/
      orkestra-registry/
        postgres/
          v14/
            katalog.yaml
            pattern.yaml
            README.md
            cr.yaml
            crd.yaml
```

`ork run` and Komposer pull from cache first, registry second.
`ork registry pull --refresh` bypasses cache.

---

## Commands in detail

### ork registry push

```bash
ork registry push <name>:<version> <dir>

# Examples:
ork registry push postgres:v14 ./postgres/
ork registry push mycompany/payments-operator:v1.2.0 ./payments/

# Push to a specific registry:
ORKESTRA_REGISTRY=oci://myregistry.internal/patterns \
  ork registry push postgres:v14 ./postgres/
```

Validates the directory structure before pushing:
- `katalog.yaml` and `pattern.yaml` must exist and be valid
- `ork validate -k katalog.yaml` is run automatically
- Fails fast with a clear error if validation fails

Pushes atomically — all files or none.

Output:
```
Validating pattern...
  ✓ katalog.yaml valid
  ✓ pattern.yaml valid
  ✓ crd.yaml present

Pushing postgres:v14 to ghcr.io/orkspace/orkestra-registry...
  ✓ katalog.yaml    (4.2 KB)
  ✓ pattern.yaml    (0.8 KB)
  ✓ README.md       (12.1 KB)
  ✓ cr.yaml         (0.6 KB)
  ✓ crd.yaml        (8.3 KB)

✓ Pushed: oci://ghcr.io/orkspace/orkestra-registry/postgres:v14
  Digest: sha256:a3f5c2b1...

Reference this pattern in a Komposer:
  sources:
    registry:
      - url: oci://ghcr.io/orkspace/orkestra-registry/postgres:v14
```

### ork registry pull

```bash
ork registry pull <name>:<version>

# Examples:
ork registry pull postgres:v14
ork registry pull oci://ghcr.io/myorg/patterns/redis:v7
ork registry pull postgres:v14 --refresh        # bypass cache
ork registry pull postgres:v14 --out ./postgres/ # extract to directory
```

Output:
```
Pulling postgres:v14...
  → oci://ghcr.io/orkspace/orkestra-registry/postgres:v14
  ✓ Cached at ~/.orkestra/registry/ghcr.io/orkspace/orkestra-registry/postgres/v14/

To use this pattern:
  ork run -k ~/.orkestra/registry/.../postgres/v14/katalog.yaml

Or reference in a Komposer:
  sources:
    registry:
      - url: oci://ghcr.io/orkspace/orkestra-registry/postgres:v14
```

### ork registry info

```bash
ork registry info <name>:<version>

# Examples:
ork registry info postgres:v14
ork registry info oci://ghcr.io/myorg/patterns/redis:v7
```

Output:
```
postgres:v14
  Registry:     ghcr.io/orkspace/orkestra-registry
  Digest:       sha256:a3f5c2b1...
  Pushed:       2026-03-15T10:23:00Z
  Size:         26.0 KB

  Description:  Production-ready PostgreSQL operator with automatic backups,
                connection pooling, and cross-namespace secret distribution.

  Tags:         database, postgresql, stateful, backup
  Author:       orkspace
  License:      Apache-2.0

  CRDs:
    postgres.data.orkestra.io/v1alpha1

  Requires:
    - AWS provider (for S3 backup storage)

  Changelog:
    v14: Added PodDisruptionBudget, improved backup scheduling
    v13: Initial stable release

To pull:
  ork registry pull postgres:v14

To use in a Komposer:
  sources:
    registry:
      - url: oci://ghcr.io/orkspace/orkestra-registry/postgres:v14
```

### ork registry list

```bash
ork registry list [registry-url]

# Examples:
ork registry list
ork registry list oci://ghcr.io/mycompany/patterns
ork registry list --tag database
```

Reads an index from the registry root:
`oci://ghcr.io/orkspace/orkestra-registry/index:latest`

The index is a JSON manifest listing all available patterns.
Registry operators generate it with `ork registry reindex` (operator-only command).

Output:
```
Orkestra Registry  (ghcr.io/orkspace/orkestra-registry)
─────────────────────────────────────────────────────────
NAME                     LATEST    TAGS                    DESCRIPTION
postgres                 v14       database, stateful      PostgreSQL with backups
redis                    v7        cache, stateful         Redis cluster operator
mongodb                  v6        database, stateful      MongoDB replica set
cert-manager-bridge      v1.2.0    security, certificates  cert-manager integration
namespace-provisioner    v2.0.0    platform, rbac          Namespace lifecycle management
secret-distributor       v1.1.0    secrets, security       Cross-namespace secret sync

6 patterns  ·  ghcr.io/orkspace/orkestra-registry  ·  Updated 2h ago

To pull a pattern:
  ork registry pull postgres:v14

To use ORKESTRA_REGISTRY for a different registry:
  export ORKESTRA_REGISTRY=oci://myregistry.internal/patterns
```

---

## pattern.yaml schema

```yaml
name: postgres
version: v14
description: >
  Production-ready PostgreSQL operator with automatic backups,
  connection pooling, and cross-namespace secret distribution.
author: orkspace
license: Apache-2.0
tags:
  - database
  - postgresql
  - stateful
  - backup
requires:
  providers:
    - aws      # for S3 backup storage
changelog:
  - version: v14
    notes: Added PodDisruptionBudget, improved backup scheduling
  - version: v13
    notes: Initial stable release
```

---

## Contribution model

Publishing to the official registry (`ghcr.io/orkspace/orkestra-registry`):

1. Fork `github.com/orkspace/orkestra-registry`
2. Add your pattern directory under `patterns/`
3. Include all five files
4. Open a PR — CI validates the pattern against a live kind cluster
5. On merge, CI pushes to GHCR automatically

Publishing to your own registry (no PR required):

```bash
export ORKESTRA_REGISTRY=oci://ghcr.io/mycompany/patterns
ork registry push my-operator:v1.0.0 ./my-operator/
```

Any registry URL works. Teams can run private registries for internal patterns.

---

## Komposer integration (consumer path)

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: platform
sources:
  registry:
    - url: oci://ghcr.io/orkspace/orkestra-registry/postgres:v14
    - url: oci://ghcr.io/mycompany/patterns/payments-operator:v2.1.0
  files:
    - ./katalogs/platform.yaml
```

```bash
ork run --file komposer.yaml
```

The runtime pulls missing patterns to cache, validates them, and starts the
operatorboxes. The user never calls `ork registry pull` manually.

---

## v1 scope

Ships in v1:
- `push`, `pull`, `info`, `list`
- ORAS-based OCI implementation
- `~/.docker/config.json` auth
- Local cache at `~/.orkestra/registry/`
- `ORKESTRA_REGISTRY` env var
- Komposer auto-pull

Deferred to v1.1:
- `ork registry search` (Artifact Hub integration)
- `ork registry reindex` (operator command for index generation)
- Registry mirroring
- Signed artifact verification (`cosign` integration)
- `ork registry diff <name>:<v1> <name>:<v2>`