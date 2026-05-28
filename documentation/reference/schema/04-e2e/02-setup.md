# Setup

`spec.setup` applies prerequisite resources to the cluster after the CRD is installed but before the Katalog operator starts and the CR is applied. Use it to create Secrets, ConfigMaps, Namespaces, or any other resources the operator needs.

`setup` is optional. Omit it when the test has no prerequisites.

---

## Shorthand

A plain list of strings is equivalent to `setup.apply`:

```yaml
setup:
  - ./prereqs/secret.yaml
  - ./prereqs/namespace.yaml
```

---

## Full form

```yaml
setup:
  apply:
    - ./prereqs/namespace.yaml
    - ./prereqs/pull-secret.yaml
  helm:
    - repo: https://charts.cert-manager.io
      chart: cert-manager
      version: v1.14.0
      namespace: cert-manager
      createNamespace: true
  wait:
    - kind: Deployment
      name: cert-manager
      namespace: cert-manager
      ready: true
      timeout: 120s
```

Execution order: **apply → helm → wait**. Each phase must complete before the next starts.

---

## `setup.apply`

An ordered list of YAML file paths to `kubectl apply`. Files are applied in declaration order. Each file may contain multiple YAML documents separated by `---`.

```yaml
setup:
  apply:
    - ./prereqs/namespace.yaml
    - ./prereqs/pull-secret.yaml
    - ./prereqs/seed-data.yaml
```

Runs after the CRD is installed, before the operator starts watching.

---

## `setup.helm`

An ordered list of Helm charts to install as real releases into the cluster. Runs after `apply`. Each entry uses `helm upgrade --install` — idempotent and safe to re-run.

```yaml
setup:
  helm:
    - repo: https://charts.cert-manager.io
      chart: cert-manager
      version: v1.14.0
      namespace: cert-manager
      createNamespace: true
```

| Field | Required | Description |
|-------|----------|-------------|
| `repo` | yes | Helm chart repository URL |
| `chart` | yes | Chart name within the repository |
| `release` | no | Helm release name. Defaults to the chart name. |
| `namespace` | no | Target namespace. Defaults to `default`. |
| `version` | no | Chart version. Omit for latest. |
| `valueFiles` | no | Ordered list of values files (local paths). |
| `values` | no | Inline key-value overrides (`helm --set`). |
| `createNamespace` | no | Pass `--create-namespace` to helm. |

---

## `setup.wait`

An ordered list of resources to wait for after `apply` and `helm` complete. The test blocks until every listed resource exists and satisfies the `ready` constraint. If any wait times out, setup fails and the CR is never applied.

```yaml
setup:
  wait:
    - kind: Secret
      name: image-pull-secret
      namespace: default
      timeout: 10s
    - kind: Deployment
      name: cert-manager
      namespace: cert-manager
      ready: true
      timeout: 120s
```

| Field | Required | Description |
|-------|----------|-------------|
| `kind` | yes | Kubernetes resource kind |
| `name` | yes | Exact resource name |
| `namespace` | no | Namespace. Omit for cluster-scoped resources. |
| `ready` | no | When `true`, waits for a ready/available condition, not just existence. |
| `timeout` | no | Per-resource timeout. Default: `30s`. |

---

## Real example — 03-secret-copy

```yaml
setup:
  apply:
    - ./setup.yaml   # creates platform/team-alpha/team-beta namespaces + source Secret
  wait:
    - kind: Secret
      name: database-credentials
      namespace: platform
      timeout: 15s   # source Secret must exist before the operator starts
```

The operator copies the Secret from `platform` into team namespaces. Without the `wait`, a race could let the operator start before the source Secret is present.

→ This pattern is from the beginner pack: `ork init --pack beginner`

---

→ Next: [03-expect.md](03-expect.md) — assertion checkpoints
