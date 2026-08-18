# metadata

The `metadata` block identifies and describes a Katalog. Most fields are used by the registry, the Control Center, and `ork inspect` — they have no effect on reconcile behaviour.

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: database-operator          # required
  namespace: platform              # optional — tenant scoping
  clusterName: prod-eu-west-1      # optional — cluster-level filtering
  author: platform-team            # optional
  version: 1.2.0                   # optional — semver
  description: >
    Manages PostgreSQL databases on RDS.
  license: Apache-2.0              # optional
  tags:
    - database
    - stateful
    - aws
  createdBy: operator              # optional — affects Control Center UI
```

---

## Fields

### `name` (required)

Unique identifier for the Katalog. Written as the `managed-by` annotation on every CR the operator manages. Used as the primary key in the registry.

### `namespace`

Scopes this Katalog to a logical tenant or team within a single runtime. Defaults to `"default"` — identical to Kubernetes namespace semantics.

The Control Center groups CRDs by namespace so each team sees only its own panel. Has no effect on which Kubernetes namespaces the operator can act in — that is `spec.crds.<name>.allowedNamespaces`.

### `clusterName`

Identifies the cluster this Katalog runs in. Used by the Control Center for cluster-level filtering when multiple runtimes are connected to a single Control Center. The Katalog value takes precedence over the `CLUSTER_NAME` environment variable. Empty when neither is set.

### `author`

Creator or maintainer. Displayed in `ork inspect`, `ork patterns`, and the Control Center.

### `version`

Semantic version of this Katalog (e.g. `1.2.0`). Used by the registry to version artifacts. `ork push` uses this as the artifact tag.

### `description`

Human-readable explanation of the Katalog's purpose. Shown in `ork inspect` and the Control Center. Supports multi-line YAML block scalars.

### `license`

SPDX license identifier (e.g. `Apache-2.0`, `MIT`). Displayed in registry listings and Artifact Hub indexing.

### `tags`

Keywords for categorising the Katalog in the Orkestra Registry. Aid discovery via `ork patterns --tag <tag>` and indexing in Artifact Hub. Have no effect on runtime behaviour.


## projects

Internal field injected by `ork-doctor` at generation time. Holds developer-side metadata for persona-aware tooling and the Control Center. The operator and runtime ignore it.

---

## Where to go next

- [01-top-level.md](../01-top-level.md) — full Katalog wire format
- [ork inspect](../../../cli/11-inspect.md) — displays metadata fields
- [ork push](../../../cli/09-push.md) — uses `name` and `version` as the artifact tag
